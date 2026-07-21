package nats

import (
	"context"
	"fmt"
	"github.com/eclipse-xfsc/kubernetes-operator/internal/modules"
)

type Parameters struct {
	Account   string   `json:"account,omitempty"`
	User      string   `json:"user,omitempty"`
	Publish   []string `json:"publish,omitempty"`
	Subscribe []string `json:"subscribe,omitempty"`
}
type NATSBackend struct {
	factory func(modules.ProvisionRequest) (*Client, error)
}

func NewBackend() *NATSBackend {
	return &NATSBackend{factory: func(req modules.ProvisionRequest) (*Client, error) { return NewClient(req.AdminSecret, nil) }}
}
func (b *NATSBackend) Provision(ctx context.Context, req modules.ProvisionRequest) error {
	var p Parameters
	if err := modules.DecodeParameters(req.Claim, &p); err != nil {
		return err
	}
	if p.Account == "" {
		p.Account = req.Claim.Name
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
		return fmt.Errorf("create nats client: %w", err)
	}
	if err = c.EnsureAccount(ctx, p.Account); err != nil {
		return err
	}
	return c.EnsureUser(ctx, p.Account, user, pass, p.Publish, p.Subscribe)
}
