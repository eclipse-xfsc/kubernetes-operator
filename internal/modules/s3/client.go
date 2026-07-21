package s3

import (
	"context"
	"github.com/eclipse-xfsc/kubernetes-operator/internal/modules"
	corev1 "k8s.io/api/core/v1"
)

type Client struct {
	alias  string
	runner modules.CommandRunner
}

func NewClient(secret corev1.Secret, runner modules.CommandRunner) (*Client, error) {
	endpoint, err := modules.SecretString(secret, "endpoint", "url", "host")
	if err != nil {
		return nil, err
	}
	user, err := modules.SecretString(secret, "username", "accessKey", "accessKeyId")
	if err != nil {
		return nil, err
	}
	pass, err := modules.SecretString(secret, "password", "secretKey", "secretAccessKey")
	if err != nil {
		return nil, err
	}
	alias := modules.OptionalSecretString(secret, "alias")
	if alias == "" {
		alias = "xfsc-admin"
	}
	if runner == nil {
		runner = modules.ExecRunner{}
	}
	c := &Client{alias: alias, runner: runner}
	if err := runner.Run(context.Background(), "mc", []string{"alias", "set", alias, endpoint, user, pass}, nil); err != nil {
		return nil, err
	}
	return c, nil
}
func (c *Client) EnsureBucket(ctx context.Context, bucket string) error {
	return c.runner.Run(ctx, "mc", []string{"mb", "--ignore-existing", c.alias + "/" + bucket}, nil)
}
func (c *Client) EnsureUser(ctx context.Context, user, password string) error {
	return c.runner.Run(ctx, "mc", []string{"admin", "user", "add", c.alias, user, password}, nil)
}
func (c *Client) AttachPolicy(ctx context.Context, user, policy string) error {
	return c.runner.Run(ctx, "mc", []string{"admin", "policy", "attach", c.alias, policy, "--user", user}, nil)
}
