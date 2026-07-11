package controller

import (
	"context"
	"fmt"
	"time"

	resourcesv1alpha1 "github.com/eclipse-xfsc/kubernetes-operator/api/v1alpha1"
	"github.com/eclipse-xfsc/kubernetes-operator/internal/injection"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type WorkloadReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

func (r *WorkloadReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	started := time.Now()
	log := ctrl.LoggerFrom(ctx).WithValues("kind", "Deployment", "namespace", req.Namespace, "name", req.Name)
	var dep appsv1.Deployment
	if err := r.Get(ctx, req.NamespacedName, &dep); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "failed to read consumer")
		return ctrl.Result{}, err
	}

	ann := workloadAnnotations(&dep)
	if !injection.WantsInjection(ann) {
		return ctrl.Result{}, nil
	}

	log.Info("consumer reconcile started")
	log.Info("consumer discovered", "needs", injection.SplitCSV(ann[injection.AnnotationNeeds]), "providers", injection.SplitCSV(ann[injection.AnnotationProviders]))
	providers, err := injection.ResolveProviders(ctx, r.Client, req.Namespace, ann)
	if err != nil {
		log.Error(err, "failed to resolve resource providers")
		r.event(&dep, corev1.EventTypeWarning, "ProviderResolutionFailed", err.Error())
		return ctrl.Result{}, err
	}
	if len(providers) == 0 {
		log.Info("consumer has no matching resource providers")
		r.event(&dep, corev1.EventTypeWarning, "NoProvidersFound", "No matching ResourceProvider objects were found")
		return ctrl.Result{}, nil
	}

	providerNames := make([]string, 0, len(providers))
	for i := range providers {
		providerNames = append(providerNames, providers[i].Name)
		log.Info("producer matched to consumer", "producer", providers[i].Name, "producerNamespace", providers[i].Namespace, "producerType", providers[i].Spec.Type)

		esList, err := injection.BuildExternalSecrets(&providers[i], dep.Namespace, dep.Name)
		if err != nil {
			log.Error(err, "failed to build ExternalSecret", "producer", providers[i].Name)
			return ctrl.Result{}, err
		}
		for _, built := range esList {
			action, err := upsertUnstructured(ctx, r.Client, built.Object)
			if err != nil {
				log.Error(err, "failed to install generated resource", "resourceKind", built.Object.GetKind(), "resourceNamespace", built.Object.GetNamespace(), "resourceName", built.Object.GetName())
				return ctrl.Result{}, err
			}
			switch action {
			case "created":
				log.Info("generated resource created", "resourceKind", built.Object.GetKind(), "resourceNamespace", built.Object.GetNamespace(), "resourceName", built.Object.GetName(), "producer", providers[i].Name, "consumer", dep.Name)
				r.event(&dep, corev1.EventTypeNormal, "ResourceCreated", fmt.Sprintf("Created %s %s/%s", built.Object.GetKind(), built.Object.GetNamespace(), built.Object.GetName()))
			case "updated":
				log.Info("generated resource updated", "resourceKind", built.Object.GetKind(), "resourceNamespace", built.Object.GetNamespace(), "resourceName", built.Object.GetName(), "producer", providers[i].Name, "consumer", dep.Name)
				r.event(&dep, corev1.EventTypeNormal, "ResourceUpdated", fmt.Sprintf("Updated %s %s/%s", built.Object.GetKind(), built.Object.GetNamespace(), built.Object.GetName()))
			default:
				log.V(1).Info("generated resource unchanged", "resourceKind", built.Object.GetKind(), "resourceNamespace", built.Object.GetNamespace(), "resourceName", built.Object.GetName(), "producer", providers[i].Name, "consumer", dep.Name)
			}
		}
	}

	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion("apps/v1")
	obj.SetKind("Deployment")
	if err := r.Get(ctx, req.NamespacedName, obj); err != nil {
		return ctrl.Result{}, err
	}
	oldHash, _, _ := unstructured.NestedString(obj.Object, "spec", "template", "metadata", "annotations", injection.AnnotationHash)
	if err := injection.PatchWorkload(obj, providers); err != nil {
		return ctrl.Result{}, err
	}
	newHash, _, _ := unstructured.NestedString(obj.Object, "spec", "template", "metadata", "annotations", injection.AnnotationHash)

	if oldHash != newHash {
		if err := r.Update(ctx, obj); err != nil {
			log.Error(err, "failed to patch consumer", "oldHash", oldHash, "newHash", newHash)
			return ctrl.Result{}, err
		}
		log.Info("consumer patched", "matchedProducers", providerNames, "oldHash", oldHash, "newHash", newHash)
		r.event(&dep, corev1.EventTypeNormal, "Injected", fmt.Sprintf("Injected ResourceProviders: %v", providerNames))
	} else {
		log.V(1).Info("consumer already up to date", "matchedProducers", providerNames, "hash", newHash)
	}

	log.Info("consumer reconcile completed", "duration", time.Since(started).String())
	return ctrl.Result{}, nil
}

func injectionWatchConfig(dep *appsv1.Deployment) map[string]string {
	ann := workloadAnnotations(dep)
	return map[string]string{
		injection.AnnotationEnabled:   ann[injection.AnnotationEnabled],
		injection.AnnotationNeeds:     ann[injection.AnnotationNeeds],
		injection.AnnotationProviders: ann[injection.AnnotationProviders],
	}
}

func mapsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for key, value := range a {
		if b[key] != value {
			return false
		}
	}
	return true
}

func workloadAnnotations(dep *appsv1.Deployment) map[string]string {
	out := map[string]string{}
	for k, v := range dep.Annotations {
		out[k] = v
	}
	for k, v := range dep.Spec.Template.Annotations {
		out[k] = v
	}
	return out
}

func (r *WorkloadReconciler) mapProviderToConsumers(ctx context.Context, obj client.Object) []reconcile.Request {
	provider, ok := obj.(*resourcesv1alpha1.ResourceProvider)
	if !ok {
		return nil
	}
	log := ctrl.LoggerFrom(ctx).WithValues("producer", provider.Name, "producerNamespace", provider.Namespace, "producerType", provider.Spec.Type)
	var deployments appsv1.DeploymentList
	if err := r.List(ctx, &deployments); err != nil {
		log.Error(err, "failed to list consumers for provider event")
		return nil
	}
	requests := make([]reconcile.Request, 0)
	for i := range deployments.Items {
		ann := workloadAnnotations(&deployments.Items[i])
		if !injection.WantsInjection(ann) {
			continue
		}
		requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{Namespace: deployments.Items[i].Namespace, Name: deployments.Items[i].Name}})
	}
	log.Info("resource provider event queued consumers", "consumerCount", len(requests))
	return requests
}

func (r *WorkloadReconciler) event(obj runtime.Object, eventType, reason, message string) {
	if r.Recorder != nil {
		r.Recorder.Event(obj, eventType, reason, message)
	}
}

func upsertUnstructured(ctx context.Context, c client.Client, desired *unstructured.Unstructured) (string, error) {
	var existing unstructured.Unstructured
	existing.SetAPIVersion(desired.GetAPIVersion())
	existing.SetKind(desired.GetKind())
	key := types.NamespacedName{Name: desired.GetName(), Namespace: desired.GetNamespace()}
	if err := c.Get(ctx, key, &existing); apierrors.IsNotFound(err) {
		if err := c.Create(ctx, desired); err != nil {
			return "", err
		}
		return "created", nil
	} else if err != nil {
		return "", err
	}
	if fmt.Sprintf("%v", existing.Object["spec"]) == fmt.Sprintf("%v", desired.Object["spec"]) {
		return "unchanged", nil
	}
	desired.SetResourceVersion(existing.GetResourceVersion())
	if err := c.Update(ctx, desired); err != nil {
		return "", err
	}
	return "updated", nil
}

func (r *WorkloadReconciler) SetupWithManager(mgr ctrl.Manager) error {
	consumerPredicate := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			dep, ok := e.Object.(*appsv1.Deployment)
			if !ok || !injection.WantsInjection(workloadAnnotations(dep)) {
				return false
			}
			ctrl.Log.WithName("watch").Info("consumer created", "namespace", dep.Namespace, "name", dep.Name, "needs", injection.SplitCSV(workloadAnnotations(dep)[injection.AnnotationNeeds]))
			return true
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldDep, oldOK := e.ObjectOld.(*appsv1.Deployment)
			newDep, newOK := e.ObjectNew.(*appsv1.Deployment)
			if !oldOK || !newOK {
				return false
			}
			oldEnabled := injection.WantsInjection(workloadAnnotations(oldDep))
			newEnabled := injection.WantsInjection(workloadAnnotations(newDep))
			if !oldEnabled && !newEnabled {
				return false
			}

			// Ignore status-only and metadata noise. Reconcile only when the pod
			// template/spec or the injection annotations actually changed.
			specChanged := oldDep.Generation != newDep.Generation
			annotationsChanged := !mapsEqual(injectionWatchConfig(oldDep), injectionWatchConfig(newDep))
			if !specChanged && !annotationsChanged {
				return false
			}

			action := "updated"
			if oldEnabled && !newEnabled {
				action = "injection-disabled"
			} else if !oldEnabled && newEnabled {
				action = "injection-enabled"
			}
			ctrl.Log.WithName("watch").Info("consumer changed", "action", action, "namespace", newDep.Namespace, "name", newDep.Name, "generation", newDep.Generation)
			return newEnabled
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			dep, ok := e.Object.(*appsv1.Deployment)
			if !ok || !injection.WantsInjection(workloadAnnotations(dep)) {
				return false
			}
			ctrl.Log.WithName("watch").Info("consumer deleted", "namespace", dep.Namespace, "name", dep.Name)
			// The object is gone; there is nothing useful for Reconcile to read.
			return false
		},
		GenericFunc: func(event.GenericEvent) bool { return false },
	}

	providerPredicate := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			p, ok := e.Object.(*resourcesv1alpha1.ResourceProvider)
			if ok {
				ctrl.Log.WithName("watch").Info("resource provider created", "namespace", p.Namespace, "name", p.Name, "type", p.Spec.Type)
			}
			return ok
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldP, oldOK := e.ObjectOld.(*resourcesv1alpha1.ResourceProvider)
			newP, newOK := e.ObjectNew.(*resourcesv1alpha1.ResourceProvider)
			if !oldOK || !newOK || oldP.Generation == newP.Generation {
				return false
			}
			ctrl.Log.WithName("watch").Info("resource provider updated", "namespace", newP.Namespace, "name", newP.Name, "type", newP.Spec.Type, "generation", newP.Generation)
			return true
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			p, ok := e.Object.(*resourcesv1alpha1.ResourceProvider)
			if ok {
				ctrl.Log.WithName("watch").Info("resource provider deleted", "namespace", p.Namespace, "name", p.Name, "type", p.Spec.Type)
			}
			return ok
		},
		GenericFunc: func(event.GenericEvent) bool { return false },
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.Deployment{}, builder.WithPredicates(consumerPredicate)).
		Watches(
			&resourcesv1alpha1.ResourceProvider{},
			handler.EnqueueRequestsFromMapFunc(r.mapProviderToConsumers),
			builder.WithPredicates(providerPredicate),
		).
		Complete(r)
}
