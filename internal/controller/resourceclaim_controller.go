package controller

import (
	"context"
	"fmt"
	"sort"
	"strings"

	resourcesv1alpha1 "github.com/eclipse-xfsc/kubernetes-operator/api/v1alpha1"
	"github.com/eclipse-xfsc/kubernetes-operator/internal/modules"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apiMeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const claimReadyCondition = "Ready"

type ResourceClaimReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
	Modules  *modules.Registry
}

func (r *ResourceClaimReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx).WithValues("resourceClaim", req.NamespacedName)
	var claim resourcesv1alpha1.ResourceClaim
	if err := r.Get(ctx, req.NamespacedName, &claim); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if strings.TrimSpace(claim.Spec.Type) == "" {
		return r.fail(ctx, &claim, "InvalidClaim", "spec.type must not be empty")
	}
	provider, err := r.resolveProvider(ctx, &claim)
	if err != nil {
		return r.fail(ctx, &claim, "ProviderUnavailable", err.Error())
	}
	secretName := strings.TrimSpace(claim.Spec.SecretName)
	if secretName == "" {
		secretName = claim.Name
	}

	var existing corev1.Secret
	secretKey := types.NamespacedName{Namespace: claim.Namespace, Name: secretName}
	if err := r.Get(ctx, secretKey, &existing); err == nil {
		return ctrl.Result{}, r.markReady(ctx, &claim, provider.Name, secretName)
	} else if !apierrors.IsNotFound(err) {
		return ctrl.Result{}, err
	}

	if r.Modules == nil {
		return r.fail(ctx, &claim, "ModuleUnavailable", "module registry is not configured")
	}
	module, found := r.Modules.Get(claim.Spec.Type)
	if !found {
		return r.fail(ctx, &claim, "ModuleUnavailable", fmt.Sprintf("no module registered for type %q", claim.Spec.Type))
	}
	result, err := module.Reconcile(ctx, modules.Request{Client: r.Client, Provider: *provider, Claim: &claim, Namespace: claim.Namespace})
	if err != nil {
		return r.fail(ctx, &claim, "ProvisioningFailed", err.Error())
	}

	for _, resource := range result.Resources {
		if resource == nil {
			continue
		}
		if resource.GetNamespace() == "" {
			resource.SetNamespace(claim.Namespace)
		}
		if err := controllerutil.SetControllerReference(&claim, resource, r.Scheme); err != nil {
			return ctrl.Result{}, err
		}
		if _, err := upsertUnstructured(ctx, r.Client, resource); err != nil {
			return ctrl.Result{}, err
		}
	}
	if len(result.SecretData) == 0 {
		return r.fail(ctx, &claim, "EmptyProvisionResult", "module returned no secret data")
	}
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: claim.Namespace, Labels: map[string]string{"resources.xfsc.io/claim": claim.Name, "resources.xfsc.io/provider": provider.Name}}, Type: corev1.SecretTypeOpaque, Data: result.SecretData}
	if err := controllerutil.SetControllerReference(&claim, secret, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.Create(ctx, secret); err != nil && !apierrors.IsAlreadyExists(err) {
		return ctrl.Result{}, err
	}
	log.Info("resource claim provisioned", "provider", provider.Name, "secret", secretName)
	if r.Recorder != nil {
		r.Recorder.Event(&claim, corev1.EventTypeNormal, "Provisioned", "Resource provisioned by "+provider.Name)
	}
	return ctrl.Result{}, r.markReady(ctx, &claim, provider.Name, secretName)
}

func (r *ResourceClaimReconciler) resolveProvider(ctx context.Context, claim *resourcesv1alpha1.ResourceClaim) (*resourcesv1alpha1.ResourceProvider, error) {
	if claim.Spec.ProviderRef != nil && strings.TrimSpace(claim.Spec.ProviderRef.Name) != "" {
		var provider resourcesv1alpha1.ResourceProvider
		if err := r.Get(ctx, types.NamespacedName{Name: claim.Spec.ProviderRef.Name}, &provider); err != nil {
			return nil, err
		}
		if !strings.EqualFold(provider.Spec.Type, claim.Spec.Type) {
			return nil, fmt.Errorf("provider %q has type %q, expected %q", provider.Name, provider.Spec.Type, claim.Spec.Type)
		}
		if !providerAllows(&provider, claim.Namespace) {
			return nil, fmt.Errorf("provider %q does not allow namespace %q", provider.Name, claim.Namespace)
		}
		return &provider, nil
	}
	selector := labels.Everything()
	if claim.Spec.ProviderSelector != nil {
		parsed, err := metav1.LabelSelectorAsSelector(claim.Spec.ProviderSelector)
		if err != nil {
			return nil, fmt.Errorf("invalid providerSelector: %w", err)
		}
		selector = parsed
	}
	var list resourcesv1alpha1.ResourceProviderList
	if err := r.List(ctx, &list, client.MatchingLabelsSelector{Selector: selector}); err != nil {
		return nil, err
	}
	matches := make([]resourcesv1alpha1.ResourceProvider, 0)
	for i := range list.Items {
		p := list.Items[i]
		if strings.EqualFold(p.Spec.Type, claim.Spec.Type) && providerAllows(&p, claim.Namespace) {
			matches = append(matches, p)
		}
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("no ResourceProvider found for type %q", claim.Spec.Type)
	}
	sort.Slice(matches, func(i, j int) bool { return matches[i].Name < matches[j].Name })
	return &matches[0], nil
}

func providerAllows(provider *resourcesv1alpha1.ResourceProvider, namespace string) bool {
	if len(provider.Spec.Allow.Namespaces) == 0 {
		return true
	}
	for _, allowed := range provider.Spec.Allow.Namespaces {
		if allowed == "*" || allowed == namespace {
			return true
		}
	}
	return false
}

func (r *ResourceClaimReconciler) markReady(ctx context.Context, claim *resourcesv1alpha1.ResourceClaim, provider, secret string) error {
	claim.Status.ObservedGeneration = claim.Generation
	claim.Status.Phase = "Ready"
	claim.Status.ProviderRef = provider
	claim.Status.SecretRef = secret
	apiMeta.SetStatusCondition(&claim.Status.Conditions, metav1.Condition{Type: claimReadyCondition, Status: metav1.ConditionTrue, Reason: "Provisioned", Message: "Resource is ready", ObservedGeneration: claim.Generation})
	return r.Status().Update(ctx, claim)
}
func (r *ResourceClaimReconciler) fail(ctx context.Context, claim *resourcesv1alpha1.ResourceClaim, reason, message string) (ctrl.Result, error) {
	claim.Status.ObservedGeneration = claim.Generation
	claim.Status.Phase = "Pending"
	apiMeta.SetStatusCondition(&claim.Status.Conditions, metav1.Condition{Type: claimReadyCondition, Status: metav1.ConditionFalse, Reason: reason, Message: message, ObservedGeneration: claim.Generation})
	_ = r.Status().Update(ctx, claim)
	if r.Recorder != nil {
		r.Recorder.Event(claim, corev1.EventTypeWarning, reason, message)
	}
	return ctrl.Result{RequeueAfter: 30_000_000_000}, nil
}
func (r *ResourceClaimReconciler) mapProviderToClaims(ctx context.Context, obj client.Object) []reconcile.Request {
	provider, ok := obj.(*resourcesv1alpha1.ResourceProvider)
	if !ok {
		return nil
	}
	var claims resourcesv1alpha1.ResourceClaimList
	if err := r.List(ctx, &claims); err != nil {
		return nil
	}
	out := []reconcile.Request{}
	for i := range claims.Items {
		c := &claims.Items[i]
		if strings.EqualFold(c.Spec.Type, provider.Spec.Type) {
			out = append(out, reconcile.Request{NamespacedName: types.NamespacedName{Namespace: c.Namespace, Name: c.Name}})
		}
	}
	return out
}
func (r *ResourceClaimReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&resourcesv1alpha1.ResourceClaim{}).Watches(&resourcesv1alpha1.ResourceProvider{}, handler.EnqueueRequestsFromMapFunc(r.mapProviderToClaims)).Owns(&corev1.Secret{}).Complete(r)
}
