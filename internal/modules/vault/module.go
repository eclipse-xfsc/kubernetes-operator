package vault

import (
	"context"

	"github.com/eclipse-xfsc/kubernetes-operator/internal/modules"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Provisioner defines the provider-specific Vault/OpenBao provisioning boundary.
type Provisioner interface {
	EnsureAccess(context.Context, modules.Request) ([]*unstructured.Unstructured, error)
}

type Module struct {
	provisioner Provisioner
}

func New(provisioner Provisioner) *Module {
	return &Module{provisioner: provisioner}
}

func (m *Module) Type() string {
	return "vault"
}

func (m *Module) Reconcile(ctx context.Context, req modules.Request) (modules.Result, error) {
	if m == nil || m.provisioner == nil {
		return modules.Result{}, nil
	}
	resources, err := m.provisioner.EnsureAccess(ctx, req)
	if err != nil {
		return modules.Result{}, err
	}
	return modules.Result{Resources: resources}, nil
}
