package modules

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Capability string

const (
	CapabilityProvide        Capability = "provide"
	CapabilityConsume        Capability = "consume"
	CapabilityInject         Capability = "inject"
	CapabilityCreateResource Capability = "createResource"
)

type Provider struct{ Type, Name, Namespace, Kind, Resource, Module string }
type Consumer struct {
	Type, Name, Namespace, Kind, Resource string
	RequestedTypes                        []string
}
type InjectionRequest struct {
	ConsumerNamespace, ConsumerName, ConsumerKind, Container, Mode string
	RequestedTypes                                                 []string
}
type InjectionResult struct {
	ConsumerNamespace, ConsumerName, ConsumerKind, Container string
	RequestedTypes                                           []string
	Mode, Status, SourceManifest                             string
}
type CreateResourceRequest struct {
	Consumer Consumer
	Provider Provider
	Type     string
}
type Account struct{ Name, Namespace, Type, ConsumerNamespace, ConsumerName, ProviderName, ProviderNamespace, CreatedBy string }

type Module interface {
	Name() string
	Version() string
	Types() []string
	Capabilities() []Capability
	Provide(ctx context.Context, obj client.Object) ([]Provider, error)
	Consume(ctx context.Context, obj client.Object) ([]Consumer, error)
	Inject(ctx context.Context, req InjectionRequest) (*InjectionResult, error)
	CreateResources(ctx context.Context, req CreateResourceRequest) ([]client.Object, error)
}
