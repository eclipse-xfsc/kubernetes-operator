package cassandra

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	resourcesv1alpha1 "github.com/eclipse-xfsc/kubernetes-operator/api/v1alpha1"
	"github.com/eclipse-xfsc/kubernetes-operator/internal/modules"
	"github.com/gocql/gocql"
	corev1 "k8s.io/api/core/v1"
)

type Client struct{ session *gocql.Session }

func NewClient(provider resourcesv1alpha1.ResourceProvider, secret corev1.Secret) (*Client, error) {
	hosts, err := modules.ProviderRequiredEnv(provider, "CASSANDRA_HOSTS", "CASSANDRA_HOST")
	if err != nil {
		return nil, err
	}
	cluster := gocql.NewCluster(splitHosts(hosts)...)
	if port := modules.ProviderEnv(provider, "CASSANDRA_PORT"); port != "" {
		v, err := strconv.Atoi(port)
		if err != nil {
			return nil, fmt.Errorf("invalid cassandra port %q: %w", port, err)
		}
		cluster.Port = v
	}
	user := modules.OptionalSecretString(secret, "username", "user")
	pass := modules.OptionalSecretString(secret, "password")
	if user != "" || pass != "" {
		cluster.Authenticator = gocql.PasswordAuthenticator{Username: user, Password: pass}
	}
	cluster.Consistency = gocql.Quorum
	session, err := cluster.CreateSession()
	if err != nil {
		return nil, fmt.Errorf("connect cassandra: %w", err)
	}
	return &Client{session: session}, nil
}
func splitHosts(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
func quoteIdentifier(v string) string { return `"` + strings.ReplaceAll(v, `"`, `""`) + `"` }
func quoteLiteral(v string) string    { return `'` + strings.ReplaceAll(v, `'`, `''`) + `'` }
func (c *Client) Close() {
	if c != nil && c.session != nil {
		c.session.Close()
	}
}
func (c *Client) EnsureRole(ctx context.Context, user, password string) error {
	if err := c.session.Query(fmt.Sprintf("CREATE ROLE IF NOT EXISTS %s WITH PASSWORD = %s AND LOGIN = true", quoteIdentifier(user), quoteLiteral(password))).WithContext(ctx).Exec(); err != nil {
		return err
	}
	// CREATE ROLE IF NOT EXISTS does not update credentials of an existing role.
	// Always reconcile the password to the value delivered by ESO/OpenBao.
	return c.session.Query(fmt.Sprintf("ALTER ROLE %s WITH PASSWORD = %s AND LOGIN = true", quoteIdentifier(user), quoteLiteral(password))).WithContext(ctx).Exec()
}
func (c *Client) EnsureKeyspace(ctx context.Context, keyspace, dc string, rf int) error {
	strategy := fmt.Sprintf("{'class':'SimpleStrategy','replication_factor':%d}", rf)
	if dc != "" {
		strategy = fmt.Sprintf("{'class':'NetworkTopologyStrategy',%s:%d}", quoteLiteral(dc), rf)
	}
	return c.session.Query(fmt.Sprintf("CREATE KEYSPACE IF NOT EXISTS %s WITH replication = %s", quoteIdentifier(keyspace), strategy)).WithContext(ctx).Exec()
}
func (c *Client) GrantAllOnKeyspace(ctx context.Context, keyspace, user string) error {
	return c.session.Query(fmt.Sprintf("GRANT ALL PERMISSIONS ON KEYSPACE %s TO %s", quoteIdentifier(keyspace), quoteIdentifier(user))).WithContext(ctx).Exec()
}
