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
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
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
	log.Info("consumer reconcile started")

	var dep appsv1.Deployment
	if err := r.Get(ctx, req.NamespacedName, &dep); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("consumer removed")
			return ctrl.Result{}, nil
		}
		log.Error(err, "failed to read consumer")
		return ctrl.Result{}, err
	}

	ann := workloadAnnotations(&dep)
	if !injection.WantsInjection(ann) {
		log.Info("deployment discovered but injection is not enabled")
		return ctrl.Result{}, nil
	}

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
			log.Info("generated resource reconciled", "action", action, "resourceKind", built.Object.GetKind(), "resourceNamespace", built.Object.GetNamespace(), "resourceName", built.Object.GetName(), "producer", providers[i].Name, "consumer", dep.Name)
			if action != "unchanged" {
				r.event(&dep, corev1.EventTypeNormal, "ResourceApplied", fmt.Sprintf("%s %s/%s %s", built.Object.GetKind(), built.Object.GetNamespace(), built.Object.GetName(), action))
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
		log.Info("consumer already up to date", "matchedProducers", providerNames, "hash", newHash)
	}

	log.Info("consumer reconcile completed", "duration", time.Since(started).String())
	return ctrl.Result{}, nil
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
	log.Info("resource provider event received; locating consumers")
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
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.Deployment{}).
		Watches(&resourcesv1alpha1.ResourceProvider{}, handler.EnqueueRequestsFromMapFunc(r.mapProviderToConsumers)).
		Complete(r)
}
