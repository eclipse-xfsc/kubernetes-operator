package types

type Provider struct {
	Type      string `json:"type"`
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
	Kind      string `json:"kind,omitempty"`
	Resource  string `json:"resource,omitempty"`
	Module    string `json:"module,omitempty"`
}
