package telemetry

import (
	"context"

	"github.com/eclipse-xfsc/kubernetes-operator/internal/modules"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const ModuleName = "telemetry"
const ModuleVersion = "0.1.0"
const TypeName = "telemetry"

type Module struct{}

func New() *Module                { return &Module{} }
func (m *Module) Name() string    { return ModuleName }
func (m *Module) Version() string { return ModuleVersion }
func (m *Module) Types() []string { return []string{TypeName} }
func (m *Module) Capabilities() []modules.Capability {
	return []modules.Capability{modules.CapabilityProvide, modules.CapabilityConsume, modules.CapabilityInject, modules.CapabilityCreateResource}
}
func (m *Module) Provide(ctx context.Context, obj client.Object) ([]modules.Provider, error) {
	if !modules.IsProvider(obj, TypeName) {
		return nil, nil
	}
	return []modules.Provider{{Type: TypeName, Name: modules.ProviderName(obj), Namespace: obj.GetNamespace(), Kind: obj.GetObjectKind().GroupVersionKind().Kind, Resource: obj.GetName(), Module: ModuleName}}, nil
}
func (m *Module) Consume(ctx context.Context, obj client.Object) ([]modules.Consumer, error) {
	if !modules.WantsType(obj, TypeName) {
		return nil, nil
	}
	return []modules.Consumer{{Type: TypeName, Name: obj.GetName(), Namespace: obj.GetNamespace(), Kind: obj.GetObjectKind().GroupVersionKind().Kind, Resource: obj.GetName(), RequestedTypes: modules.RequestedTypes(obj)}}, nil
}
func (m *Module) Inject(ctx context.Context, req modules.InjectionRequest) (*modules.InjectionResult, error) {
	return &modules.InjectionResult{ConsumerNamespace: req.ConsumerNamespace, ConsumerName: req.ConsumerName, ConsumerKind: req.ConsumerKind, Container: req.Container, RequestedTypes: req.RequestedTypes, Mode: req.Mode, Status: "planned"}, nil
}
func (m *Module) CreateResources(ctx context.Context, req modules.CreateResourceRequest) ([]client.Object, error) {
	cm := &corev1.ConfigMap{}
	cm.Name = "xsfc-inject-" + req.Type + "-" + req.Consumer.Name
	cm.Namespace = req.Consumer.Namespace
	cm.Labels = map[string]string{"app.kubernetes.io/managed-by": "xfsc-operator", "xfsc.io/generated-for": req.Consumer.Name, "xfsc.io/resource-type": req.Type}
	return []client.Object{cm}, nil
}
