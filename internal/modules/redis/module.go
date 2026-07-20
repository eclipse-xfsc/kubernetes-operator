package redis

import (
	"context"

	"github.com/eclipse-xfsc/kubernetes-operator/internal/modules"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// AccountProvisioner is implemented by the concrete Redis integration. It can
// call a Redis administration API directly or return Kubernetes resources such
// as an account-request CR that another controller reconciles.
type AccountProvisioner interface {
	EnsureAccount(context.Context, modules.Request) ([]*unstructured.Unstructured, error)
}

type Module struct {
	provisioner AccountProvisioner
}

func New(provisioner AccountProvisioner) *Module {
	return &Module{provisioner: provisioner}
}

func (m *Module) Type() string {
	return "redis"
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
