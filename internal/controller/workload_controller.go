package controller

import (
	"context"
	"fmt"
	"time"

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
)

type WorkloadReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

func (r *WorkloadReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	started := time.Now()
	log := ctrl.LoggerFrom(ctx).WithValues(
		"kind", "Deployment",
		"namespace", req.Namespace,
		"name", req.Name,
	)
	log.Info("reconcile started")

	var dep appsv1.Deployment
	if err := r.Get(ctx, req.NamespacedName, &dep); err != nil {
		if apierrors.IsNotFound(err) {
			log.V(1).Info("workload no longer exists")
			return ctrl.Result{}, nil
		}
		log.Error(err, "failed to read workload")
		return ctrl.Result{}, err
	}

	ann := dep.Spec.Template.Annotations
	if !injection.WantsInjection(ann) {
		log.V(1).Info("injection not requested")
		return ctrl.Result{}, nil
	}

	log.Info("injection requested", "needs", ann[injection.AnnotationNeeds])
	providers, err := injection.ResolveProviders(ctx, r.Client, req.Namespace, ann)
	if err != nil {
		log.Error(err, "failed to resolve resource providers")
		r.event(&dep, corev1.EventTypeWarning, "ProviderResolutionFailed", err.Error())
		return ctrl.Result{}, err
	}
	if len(providers) == 0 {
		log.Info("no matching resource providers found")
		r.event(&dep, corev1.EventTypeWarning, "NoProvidersFound", "No matching ResourceProvider objects were found")
		return ctrl.Result{}, nil
	}

	providerNames := make([]string, 0, len(providers))
	for i := range providers {
		providerNames = append(providerNames, providers[i].Name)
		log.Info("resource provider resolved",
			"provider", providers[i].Name,
			"providerType", providers[i].Spec.Type,
		)

		esList, err := injection.BuildExternalSecrets(&providers[i], dep.Namespace, dep.Name)
		if err != nil {
			log.Error(err, "failed to build ExternalSecret", "provider", providers[i].Name)
			r.event(&dep, corev1.EventTypeWarning, "ExternalSecretBuildFailed", err.Error())
			return ctrl.Result{}, err
		}

		for _, built := range esList {
			action, err := upsertUnstructured(ctx, r.Client, built.Object)
			if err != nil {
				log.Error(err, "failed to apply resource",
					"resourceKind", built.Object.GetKind(),
					"resourceNamespace", built.Object.GetNamespace(),
					"resourceName", built.Object.GetName(),
				)
				r.event(&dep, corev1.EventTypeWarning, "ResourceApplyFailed", err.Error())
				return ctrl.Result{}, err
			}

			log.Info("resource applied",
				"action", action,
				"resourceKind", built.Object.GetKind(),
				"resourceNamespace", built.Object.GetNamespace(),
				"resourceName", built.Object.GetName(),
				"provider", providers[i].Name,
			)
			if action != "unchanged" {
				r.event(&dep, corev1.EventTypeNormal, "ResourceApplied",
					fmt.Sprintf("%s %s/%s %s", built.Object.GetKind(), built.Object.GetNamespace(), built.Object.GetName(), action))
			}
		}
	}

	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion("apps/v1")
	obj.SetKind("Deployment")
	if err := r.Get(ctx, types.NamespacedName{Name: dep.Name, Namespace: dep.Namespace}, obj); err != nil {
		log.Error(err, "failed to reload workload before patch")
		return ctrl.Result{}, err
	}

	oldHash, _, _ := unstructured.NestedString(obj.Object, "spec", "template", "metadata", "annotations", injection.AnnotationHash)
	if err := injection.PatchWorkload(obj, providers); err != nil {
		log.Error(err, "failed to prepare workload patch")
		r.event(&dep, corev1.EventTypeWarning, "InjectionFailed", err.Error())
		return ctrl.Result{}, err
	}
	newHash, _, _ := unstructured.NestedString(obj.Object, "spec", "template", "metadata", "annotations", injection.AnnotationHash)

	if oldHash != newHash {
		if err := r.Update(ctx, obj); err != nil {
			log.Error(err, "failed to patch workload", "oldHash", oldHash, "newHash", newHash)
			r.event(&dep, corev1.EventTypeWarning, "WorkloadPatchFailed", err.Error())
			return ctrl.Result{}, err
		}
		log.Info("workload patched",
			"providers", providerNames,
			"oldHash", oldHash,
			"newHash", newHash,
		)
		r.event(&dep, corev1.EventTypeNormal, "Injected",
			fmt.Sprintf("Injected ResourceProviders: %v", providerNames))
	} else {
		log.V(1).Info("workload already up to date", "hash", newHash, "providers", providerNames)
	}

	log.Info("reconcile completed", "duration", time.Since(started).String())
	return ctrl.Result{}, nil
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

	desired.SetResourceVersion(existing.GetResourceVersion())
	if err := c.Update(ctx, desired); err != nil {
		return "", err
	}
	return "updated", nil
}

func (r *WorkloadReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&appsv1.Deployment{}).Complete(r)
}
