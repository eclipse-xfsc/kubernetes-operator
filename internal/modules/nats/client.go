package nats

import (
	"context"
	"github.com/eclipse-xfsc/kubernetes-operator/internal/modules"
	corev1 "k8s.io/api/core/v1"
)

type Client struct {
	operator string
	runner   modules.CommandRunner
}

func NewClient(secret corev1.Secret, runner modules.CommandRunner) (*Client, error) {
	operator, err := modules.SecretString(secret, "operator")
	if err != nil {
		return nil, err
	}
	if runner == nil {
		runner = modules.ExecRunner{}
	}
	return &Client{operator: operator, runner: runner}, nil
}
func (c *Client) EnsureAccount(ctx context.Context, account string) error {
	return c.runner.Run(ctx, "nsc", []string{"add", "account", account, "--operator", c.operator}, nil)
}
func (c *Client) EnsureUser(ctx context.Context, account, user, password string, pub, sub []string) error {
	args := []string{"add", "user", user, "--operator", c.operator, "--account", account, "--password", password}
	for _, v := range pub {
		args = append(args, "--allow-pub", v)
	}
	for _, v := range sub {
		args = append(args, "--allow-sub", v)
	}
	return c.runner.Run(ctx, "nsc", args, nil)
}
