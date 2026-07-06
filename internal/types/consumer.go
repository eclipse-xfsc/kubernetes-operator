package types

type Consumer struct {
	Type           string   `json:"type"`
	Name           string   `json:"name"`
	Namespace      string   `json:"namespace,omitempty"`
	Kind           string   `json:"kind,omitempty"`
	Resource       string   `json:"resource,omitempty"`
	RequestedTypes []string `json:"requestedTypes,omitempty"`
}
