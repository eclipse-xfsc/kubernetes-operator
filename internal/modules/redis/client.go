package redis

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/eclipse-xfsc/kubernetes-operator/internal/modules"
	corev1 "k8s.io/api/core/v1"
)

type Client struct {
	conn   net.Conn
	reader *bufio.Reader
}

func NewClient(secret corev1.Secret) (*Client, error) {
	host, err := modules.SecretString(secret, "host", "hostname")
	if err != nil {
		return nil, err
	}
	port := modules.OptionalSecretString(secret, "port")
	if port == "" {
		port = "6379"
	}
	address := net.JoinHostPort(host, port)
	dialer := net.Dialer{Timeout: 10 * time.Second}
	var conn net.Conn
	if strings.EqualFold(modules.OptionalSecretString(secret, "tls"), "true") {
		conn, err = tls.DialWithDialer(&dialer, "tcp", address, &tls.Config{MinVersion: tls.VersionTLS12, ServerName: host})
	} else {
		conn, err = dialer.Dial("tcp", address)
	}
	if err != nil {
		return nil, fmt.Errorf("connect to redis %s: %w", address, err)
	}
	c := &Client{conn: conn, reader: bufio.NewReader(conn)}
	username := modules.OptionalSecretString(secret, "username", "user")
	password := modules.OptionalSecretString(secret, "password")
	if password != "" {
		args := []string{"AUTH"}
		if username != "" {
			args = append(args, username)
		}
		args = append(args, password)
		if _, err := c.command(context.Background(), args...); err != nil {
			c.Close()
			return nil, fmt.Errorf("redis authentication failed: %w", err)
		}
	}
	if _, err := c.command(context.Background(), "PING"); err != nil {
		c.Close()
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}
	return c, nil
}

func (c *Client) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}
func (c *Client) EnsureUser(ctx context.Context, username string, rules []string) error {
	args := append([]string{"ACL", "SETUSER", username}, rules...)
	_, err := c.command(ctx, args...)
	return err
}
func (c *Client) command(ctx context.Context, args ...string) (string, error) {
	if deadline, ok := ctx.Deadline(); ok {
		_ = c.conn.SetDeadline(deadline)
	} else {
		_ = c.conn.SetDeadline(time.Now().Add(15 * time.Second))
	}
	var b strings.Builder
	b.WriteString("*" + strconv.Itoa(len(args)) + "\r\n")
	for _, arg := range args {
		b.WriteString("$" + strconv.Itoa(len(arg)) + "\r\n" + arg + "\r\n")
	}
	if _, err := io.WriteString(c.conn, b.String()); err != nil {
		return "", err
	}
	line, err := c.reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	line = strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
	if strings.HasPrefix(line, "-") {
		return "", fmt.Errorf("redis error: %s", strings.TrimPrefix(line, "-"))
	}
	if strings.HasPrefix(line, "+") {
		return strings.TrimPrefix(line, "+"), nil
	}
	if strings.HasPrefix(line, ":") {
		return strings.TrimPrefix(line, ":"), nil
	}
	return line, nil
}
