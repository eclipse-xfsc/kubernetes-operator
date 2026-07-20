package nats

import (
	"context"
	"fmt"
	"github.com/eclipse-xfsc/kubernetes-operator/internal/modules"
)

type Provisioner interface {
	EnsureAccount(context.Context, modules.Request) (modules.Result, error)
}
type Module struct{ provisioner Provisioner }

func New(p Provisioner) *Module { return &Module{provisioner: p} }
func (m *Module) Type() string  { return "nats" }
func (m *Module) Reconcile(ctx context.Context, req modules.Request) (modules.Result, error) {
	if m != nil && m.provisioner != nil {
		return m.provisioner.EnsureAccount(ctx, req)
	}
	if req.Claim == nil {
		return modules.Result{}, nil
	}
	p, err := modules.Parameters(req)
	if err != nil {
		return modules.Result{}, err
	}
	name := modules.ClaimBaseName(req)
	account := modules.StringParameter(p, "account", name)
	user := modules.StringParameter(p, "user", name)
	password, err := modules.RandomPassword(24)
	if err != nil {
		return modules.Result{}, err
	}
	host := modules.ProviderEnv(req, "host", "nats.default.svc")
	port := modules.ProviderEnv(req, "port", "4222")
	return modules.Result{SecretData: map[string][]byte{"host": []byte(host), "port": []byte(port), "url": []byte(fmt.Sprintf("nats://%s:%s", host, port)), "account": []byte(account), "username": []byte(user), "password": password}}, nil
}
