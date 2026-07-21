package postgres

import (
	"context"
	"fmt"

	"github.com/eclipse-xfsc/kubernetes-operator/internal/modules"
)

type Backend interface {
	Provision(context.Context, modules.ProvisionRequest) error
}

type Module struct {
	backend Backend
}

func New(backend Backend) *Module { return &Module{backend: backend} }
func (m *Module) Type() string    { return "postgres" }
func (m *Module) Provision(ctx context.Context, req modules.ProvisionRequest) error {
	if len(req.AdminSecret.Data) == 0 {
		return fmt.Errorf("%s provider admin secret is empty", m.Type())
	}
	if len(req.ClaimSecret.Data) == 0 {
		return fmt.Errorf("%s claim secret is empty", m.Type())
	}
	if m.backend == nil {
		return fmt.Errorf("%s provisioning backend is not configured", m.Type())
	}
	return m.backend.Provision(ctx, req)
}
