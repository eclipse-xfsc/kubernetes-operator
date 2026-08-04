package controller

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/eclipse-xfsc/kubernetes-operator/internal/openbao"

	resourcesv1alpha1 "github.com/eclipse-xfsc/kubernetes-operator/api/v1alpha1"
	"github.com/eclipse-xfsc/kubernetes-operator/internal/modules"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apiMeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	claimReadyCondition = "Ready"
	claimFinalizer      = "resources.xfsc.io/openbao-cleanup"
)

type ResourceClaimReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
	Modules  *modules.Registry
	OpenBao  *openbao.Client
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
	if strings.TrimSpace(claim.Spec.ExternalSecretRef.Name) == "" {
		return r.fail(ctx, &claim, "InvalidClaim", "spec.externalSecretRef.name must not be empty")
	}

	if !claim.DeletionTimestamp.IsZero() {
		if containsString(claim.Finalizers, claimFinalizer) {
			remoteKey := claim.Status.RemoteSecretPath
			if r.OpenBao != nil && r.OpenBao.Enabled() && remoteKey != "" {
				if err := r.OpenBao.Delete(ctx, remoteKey); err != nil {
					return ctrl.Result{}, err
				}
			}
			claim.Finalizers = removeString(claim.Finalizers, claimFinalizer)
			if err := r.Update(ctx, &claim); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	if !containsString(claim.Finalizers, claimFinalizer) {
		claim.Finalizers = append(claim.Finalizers, claimFinalizer)
		if err := r.Update(ctx, &claim); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	externalSecret, targetSecretName, remoteKey, err := r.resolveExternalSecret(ctx, &claim)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return r.fail(ctx, &claim, "ExternalSecretPending", fmt.Sprintf("ExternalSecret %s/%s does not exist yet", claim.Namespace, claim.Spec.ExternalSecretRef.Name))
		}
		return r.fail(ctx, &claim, "ExternalSecretInvalid", err.Error())
	}
	_ = externalSecret

	if r.OpenBao == nil || !r.OpenBao.Enabled() {
		return r.fail(ctx, &claim, "OpenBaoUnavailable", "OpenBao integration is not configured")
	}
	if err := r.OpenBao.EnsureCredentials(ctx, remoteKey, claim.Name); err != nil {
		return r.fail(ctx, &claim, "OpenBaoWriteFailed", err.Error())
	}

	provider, err := r.resolveProvider(ctx, &claim)
	if err != nil {
		return r.fail(ctx, &claim, "ProviderUnavailable", err.Error())
	}
	if provider.Spec.AdminSecretRef == nil || strings.TrimSpace(provider.Spec.AdminSecretRef.Name) == "" {
		return r.fail(ctx, &claim, "AdminSecretMissing", fmt.Sprintf("provider %q has no spec.adminSecretRef", provider.Name))
	}

	claimSecret, err := r.readSecret(ctx, claim.Namespace, targetSecretName)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return r.fail(ctx, &claim, "ClaimSecretPending", fmt.Sprintf("ESO target secret %s/%s does not exist yet", claim.Namespace, targetSecretName))
		}
		return ctrl.Result{}, err
	}

	adminNamespace := strings.TrimSpace(provider.Spec.AdminSecretRef.Namespace)
	if adminNamespace == "" {
		adminNamespace = claim.Namespace
	}
	adminSecret, err := r.readSecret(ctx, adminNamespace, provider.Spec.AdminSecretRef.Name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return r.fail(ctx, &claim, "AdminSecretPending", fmt.Sprintf("admin secret %s/%s does not exist", adminNamespace, provider.Spec.AdminSecretRef.Name))
		}
		return ctrl.Result{}, err
	}

	if r.Modules == nil {
		return r.fail(ctx, &claim, "ModuleUnavailable", "provisioner registry is not configured")
	}
	if err := r.Modules.Provision(ctx, claim.Spec.Type, modules.ProvisionRequest{Provider: *provider, Claim: claim, AdminSecret: *adminSecret, ClaimSecret: *claimSecret}); err != nil {
		return r.fail(ctx, &claim, "ProvisioningFailed", err.Error())
	}

	log.Info("resource claim provisioned", "provider", provider.Name, "externalSecret", claim.Spec.ExternalSecretRef.Name, "claimSecret", targetSecretName)
	if r.Recorder != nil {
		r.Recorder.Event(&claim, corev1.EventTypeNormal, "Provisioned", "Resource provisioned by "+provider.Name)
	}
	return ctrl.Result{}, r.markReady(ctx, &claim, provider.Name, claim.Spec.ExternalSecretRef.Name, targetSecretName, remoteKey)
}

func (r *ResourceClaimReconciler) resolveExternalSecret(ctx context.Context, claim *resourcesv1alpha1.ResourceClaim) (*unstructured.Unstructured, string, string, error) {
	es := &unstructured.Unstructured{}
	es.SetGroupVersionKind(schema.GroupVersionKind{Group: "external-secrets.io", Version: "v1", Kind: "ExternalSecret"})
	if err := r.Get(ctx, types.NamespacedName{Namespace: claim.Namespace, Name: claim.Spec.ExternalSecretRef.Name}, es); err != nil {
		return nil, "", "", err
	}
	targetName, _, _ := unstructured.NestedString(es.Object, "spec", "target", "name")
	if strings.TrimSpace(targetName) == "" {
		targetName = es.GetName()
	}
	data, _, _ := unstructured.NestedSlice(es.Object, "spec", "data")
	remoteKey := ""
	for _, item := range data {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		ref, ok := entry["remoteRef"].(map[string]any)
		if !ok {
			continue
		}
		key, _ := ref["key"].(string)
		if strings.TrimSpace(key) == "" {
			continue
		}
		if remoteKey == "" {
			remoteKey = key
		}
		if remoteKey != key {
			return nil, "", "", fmt.Errorf("ExternalSecret %s/%s references multiple remote keys; one credentials path is required", claim.Namespace, es.GetName())
		}
	}
	if remoteKey == "" {
		return nil, "", "", fmt.Errorf("ExternalSecret %s/%s has no spec.data[].remoteRef.key", claim.Namespace, es.GetName())
	}
	return es, targetName, remoteKey, nil
}

func containsString(items []string, value string) bool {
	for _, item := range items {
		if item == value {
			return true
		}
	}
	return false
}
func removeString(items []string, value string) []string {
	out := items[:0]
	for _, item := range items {
		if item != value {
			out = append(out, item)
		}
	}
	return out
}

func (r *ResourceClaimReconciler) readSecret(ctx context.Context, namespace, name string) (*corev1.Secret, error) {
	var secret corev1.Secret
	if err := r.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, &secret); err != nil {
		return nil, err
	}
	return &secret, nil
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
		provider := list.Items[i]
		if strings.EqualFold(provider.Spec.Type, claim.Spec.Type) && providerAllows(&provider, claim.Namespace) {
			matches = append(matches, provider)
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

func (r *ResourceClaimReconciler) markReady(ctx context.Context, claim *resourcesv1alpha1.ResourceClaim, provider, externalSecret, secret, remotePath string) error {
	claim.Status.ObservedGeneration = claim.Generation
	claim.Status.Phase = "Ready"
	claim.Status.ProviderRef = provider
	claim.Status.ExternalSecretRef = externalSecret
	claim.Status.SecretRef = secret
	claim.Status.RemoteSecretPath = remotePath
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
	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
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
	out := make([]reconcile.Request, 0)
	for i := range claims.Items {
		claim := &claims.Items[i]
		if strings.EqualFold(claim.Spec.Type, provider.Spec.Type) {
			out = append(out, reconcile.Request{NamespacedName: types.NamespacedName{Namespace: claim.Namespace, Name: claim.Name}})
		}
	}
	return out
}

func (r *ResourceClaimReconciler) mapSecretToClaims(ctx context.Context, obj client.Object) []reconcile.Request {
	secret, ok := obj.(*corev1.Secret)
	if !ok {
		return nil
	}
	var claims resourcesv1alpha1.ResourceClaimList
	if err := r.List(ctx, &claims); err != nil {
		return nil
	}
	out := make([]reconcile.Request, 0)
	for i := range claims.Items {
		claim := &claims.Items[i]
		targetName := claim.Status.SecretRef
		if targetName != "" && claim.Namespace == secret.Namespace && targetName == secret.Name {
			out = append(out, reconcile.Request{NamespacedName: types.NamespacedName{Namespace: claim.Namespace, Name: claim.Name}})
			continue
		}
		provider, err := r.resolveProvider(ctx, claim)
		if err != nil || provider.Spec.AdminSecretRef == nil {
			continue
		}
		adminNamespace := provider.Spec.AdminSecretRef.Namespace
		if adminNamespace == "" {
			adminNamespace = claim.Namespace
		}
		if adminNamespace == secret.Namespace && provider.Spec.AdminSecretRef.Name == secret.Name {
			out = append(out, reconcile.Request{NamespacedName: types.NamespacedName{Namespace: claim.Namespace, Name: claim.Name}})
		}
	}
	return out
}

func (r *ResourceClaimReconciler) mapExternalSecretToClaims(ctx context.Context, obj client.Object) []reconcile.Request {
	es, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return nil
	}
	var claims resourcesv1alpha1.ResourceClaimList
	if err := r.List(ctx, &claims, client.InNamespace(es.GetNamespace())); err != nil {
		return nil
	}
	out := make([]reconcile.Request, 0)
	for i := range claims.Items {
		claim := &claims.Items[i]
		if claim.Spec.ExternalSecretRef.Name == es.GetName() {
			out = append(out, reconcile.Request{NamespacedName: types.NamespacedName{Namespace: claim.Namespace, Name: claim.Name}})
		}
	}
	return out
}

func (r *ResourceClaimReconciler) SetupWithManager(mgr ctrl.Manager) error {
	externalSecret := &unstructured.Unstructured{}
	externalSecret.SetGroupVersionKind(schema.GroupVersionKind{Group: "external-secrets.io", Version: "v1", Kind: "ExternalSecret"})
	return ctrl.NewControllerManagedBy(mgr).
		For(&resourcesv1alpha1.ResourceClaim{}).
		Watches(&resourcesv1alpha1.ResourceProvider{}, handler.EnqueueRequestsFromMapFunc(r.mapProviderToClaims)).
		Watches(&corev1.Secret{}, handler.EnqueueRequestsFromMapFunc(r.mapSecretToClaims)).
		Watches(externalSecret, handler.EnqueueRequestsFromMapFunc(r.mapExternalSecretToClaims)).
		Complete(r)
}
