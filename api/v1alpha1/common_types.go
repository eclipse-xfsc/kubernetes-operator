package v1alpha1
type ProductRef struct{Name string `json:"name,omitempty"`; Component string `json:"component,omitempty"`; Version string `json:"version,omitempty"`}
type ResourceExport struct{Type string `json:"type"`; Name string `json:"name,omitempty"`; Scope string `json:"scope,omitempty"`}
type ResourceNeed struct{Type string `json:"type"`; Name string `json:"name,omitempty"`; Required bool `json:"required,omitempty"`}
