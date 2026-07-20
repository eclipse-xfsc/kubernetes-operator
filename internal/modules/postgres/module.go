package postgres

import (
	"context"
	"fmt"

	"github.com/eclipse-xfsc/kubernetes-operator/internal/modules"
)

type Provisioner interface {
	EnsureDatabaseAndRole(context.Context, modules.Request) (modules.Result, error)
}
type Module struct{ provisioner Provisioner }

func New(p Provisioner) *Module { return &Module{provisioner: p} }
func (m *Module) Type() string  { return "postgres" }
func (m *Module) Reconcile(ctx context.Context, req modules.Request) (modules.Result, error) {
	if m != nil && m.provisioner != nil {
		return m.provisioner.EnsureDatabaseAndRole(ctx, req)
	}
	if req.Claim == nil {
		return modules.Result{}, nil
	}
	p, err := modules.Parameters(req)
	if err != nil {
		return modules.Result{}, err
	}
	name := modules.ClaimBaseName(req)
	db := modules.StringParameter(p, "database", name)
	user := modules.StringParameter(p, "username", name)
	password, err := modules.RandomPassword(24)
	if err != nil {
		return modules.Result{}, err
	}
	host := modules.ProviderEnv(req, "host", "postgres.default.svc")
	port := modules.ProviderEnv(req, "port", "5432")
	uri := fmt.Sprintf("postgresql://%s:%s@%s:%s/%s", user, string(password), host, port, db)
	return modules.Result{SecretData: map[string][]byte{"host": []byte(host), "port": []byte(port), "database": []byte(db), "username": []byte(user), "password": password, "uri": []byte(uri)}}, nil
}
