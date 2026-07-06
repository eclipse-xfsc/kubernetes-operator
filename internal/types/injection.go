package types

type InjectionRequest struct {
	ConsumerNamespace string
	ConsumerName      string
	ConsumerKind      string

	Container string

	RequestedTypes []string

	Mode string
}

type InjectionResult struct {
	ConsumerNamespace string
	ConsumerName      string
	ConsumerKind      string

	Container string

	RequestedTypes []string

	Mode string

	Status string

	SourceManifest string
}
