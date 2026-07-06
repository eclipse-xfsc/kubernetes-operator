package controller

import (
	"context"
	"fmt"

	resourcesv1alpha1 "github.com/eclipse-xfsc/kubernetes-operator/api/v1alpha1"
	"github.com/eclipse-xfsc/kubernetes-operator/internal/injection"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type ResourceBindingReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *ResourceBindingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var binding resourcesv1alpha1.ResourceBinding
	if err := r.Get(ctx, req.NamespacedName, &binding); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	providerNS := binding.Spec.ProviderRef.Namespace
	if providerNS == "" {
		providerNS = binding.Namespace
	}
	var provider resourcesv1alpha1.ResourceProvider
	if err := r.Get(ctx, types.NamespacedName{Name: binding.Spec.ProviderRef.Name, Namespace: providerNS}, &provider); err != nil {
		return ctrl.Result{}, err
	}

	es := injection.BuildExternalSecret(&binding, &provider)
	if err := controllerutil.SetControllerReference(&binding, es, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}
	var existing unstructured.Unstructured
	existing.SetAPIVersion("external-secrets.io/v1")
	existing.SetKind("ExternalSecret")
	if err := r.Get(ctx, types.NamespacedName{Name: es.GetName(), Namespace: es.GetNamespace()}, &existing); apierrors.IsNotFound(err) {
		if err := r.Create(ctx, es); err != nil {
			return ctrl.Result{}, err
		}
	} else if err != nil {
		return ctrl.Result{}, err
	} else {
		es.SetResourceVersion(existing.GetResourceVersion())
		if err := r.Update(ctx, es); err != nil {
			return ctrl.Result{}, err
		}
	}

	if err := r.patchConsumer(ctx, &binding, &provider); err != nil {
		return ctrl.Result{}, err
	}

	binding.Status.Phase = "Ready"
	binding.Status.ExternalSecretName = es.GetName()
	binding.Status.TargetSecretName = targetSecretName(&binding)
	binding.Status.ObservedGeneration = binding.Generation
	metaSet(&binding.Status.Conditions, "Ready", metav1.ConditionTrue, "Reconciled", "ExternalSecret and injection applied")
	_ = r.Status().Update(ctx, &binding)
	return ctrl.Result{}, nil
}

func (r *ResourceBindingReconciler) patchConsumer(ctx context.Context, binding *resourcesv1alpha1.ResourceBinding, provider *resourcesv1alpha1.ResourceProvider) error {
	obj := &unstructured.Unstructured{}
	switch binding.Spec.ConsumerRef.Kind {
	case "Deployment":
		obj.SetAPIVersion(appsv1.SchemeGroupVersion.String())
		obj.SetKind("Deployment")
	case "StatefulSet":
		obj.SetAPIVersion(appsv1.SchemeGroupVersion.String())
		obj.SetKind("StatefulSet")
	case "DaemonSet":
		obj.SetAPIVersion(appsv1.SchemeGroupVersion.String())
		obj.SetKind("DaemonSet")
	case "Job":
		obj.SetAPIVersion(batchv1.SchemeGroupVersion.String())
		obj.SetKind("Job")
	default:
		return fmt.Errorf("unsupported consumer kind %s", binding.Spec.ConsumerRef.Kind)
	}
	if err := r.Get(ctx, types.NamespacedName{Name: binding.Spec.ConsumerRef.Name, Namespace: binding.Namespace}, obj); err != nil {
		return err
	}
	providers := map[string]resourcesv1alpha1.ResourceProvider{providerNSName(binding, provider): *provider}
	if err := injection.PatchWorkload(obj, []resourcesv1alpha1.ResourceBinding{*binding}, providers); err != nil {
		return err
	}
	return r.Update(ctx, obj)
}

func (r *ResourceBindingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&resourcesv1alpha1.ResourceBinding{}).Complete(r)
}
func targetSecretName(b *resourcesv1alpha1.ResourceBinding) string {
	if b.Spec.Secret.TargetSecretName != "" {
		return b.Spec.Secret.TargetSecretName
	}
	return b.Name + "-secret"
}
func providerNSName(b *resourcesv1alpha1.ResourceBinding, p *resourcesv1alpha1.ResourceProvider) string {
	ns := b.Spec.ProviderRef.Namespace
	if ns == "" {
		ns = b.Namespace
	}
	return ns + "/" + p.Name
}
func metaSet(conditions *[]metav1.Condition, t string, s metav1.ConditionStatus, r string, m string) {
	meta.SetStatusCondition(conditions, metav1.Condition{Type: t, Status: s, Reason: r, Message: m})
}
