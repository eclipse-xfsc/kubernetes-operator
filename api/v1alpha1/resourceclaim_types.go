package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type ResourceClaimSpec struct {
	Type             string                `json:"type"`
	ProviderRef      *LocalProviderRef     `json:"providerRef,omitempty"`
	ProviderSelector *metav1.LabelSelector `json:"providerSelector,omitempty"`
	Parameters       runtime.RawExtension  `json:"parameters,omitempty"`
	SecretName       string                `json:"secretName,omitempty"`
}

type LocalProviderRef struct {
	Name string `json:"name"`
}

type ResourceClaimStatus struct {
	ObservedGeneration int64              `json:"observedGeneration,omitempty"`
	Phase              string             `json:"phase,omitempty"`
	ProviderRef        string             `json:"providerRef,omitempty"`
	SecretRef          string             `json:"secretRef,omitempty"`
	Conditions         []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced,shortName=rc
// +kubebuilder:subresource:status
type ResourceClaim struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ResourceClaimSpec   `json:"spec,omitempty"`
	Status            ResourceClaimStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type ResourceClaimList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ResourceClaim `json:"items"`
}
