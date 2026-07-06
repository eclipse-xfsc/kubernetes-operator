package types

type Account struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`

	Type string `json:"type"`

	ConsumerNamespace string `json:"consumerNamespace"`
	ConsumerName      string `json:"consumerName"`

	ProviderName      string `json:"providerName,omitempty"`
	ProviderNamespace string `json:"providerNamespace,omitempty"`

	CreatedBy string `json:"createdBy"`
}
