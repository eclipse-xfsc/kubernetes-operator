package redis

import (
	"context"
	"fmt"
	"github.com/eclipse-xfsc/kubernetes-operator/internal/modules"
)

type Provisioner interface {
	EnsureUser(context.Context, modules.Request) (modules.Result, error)
}
type Module struct{ provisioner Provisioner }

func New(p Provisioner) *Module { return &Module{provisioner: p} }
func (m *Module) Type() string  { return "redis" }
func (m *Module) Reconcile(ctx context.Context, req modules.Request) (modules.Result, error) {
	if m != nil && m.provisioner != nil {
		return m.provisioner.EnsureUser(ctx, req)
	}
	if req.Claim == nil {
		return modules.Result{}, nil
	}
	p, err := modules.Parameters(req)
	if err != nil {
		return modules.Result{}, err
	}
	user := modules.StringParameter(p, "username", modules.ClaimBaseName(req))
	password, err := modules.RandomPassword(24)
	if err != nil {
		return modules.Result{}, err
	}
	host := modules.ProviderEnv(req, "host", "redis.default.svc")
	port := modules.ProviderEnv(req, "port", "6379")
	return modules.Result{SecretData: map[string][]byte{"host": []byte(host), "port": []byte(port), "username": []byte(user), "password": password, "uri": []byte(fmt.Sprintf("redis://%s:%s@%s:%s", user, string(password), host, port))}}, nil
}
