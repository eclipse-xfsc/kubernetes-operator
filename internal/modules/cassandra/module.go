package cassandra

import (
	"context"

	"github.com/eclipse-xfsc/kubernetes-operator/internal/modules"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Provisioner defines the provider-specific Cassandra provisioning boundary.
type Provisioner interface {
	EnsureAccount(context.Context, modules.Request) ([]*unstructured.Unstructured, error)
}

type Module struct {
	provisioner Provisioner
}

func New(provisioner Provisioner) *Module {
	return &Module{provisioner: provisioner}
}

func (m *Module) Type() string {
	return "cassandra"
}

func (m *Module) Reconcile(ctx context.Context, req modules.Request) (modules.Result, error) {
	if m == nil || m.provisioner == nil {
		return modules.Result{}, nil
	}
	resources, err := m.provisioner.EnsureAccount(ctx, req)
	if err != nil {
		return modules.Result{}, err
	}
	return modules.Result{Resources: resources}, nil
}
