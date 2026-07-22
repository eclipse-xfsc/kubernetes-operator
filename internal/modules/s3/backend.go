package s3

import (
	"context"
	"fmt"
	"github.com/eclipse-xfsc/kubernetes-operator/internal/modules"
)

type Parameters struct {
	Bucket string `json:"bucket,omitempty"`
	Policy string `json:"policy,omitempty"`
}
type S3Backend struct {
	factory func(modules.ProvisionRequest) (*Client, error)
}

func NewBackend() *S3Backend {
	return &S3Backend{factory: func(req modules.ProvisionRequest) (*Client, error) { return NewClient(req.AdminSecret, nil) }}
}
func (b *S3Backend) Provision(ctx context.Context, req modules.ProvisionRequest) error {
	var p Parameters
	if err := modules.DecodeParameters(req.Claim, &p); err != nil {
		return err
	}
	if p.Bucket == "" {
		p.Bucket = req.Claim.Name
	}
	user, err := modules.SecretString(req.ClaimSecret, "username", "accessKey", "accessKeyId")
	if err != nil {
		return err
	}
	pass, err := modules.SecretString(req.ClaimSecret, "password", "secretKey", "secretAccessKey")
	if err != nil {
		return err
	}
	c, err := b.factory(req)
	if err != nil {
		return fmt.Errorf("create s3 client: %w", err)
	}
	if err = c.EnsureBucket(ctx, p.Bucket); err != nil {
		return err
	}
	if err = c.EnsureUser(ctx, user, pass); err != nil {
		return err
	}
	if p.Policy != "" {
		return c.AttachPolicy(ctx, user, p.Policy)
	}
	return nil
}
