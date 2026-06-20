package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +kubebuilder:object:generate=true
type ResourceProfileSpec struct {
	Product  ProductRef       `json:"product,omitempty"`
	Exports  []ResourceExport `json:"exports,omitempty"`
	Requires []ResourceNeed   `json:"requires,omitempty"`
}

// +kubebuilder:object:generate=true
type ResourceProfileStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type ResourceProfile struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ResourceProfileSpec   `json:"spec,omitempty"`
	Status            ResourceProfileStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type ResourceProfileList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ResourceProfile `json:"items"`
}

func init() { SchemeBuilder.Register(&ResourceProfile{}, &ResourceProfileList{}) }
