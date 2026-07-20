package controller

import (
	"context"
	"fmt"
	"strings"
	"time"

	resourcesv1alpha1 "github.com/eclipse-xfsc/kubernetes-operator/api/v1alpha1"
	"github.com/eclipse-xfsc/kubernetes-operator/internal/injection"
	"github.com/eclipse-xfsc/kubernetes-operator/internal/modules"
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
	Modules  *modules.Registry
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
	log.Info("consumer discovered", "needs", injection.SplitCSV(ann[injection.AnnotationNeeds]), "providers", injection.SplitCSV(ann[injection.AnnotationProviders]), "envPrefix", ann[injection.AnnotationEnvPrefix])
	providers, err := injection.ResolveProviders(ctx, r.Client, req.Namespace, ann)
	if err != nil {
		log.Error(err, "failed to resolve resource providers")
		r.event(&dep, corev1.EventTypeWarning, "ProviderResolutionFailed", err.Error())
		return ctrl.Result{}, err
	}
	missing := missingProviderRequests(ann, providers)
	providerNames := make([]string, 0, len(providers))
	generated := map[string][]string{}

	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion("apps/v1")
	obj.SetKind("Deployment")
	if err := r.Get(ctx, req.NamespacedName, obj); err != nil {
		return ctrl.Result{}, err
	}
	oldState := injection.ReadManagedState(obj)

	for i := range providers {
		providerKey := providers[i].Name
		providerNames = append(providerNames, providers[i].Name)
		log.Info("producer matched to consumer", "producer", providers[i].Name, "producerType", providers[i].Spec.Type)

		moduleResult, err := r.Modules.Reconcile(ctx, modules.Request{
			Client:      r.Client,
			Provider:    providers[i],
			Namespace:   dep.Namespace,
			Workload:    dep.Name,
			Annotations: ann,
		})

		if err != nil {
			log.Error(err, "resource module reconciliation failed", "producer", providers[i].Name, "producerType", providers[i].Spec.Type)
			r.event(&dep, corev1.EventTypeWarning, "ModuleReconcileFailed", err.Error())
			return ctrl.Result{}, err
		}

		for _, moduleResource := range moduleResult.Resources {
			if moduleResource == nil {
				continue
			}
			resourceID := managedResourceID(moduleResource)
			generated[providerKey] = append(generated[providerKey], resourceID)
			action, err := upsertUnstructured(ctx, r.Client, moduleResource)
			if err != nil {
				return ctrl.Result{}, err
			}
			if action == "created" || action == "updated" {
				log.Info("module resource "+action, "resourceKind", moduleResource.GetKind(), "resourceNamespace", moduleResource.GetNamespace(), "resourceName", moduleResource.GetName(), "producer", providers[i].Name)
			}
		}

		esList, err := injection.BuildExternalSecrets(&providers[i], dep.Namespace, dep.Name)
		if err != nil {
			return ctrl.Result{}, err
		}
		for _, built := range esList {
			resourceID := managedResourceID(built.Object)
			generated[providerKey] = append(generated[providerKey], resourceID)
			action, err := upsertUnstructured(ctx, r.Client, built.Object)
			if err != nil {
				return ctrl.Result{}, err
			}
			if action == "created" || action == "updated" {
				log.Info("generated resource "+action, "resourceKind", built.Object.GetKind(), "resourceNamespace", built.Object.GetNamespace(), "resourceName", built.Object.GetName(), "producer", providers[i].Name)
			}
		}
	}

	// Remove generated resources that were owned by a provider but are no longer desired.
	desiredResources := map[string]struct{}{}
	for _, ids := range generated {
		for _, id := range ids {
			desiredResources[id] = struct{}{}
		}
	}
	for providerKey, oldProvider := range oldState.Providers {
		for _, id := range oldProvider.Resources {
			if _, keep := desiredResources[id]; keep {
				continue
			}
			if err := deleteManagedResource(ctx, r.Client, id); err != nil {
				log.Error(err, "failed to delete obsolete generated resource", "provider", providerKey, "resource", id)
				return ctrl.Result{}, err
			}
			log.Info("generated resource deleted", "provider", providerKey, "resource", id)
			r.event(&dep, corev1.EventTypeNormal, "ResourceDeleted", "Deleted obsolete managed resource "+id)
		}
	}

	oldHash, _, _ := unstructured.NestedString(obj.Object, "spec", "template", "metadata", "annotations", injection.AnnotationHash)
	_, err = injection.PatchWorkload(obj, providers, generated, missing)
	if err != nil {
		return ctrl.Result{}, err
	}
	newHash, _, _ := unstructured.NestedString(obj.Object, "spec", "template", "metadata", "annotations", injection.AnnotationHash)
	oldWarning := dep.Annotations[injection.AnnotationWarning]
	newWarning := obj.GetAnnotations()[injection.AnnotationWarning]
	if oldHash != newHash || oldWarning != newWarning || managedStateChanged(oldState, injection.ReadManagedState(obj)) {
		if err := r.Update(ctx, obj); err != nil {
			return ctrl.Result{}, err
		}
		log.Info("consumer patched", "matchedProducers", providerNames, "missingProviders", missing, "oldHash", oldHash, "newHash", newHash)
		if len(missing) > 0 {
			msg := "Required ResourceProviders are unavailable: " + strings.Join(missing, ", ") + ". Managed injections for them were removed."
			r.event(&dep, corev1.EventTypeWarning, "ProviderUnavailable", msg)
			log.Info("consumer warning applied", "missingProviders", missing)
		} else {
			r.event(&dep, corev1.EventTypeNormal, "Injected", fmt.Sprintf("Reconciled ResourceProviders: %v", providerNames))
		}
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
		injection.AnnotationEnvPrefix: ann[injection.AnnotationEnvPrefix],
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
	log := ctrl.LoggerFrom(ctx).WithValues("producer", provider.Name, "producerType", provider.Spec.Type)
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

func missingProviderRequests(ann map[string]string, providers []resourcesv1alpha1.ResourceProvider) []string {
	foundTypes := map[string]bool{}
	foundNames := map[string]bool{}
	for _, p := range providers {
		foundTypes[p.Spec.Type] = true
		foundNames[p.Name] = true
	}
	missing := []string{}
	for _, t := range injection.SplitCSV(ann[injection.AnnotationNeeds]) {
		if !foundTypes[t] {
			missing = append(missing, t)
		}
	}
	for _, n := range injection.SplitCSV(ann[injection.AnnotationProviders]) {
		if !foundNames[n] {
			missing = append(missing, n)
		}
	}
	return missing
}

func managedResourceID(obj *unstructured.Unstructured) string {
	return strings.Join([]string{obj.GetAPIVersion(), obj.GetKind(), obj.GetNamespace(), obj.GetName()}, "|")
}

func deleteManagedResource(ctx context.Context, c client.Client, id string) error {
	parts := strings.Split(id, "|")
	if len(parts) != 4 {
		return fmt.Errorf("invalid managed resource id %q", id)
	}
	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion(parts[0])
	obj.SetKind(parts[1])
	obj.SetNamespace(parts[2])
	obj.SetName(parts[3])
	err := c.Delete(ctx, obj)
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

func managedStateChanged(a, b injection.ManagedState) bool {
	return fmt.Sprintf("%v", a) != fmt.Sprintf("%v", b)
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
				ctrl.Log.WithName("watch").Info("resource provider created", "name", p.Name, "type", p.Spec.Type)
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
				ctrl.Log.WithName("watch").Info("resource provider deleted", "name", p.Name, "type", p.Spec.Type)
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
