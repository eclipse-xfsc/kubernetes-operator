package postgres

import (
	"context"
	"fmt"
	"github.com/eclipse-xfsc/kubernetes-operator/internal/modules"
)

type Parameters struct {
	Database   string   `json:"database,omitempty"`
	Username   string   `json:"username,omitempty"`
	Schemas    []string `json:"schemas,omitempty"`
	Extensions []string `json:"extensions,omitempty"`
}
type PostgresBackend struct {
	factory func(modules.ProvisionRequest) (*Client, error)
}

func NewBackend() *PostgresBackend {
	return &PostgresBackend{factory: func(req modules.ProvisionRequest) (*Client, error) { return NewClient(req.AdminSecret, nil) }}
}
func (b *PostgresBackend) Provision(ctx context.Context, req modules.ProvisionRequest) error {
	var p Parameters
	if err := modules.DecodeParameters(req.Claim, &p); err != nil {
		return err
	}
	if p.Database == "" {
		p.Database = req.Claim.Name
	}
	user, err := modules.SecretString(req.ClaimSecret, "username", "user")
	if err != nil {
		return err
	}
	pass, err := modules.SecretString(req.ClaimSecret, "password")
	if err != nil {
		return err
	}
	c, err := b.factory(req)
	if err != nil {
		return fmt.Errorf("create postgres client: %w", err)
	}
	if err = c.EnsureRole(ctx, user, pass); err != nil {
		return err
	}
	if err = c.EnsureDatabase(ctx, p.Database, user); err != nil {
		return err
	}
	return c.EnsureDatabaseObjects(ctx, p.Database, user, p.Schemas, p.Extensions)
}
