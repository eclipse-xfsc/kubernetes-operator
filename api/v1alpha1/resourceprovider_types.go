package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type ResourceProviderSpec struct {
	Type        string            `json:"type"`
	Description string            `json:"description,omitempty"`
	Scope       string            `json:"scope,omitempty"` // Namespaced or Cluster
	Allow       ProviderAllowSpec `json:"allow,omitempty"`
	Config      ProviderConfig    `json:"config,omitempty"`
	SecretStore SecretStoreRef    `json:"secretStoreRef,omitempty"`
}

type ProviderAllowSpec struct {
	Namespaces []string          `json:"namespaces,omitempty"`
	Selector   map[string]string `json:"selector,omitempty"`
}

type ProviderConfig struct {
	Env map[string]string `json:"env,omitempty"`
}

type SecretStoreRef struct {
	Kind string `json:"kind,omitempty"` // SecretStore or ClusterSecretStore
	Name string `json:"name,omitempty"`
}

type ResourceProviderStatus struct {
	ObservedGeneration int64              `json:"observedGeneration,omitempty"`
	Conditions         []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type ResourceProvider struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ResourceProviderSpec   `json:"spec,omitempty"`
	Status            ResourceProviderStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type ResourceProviderList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ResourceProvider `json:"items"`
}
