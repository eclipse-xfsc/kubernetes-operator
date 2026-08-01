package s3

import (
	"context"
	resourcesv1alpha1 "github.com/eclipse-xfsc/kubernetes-operator/api/v1alpha1"
	"github.com/eclipse-xfsc/kubernetes-operator/internal/modules"
	corev1 "k8s.io/api/core/v1"
)

type Client struct {
	alias  string
	runner modules.CommandRunner
}

func NewClient(provider resourcesv1alpha1.ResourceProvider, secret corev1.Secret, runner modules.CommandRunner) (*Client, error) {
	endpoint, err := modules.ProviderRequiredEnv(provider, "S3_ENDPOINT", "MINIO_ENDPOINT")
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
	alias := modules.ProviderEnv(provider, "S3_ALIAS", "MINIO_ALIAS")
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
