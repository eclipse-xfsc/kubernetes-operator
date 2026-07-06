package types

type Binding struct {
	Type string

	Provider Provider
	Consumer Consumer

	Status string
}
