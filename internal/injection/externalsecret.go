package injection

import (
	resourcesv1alpha1 "github.com/eclipse-xfsc/kubernetes-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func BuildExternalSecret(binding *resourcesv1alpha1.ResourceBinding, provider *resourcesv1alpha1.ResourceProvider) *unstructured.Unstructured {
	store := binding.Spec.Secret.StoreRef
	if store.Name == "" {
		store = provider.Spec.SecretStore
	}
	if store.Kind == "" {
		store.Kind = "ClusterSecretStore"
	}

	targetName := binding.Spec.Secret.TargetSecretName
	if targetName == "" {
		targetName = binding.Name + "-secret"
	}
	refresh := binding.Spec.Secret.RefreshInterval
	if refresh == "" {
		refresh = "1h"
	}

	data := make([]any, 0, len(binding.Spec.Secret.Data))
	for _, d := range binding.Spec.Secret.Data {
		data = append(data, map[string]any{
			"secretKey": d.EnvName,
			"remoteRef": map[string]any{
				"key":      binding.Spec.Secret.RemoteKey,
				"property": d.Property,
			},
		})
	}

	es := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "external-secrets.io/v1",
		"kind":       "ExternalSecret",
		"metadata": map[string]any{
			"name":      binding.Name + "-eso",
			"namespace": binding.Namespace,
			"labels":    map[string]any{"app.kubernetes.io/managed-by": "xsfc-resource-operator"},
		},
		"spec": map[string]any{
			"refreshInterval": refresh,
			"secretStoreRef":  map[string]any{"kind": store.Kind, "name": store.Name},
			"target":          map[string]any{"name": targetName, "creationPolicy": "Owner"},
			"data":            data,
		},
	}}
	es.SetOwnerReferences([]metav1.OwnerReference{{APIVersion: binding.APIVersion, Kind: binding.Kind, Name: binding.Name, UID: binding.UID, Controller: ptr(true)}})
	return es
}

func ptr[T any](v T) *T { return &v }
