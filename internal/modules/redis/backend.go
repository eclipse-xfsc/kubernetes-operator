package redis

import (
	"context"
	"fmt"

	"github.com/eclipse-xfsc/kubernetes-operator/internal/modules"
)

type ACLParameters struct {
	Keys     []string `json:"keys,omitempty"`
	Channels []string `json:"channels,omitempty"`
	Commands []string `json:"commands,omitempty"`
}

type Parameters struct {
	Database int           `json:"database,omitempty"`
	ACL      ACLParameters `json:"acl,omitempty"`
}

type RedisBackend struct {
	factory func(modules.ProvisionRequest) (*Client, error)
}

func NewBackend() *RedisBackend {
	return &RedisBackend{factory: func(req modules.ProvisionRequest) (*Client, error) {
		return NewClient(req.AdminSecret)
	}}
}

func (b *RedisBackend) Provision(ctx context.Context, req modules.ProvisionRequest) error {
	username, err := modules.SecretString(req.ClaimSecret, "username", "user")
	if err != nil {
		return err
	}
	password, err := modules.SecretString(req.ClaimSecret, "password")
	if err != nil {
		return err
	}
	var parameters Parameters
	if err := modules.DecodeParameters(req.Claim, &parameters); err != nil {
		return err
	}
	client, err := b.factory(req)
	if err != nil {
		return fmt.Errorf("create redis client: %w", err)
	}
	defer client.Close()
	rules := []string{"reset", "on", ">" + password}
	if len(parameters.ACL.Keys) == 0 {
		rules = append(rules, "~*")
	}
	for _, key := range parameters.ACL.Keys {
		rules = append(rules, "~"+key)
	}
	for _, channel := range parameters.ACL.Channels {
		rules = append(rules, "&"+channel)
	}
	if len(parameters.ACL.Commands) == 0 {
		rules = append(rules, "+@all")
	}
	rules = append(rules, parameters.ACL.Commands...)
	if err := client.EnsureUser(ctx, username, rules); err != nil {
		return fmt.Errorf("ensure redis user %q: %w", username, err)
	}
	return nil
}
