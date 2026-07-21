package vault

import (
	"context"
	"fmt"

	"github.com/eclipse-xfsc/kubernetes-operator/internal/modules"
)

type Backend interface {
	Provision(context.Context, modules.ProvisionRequest) error
}

type Module struct{ backend Backend }

func New(backend Backend) *Module { return &Module{backend: backend} }
func (m *Module) Type() string    { return "vault" }
func (m *Module) Provision(ctx context.Context, req modules.ProvisionRequest) error {
	if m.backend == nil {
		return fmt.Errorf("vault provisioning backend is not configured")
	}
	return m.backend.Provision(ctx, req)
}
