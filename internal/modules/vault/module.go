package vault

import (
	"context"
	"github.com/eclipse-xfsc/kubernetes-operator/internal/modules"
)

type Provisioner interface {
	EnsureAccess(context.Context, modules.Request) (modules.Result, error)
}
type Module struct{ provisioner Provisioner }

func New(p Provisioner) *Module { return &Module{provisioner: p} }
func (m *Module) Type() string  { return "vault" }
func (m *Module) Reconcile(ctx context.Context, req modules.Request) (modules.Result, error) {
	if m != nil && m.provisioner != nil {
		return m.provisioner.EnsureAccess(ctx, req)
	}
	return modules.Result{}, nil
}
