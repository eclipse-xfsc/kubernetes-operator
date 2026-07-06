package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type ResourceBindingSpec struct {
	Type        string               `json:"type"`
	ProviderRef NamespacedNameRef    `json:"providerRef"`
	ConsumerRef ConsumerRef          `json:"consumerRef"`
	Config      ProviderConfig       `json:"config,omitempty"`
	Secret      BindingSecretSpec    `json:"secret,omitempty"`
	Injection   BindingInjectionSpec `json:"injection,omitempty"`
}

type NamespacedNameRef struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
}

type ConsumerRef struct {
	APIVersion string `json:"apiVersion,omitempty"`
	Kind       string `json:"kind"`
	Name       string `json:"name"`
}

type BindingSecretSpec struct {
	RemoteKey        string              `json:"remoteKey"`
	TargetSecretName string              `json:"targetSecretName,omitempty"`
	RefreshInterval  string              `json:"refreshInterval,omitempty"`
	StoreRef         SecretStoreRef      `json:"storeRef,omitempty"`
	Data             []ExternalSecretKey `json:"data,omitempty"`
}

type ExternalSecretKey struct {
	EnvName  string `json:"envName"`
	Property string `json:"property"`
}

type BindingInjectionSpec struct {
	Mode       string   `json:"mode,omitempty"` // Env, EnvFrom
	Containers []string `json:"containers,omitempty"`
}

type ResourceBindingStatus struct {
	ObservedGeneration int64              `json:"observedGeneration,omitempty"`
	Phase              string             `json:"phase,omitempty"`
	ExternalSecretName string             `json:"externalSecretName,omitempty"`
	TargetSecretName   string             `json:"targetSecretName,omitempty"`
	Conditions         []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type ResourceBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ResourceBindingSpec   `json:"spec,omitempty"`
	Status            ResourceBindingStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type ResourceBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ResourceBinding `json:"items"`
}
