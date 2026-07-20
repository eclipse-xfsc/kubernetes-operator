package postgres

import (
	"context"

	"github.com/eclipse-xfsc/kubernetes-operator/internal/modules"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Provisioner defines the provider-specific PostgreSQL provisioning boundary.
// Concrete behavior will be added independently from the generic injection
// pipeline.
type Provisioner interface {
	EnsureDatabaseAndRole(context.Context, modules.Request) ([]*unstructured.Unstructured, error)
}

type Module struct {
	provisioner Provisioner
}

func New(provisioner Provisioner) *Module {
	return &Module{provisioner: provisioner}
}

func (m *Module) Type() string {
	return "postgres"
}

func (m *Module) Reconcile(ctx context.Context, req modules.Request) (modules.Result, error) {
	if m == nil || m.provisioner == nil {
		return modules.Result{}, nil
	}
	resources, err := m.provisioner.EnsureDatabaseAndRole(ctx, req)
	if err != nil {
		return modules.Result{}, err
	}
	return modules.Result{Resources: resources}, nil
}
