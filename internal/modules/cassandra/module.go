package cassandra

import (
	"context"
	"github.com/eclipse-xfsc/kubernetes-operator/internal/modules"
)

type Provisioner interface {
	EnsureKeyspaceAndRole(context.Context, modules.Request) (modules.Result, error)
}
type Module struct{ provisioner Provisioner }

func New(p Provisioner) *Module { return &Module{provisioner: p} }
func (m *Module) Type() string  { return "cassandra" }
func (m *Module) Reconcile(ctx context.Context, req modules.Request) (modules.Result, error) {
	if m != nil && m.provisioner != nil {
		return m.provisioner.EnsureKeyspaceAndRole(ctx, req)
	}
	if req.Claim == nil {
		return modules.Result{}, nil
	}
	p, err := modules.Parameters(req)
	if err != nil {
		return modules.Result{}, err
	}
	name := modules.ClaimBaseName(req)
	keyspace := modules.StringParameter(p, "keyspace", name)
	user := modules.StringParameter(p, "username", name)
	password, err := modules.RandomPassword(24)
	if err != nil {
		return modules.Result{}, err
	}
	return modules.Result{SecretData: map[string][]byte{"contactPoints": []byte(modules.ProviderEnv(req, "contactPoints", "cassandra.default.svc")), "port": []byte(modules.ProviderEnv(req, "port", "9042")), "keyspace": []byte(keyspace), "username": []byte(user), "password": password}}, nil
}
