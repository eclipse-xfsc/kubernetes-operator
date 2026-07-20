package s3

import (
	"context"
	"github.com/eclipse-xfsc/kubernetes-operator/internal/modules"
)

type Provisioner interface {
	EnsureBucketAndCredentials(context.Context, modules.Request) (modules.Result, error)
}
type Module struct{ provisioner Provisioner }

func New(p Provisioner) *Module { return &Module{provisioner: p} }
func (m *Module) Type() string  { return "s3" }
func (m *Module) Reconcile(ctx context.Context, req modules.Request) (modules.Result, error) {
	if m != nil && m.provisioner != nil {
		return m.provisioner.EnsureBucketAndCredentials(ctx, req)
	}
	if req.Claim == nil {
		return modules.Result{}, nil
	}
	p, err := modules.Parameters(req)
	if err != nil {
		return modules.Result{}, err
	}
	bucket := modules.StringParameter(p, "bucket", modules.ClaimBaseName(req))
	secret, err := modules.RandomPassword(32)
	if err != nil {
		return modules.Result{}, err
	}
	access, err := modules.RandomPassword(12)
	if err != nil {
		return modules.Result{}, err
	}
	return modules.Result{SecretData: map[string][]byte{"endpoint": []byte(modules.ProviderEnv(req, "endpoint", "http://minio.default.svc:9000")), "bucket": []byte(bucket), "accessKey": access, "secretKey": secret}}, nil
}
