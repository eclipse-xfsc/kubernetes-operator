package controller

import (
	"context"

	"github.com/eclipse-xfsc/kubernetes-operator/internal/injection"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type WorkloadReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *WorkloadReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var dep appsv1.Deployment
	if err := r.Get(ctx, req.NamespacedName, &dep); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	ann := dep.Spec.Template.Annotations
	if !injection.WantsInjection(ann) {
		return ctrl.Result{}, nil
	}
	providers, err := injection.ResolveProviders(ctx, r.Client, req.Namespace, ann)
	if err != nil {
		return ctrl.Result{}, err
	}
	if len(providers) == 0 {
		return ctrl.Result{}, nil
	}
	for i := range providers {
		esList, err := injection.BuildExternalSecrets(&providers[i], dep.Namespace, dep.Name)
		if err != nil {
			return ctrl.Result{}, err
		}
		for _, built := range esList {
			if err := upsertUnstructured(ctx, r.Client, built.Object); err != nil {
				return ctrl.Result{}, err
			}
		}
	}
	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion("apps/v1")
	obj.SetKind("Deployment")
	if err := r.Get(ctx, types.NamespacedName{Name: dep.Name, Namespace: dep.Namespace}, obj); err != nil {
		return ctrl.Result{}, err
	}
	oldHash, _, _ := unstructured.NestedString(obj.Object, "spec", "template", "metadata", "annotations", injection.AnnotationHash)
	if err := injection.PatchWorkload(obj, providers); err != nil {
		return ctrl.Result{}, err
	}
	newHash, _, _ := unstructured.NestedString(obj.Object, "spec", "template", "metadata", "annotations", injection.AnnotationHash)
	if oldHash != newHash {
		return ctrl.Result{}, r.Update(ctx, obj)
	}
	return ctrl.Result{}, nil
}

func upsertUnstructured(ctx context.Context, c client.Client, desired *unstructured.Unstructured) error {
	var existing unstructured.Unstructured
	existing.SetAPIVersion(desired.GetAPIVersion())
	existing.SetKind(desired.GetKind())
	key := types.NamespacedName{Name: desired.GetName(), Namespace: desired.GetNamespace()}
	if err := c.Get(ctx, key, &existing); apierrors.IsNotFound(err) {
		return c.Create(ctx, desired)
	} else if err != nil {
		return err
	}
	desired.SetResourceVersion(existing.GetResourceVersion())
	return c.Update(ctx, desired)
}

func (r *WorkloadReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&appsv1.Deployment{}).Complete(r)
}
