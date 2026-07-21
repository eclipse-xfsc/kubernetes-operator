package postgres

import (
	"context"
	"fmt"
	"github.com/eclipse-xfsc/kubernetes-operator/internal/modules"
	corev1 "k8s.io/api/core/v1"
	"strings"
)

type Client struct {
	host, port, user, password, sslmode string
	runner                              modules.CommandRunner
}

func NewClient(secret corev1.Secret, runner modules.CommandRunner) (*Client, error) {
	host, err := modules.SecretString(secret, "host", "hostname")
	if err != nil {
		return nil, err
	}
	port := modules.OptionalSecretString(secret, "port")
	if port == "" {
		port = "5432"
	}
	user, err := modules.SecretString(secret, "username", "user")
	if err != nil {
		return nil, err
	}
	pass, err := modules.SecretString(secret, "password")
	if err != nil {
		return nil, err
	}
	ssl := modules.OptionalSecretString(secret, "sslmode")
	if ssl == "" {
		ssl = "disable"
	}
	if runner == nil {
		runner = modules.ExecRunner{}
	}
	return &Client{host: host, port: port, user: user, password: pass, sslmode: ssl, runner: runner}, nil
}
func qi(v string) string { return `"` + strings.ReplaceAll(v, `"`, `""`) + `"` }
func ql(v string) string { return `'` + strings.ReplaceAll(v, `'`, `''`) + `'` }
func (c *Client) exec(ctx context.Context, db, sql string) error {
	return c.runner.Run(ctx, "psql", []string{"-v", "ON_ERROR_STOP=1", "-h", c.host, "-p", c.port, "-U", c.user, "-d", db, "-c", sql}, []string{"PGPASSWORD=" + c.password, "PGSSLMODE=" + c.sslmode})
}
func (c *Client) EnsureRole(ctx context.Context, user, password string) error {
	return c.exec(ctx, "postgres", fmt.Sprintf("DO $$ BEGIN IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname=%s) THEN CREATE ROLE %s LOGIN PASSWORD %s; ELSE ALTER ROLE %s LOGIN PASSWORD %s; END IF; END $$;", ql(user), qi(user), ql(password), qi(user), ql(password)))
}
func (c *Client) EnsureDatabase(ctx context.Context, name, owner string) error {
	sql := fmt.Sprintf("SELECT 'CREATE DATABASE %s OWNER %s' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname=%s)\\gexec", qi(name), qi(owner), ql(name))
	return c.exec(ctx, "postgres", sql)
}
func (c *Client) EnsureDatabaseObjects(ctx context.Context, db, owner string, schemas, extensions []string) error {
	for _, s := range schemas {
		if err := c.exec(ctx, db, fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s AUTHORIZATION %s", qi(s), qi(owner))); err != nil {
			return err
		}
	}
	for _, e := range extensions {
		if err := c.exec(ctx, db, fmt.Sprintf("CREATE EXTENSION IF NOT EXISTS %s", qi(e))); err != nil {
			return err
		}
	}
	return nil
}
