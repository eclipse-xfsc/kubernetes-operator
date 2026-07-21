package cassandra

import (
	"context"
	"fmt"

	"github.com/eclipse-xfsc/kubernetes-operator/internal/modules"
)

type Parameters struct {
	Keyspace          string `json:"keyspace,omitempty"`
	Username          string `json:"username,omitempty"`
	ReplicationFactor int    `json:"replicationFactor,omitempty"`
	Datacenter        string `json:"datacenter,omitempty"`
}

type CassandraBackend struct {
	factory func(modules.ProvisionRequest) (*Client, error)
}

func NewBackend() *CassandraBackend {
	return &CassandraBackend{factory: func(req modules.ProvisionRequest) (*Client, error) { return NewClient(req.AdminSecret) }}
}
func (b *CassandraBackend) Provision(ctx context.Context, req modules.ProvisionRequest) error {
	var p Parameters
	if err := modules.DecodeParameters(req.Claim, &p); err != nil {
		return err
	}
	if p.Keyspace == "" {
		p.Keyspace = req.Claim.Name
	}
	if p.ReplicationFactor <= 0 {
		p.ReplicationFactor = 3
	}
	username, err := modules.SecretString(req.ClaimSecret, "username", "user")
	if err != nil {
		return err
	}
	password, err := modules.SecretString(req.ClaimSecret, "password")
	if err != nil {
		return err
	}
	c, err := b.factory(req)
	if err != nil {
		return fmt.Errorf("create cassandra client: %w", err)
	}
	defer c.Close()
	if err = c.EnsureRole(ctx, username, password); err != nil {
		return fmt.Errorf("ensure cassandra role: %w", err)
	}
	if err = c.EnsureKeyspace(ctx, p.Keyspace, p.Datacenter, p.ReplicationFactor); err != nil {
		return fmt.Errorf("ensure cassandra keyspace: %w", err)
	}
	return c.GrantAllOnKeyspace(ctx, p.Keyspace, username)
}
