package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type ResourceProviderSpec struct {
	Type           string               `json:"type"`
	Description    string               `json:"description,omitempty"`
	Allow          ProviderAllowSpec    `json:"allow,omitempty"`
	Outputs        ProviderOutputs      `json:"outputs,omitempty"`
	AdminSecretRef *NamespacedSecretRef `json:"adminSecretRef,omitempty"`
}

type NamespacedSecretRef struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
}

type ProviderAllowSpec struct {
	Namespaces []string          `json:"namespaces,omitempty"`
	Selector   map[string]string `json:"selector,omitempty"`
}

type ProviderOutputs struct {
	Env             map[string]string      `json:"env,omitempty"`
	ExternalSecrets []ExternalSecretOutput `json:"externalSecrets,omitempty"`
	Config          []ConfigMapOutput      `json:"config,omitempty"`
	Jobs            []JobOutput            `json:"jobs,omitempty"`
}

type ConfigMapOutput struct {
	NameTemplate string            `json:"nameTemplate,omitempty"`
	Data         map[string]string `json:"data,omitempty"`
	Env          map[string]string `json:"env,omitempty"`
}

type JobOutput struct {
	NameTemplate string `json:"nameTemplate,omitempty"`
	YAML         string `json:"yaml"`
}

type ExternalSecretOutput struct {
	NameTemplate             string              `json:"nameTemplate,omitempty"`
	TargetSecretNameTemplate string              `json:"targetSecretNameTemplate,omitempty"`
	RefreshInterval          string              `json:"refreshInterval,omitempty"`
	SecretStoreRef           SecretStoreRef      `json:"secretStoreRef,omitempty"`
	RemoteKeyTemplate        string              `json:"remoteKeyTemplate"`
	Data                     []ExternalSecretKey `json:"data,omitempty"`
}

type SecretStoreRef struct {
	Kind string `json:"kind,omitempty"`
	Name string `json:"name,omitempty"`
}

type ExternalSecretKey struct {
	EnvName  string `json:"envName"`
	Property string `json:"property"`
}

type ResourceProviderStatus struct {
	ObservedGeneration int64              `json:"observedGeneration,omitempty"`
	Conditions         []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster,shortName=rp
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
