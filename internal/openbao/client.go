package openbao

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
)

type Config struct {
	Address   string
	AuthMount string
	Role      string
	TokenFile string
}

type Client struct {
	cfg  Config
	http *http.Client
}

func New(cfg Config) *Client {
	if cfg.AuthMount == "" {
		cfg.AuthMount = "kubernetes"
	}
	if cfg.TokenFile == "" {
		cfg.TokenFile = "/var/run/secrets/kubernetes.io/serviceaccount/token"
	}
	return &Client{
		cfg:  cfg,
		http: &http.Client{Timeout: 20 * time.Second},
	}
}

func (c *Client) Enabled() bool {
	return strings.TrimSpace(c.cfg.Address) != "" &&
		strings.TrimSpace(c.cfg.Role) != ""
}

func (c *Client) EnsureCredentials(
	ctx context.Context,
	remoteKey string,
	username string,
) error {
	engine, secretPath, err := splitRemoteKey(remoteKey)
	if err != nil {
		return err
	}

	token, err := c.login(ctx)
	if err != nil {
		return err
	}

	if err := c.ensureKVEngine(ctx, token, engine); err != nil {
		return err
	}

	found, err := c.exists(ctx, token, engine, secretPath)
	if err != nil {
		return err
	}
	if found {
		return nil
	}

	password, err := randomPassword(48)
	if err != nil {
		return err
	}

	payload := map[string]any{
		"data": map[string]string{
			"username": username,
			"password": password,
		},
	}

	return c.request(
		ctx,
		http.MethodPost,
		c.kvURL(engine, "data", secretPath),
		token,
		payload,
		nil,
		http.StatusOK,
		http.StatusNoContent,
	)
}

func (c *Client) Delete(ctx context.Context, remoteKey string) error {
	engine, secretPath, err := splitRemoteKey(remoteKey)
	if err != nil {
		return err
	}

	token, err := c.login(ctx)
	if err != nil {
		return err
	}

	return c.request(
		ctx,
		http.MethodDelete,
		c.kvURL(engine, "metadata", secretPath),
		token,
		nil,
		nil,
		http.StatusNoContent,
		http.StatusOK,
		http.StatusNotFound,
	)
}

func (c *Client) login(ctx context.Context) (string, error) {
	jwt, err := os.ReadFile(c.cfg.TokenFile)
	if err != nil {
		return "", fmt.Errorf("read service account token: %w", err)
	}

	var response struct {
		Auth struct {
			ClientToken string `json:"client_token"`
		} `json:"auth"`
	}

	url := strings.TrimRight(c.cfg.Address, "/") +
		"/v1/auth/" +
		strings.Trim(c.cfg.AuthMount, "/") +
		"/login"

	err = c.request(
		ctx,
		http.MethodPost,
		url,
		"",
		map[string]string{
			"role": c.cfg.Role,
			"jwt":  strings.TrimSpace(string(jwt)),
		},
		&response,
		http.StatusOK,
	)
	if err != nil {
		return "", err
	}

	if response.Auth.ClientToken == "" {
		return "", fmt.Errorf("OpenBao login returned no client token")
	}

	return response.Auth.ClientToken, nil
}

func (c *Client) ensureKVEngine(
	ctx context.Context,
	token string,
	engine string,
) error {
	url := strings.TrimRight(c.cfg.Address, "/") +
		"/v1/sys/mounts/" +
		strings.Trim(engine, "/")

	status, payload, err := c.rawRequest(
		ctx,
		http.MethodGet,
		url,
		token,
		nil,
	)
	if err != nil {
		return err
	}

	switch status {
	case http.StatusOK:
		return nil

	case http.StatusNotFound:
		// Mount does not exist.
		return c.createKVEngine(ctx, token, url)

	case http.StatusBadRequest:
		// OpenBao returns 400 for a missing mount on this endpoint.
		if strings.Contains(
			strings.ToLower(string(payload)),
			"no secret engine mount",
		) {
			return c.createKVEngine(ctx, token, url)
		}

		return fmt.Errorf(
			"OpenBao GET %s returned %d: %s",
			url,
			status,
			strings.TrimSpace(string(payload)),
		)

	default:
		return fmt.Errorf(
			"OpenBao GET %s returned %d: %s",
			url,
			status,
			strings.TrimSpace(string(payload)),
		)
	}
}

func (c *Client) createKVEngine(
	ctx context.Context,
	token string,
	url string,
) error {
	return c.request(
		ctx,
		http.MethodPost,
		url,
		token,
		map[string]any{
			"type": "kv",
			"options": map[string]string{
				"version": "2",
			},
		},
		nil,
		http.StatusNoContent,
		http.StatusOK,
	)
}

func (c *Client) exists(
	ctx context.Context,
	token string,
	engine string,
	secretPath string,
) (bool, error) {
	status, payload, err := c.rawRequest(
		ctx,
		http.MethodGet,
		c.kvURL(engine, "data", secretPath),
		token,
		nil,
	)
	if err != nil {
		return false, err
	}

	switch status {
	case http.StatusOK:
		return len(payload) > 0, nil

	case http.StatusNotFound:
		return false, nil

	default:
		return false, fmt.Errorf(
			"OpenBao GET %s returned %d: %s",
			c.kvURL(engine, "data", secretPath),
			status,
			strings.TrimSpace(string(payload)),
		)
	}
}

func (c *Client) kvURL(engine, part, secretPath string) string {
	clean := strings.TrimPrefix(
		path.Clean("/"+secretPath),
		"/",
	)

	return strings.TrimRight(c.cfg.Address, "/") +
		"/v1/" +
		strings.Trim(engine, "/") +
		"/" +
		part +
		"/" +
		clean
}

func splitRemoteKey(remoteKey string) (string, string, error) {
	clean := strings.TrimPrefix(
		path.Clean("/"+strings.TrimSpace(remoteKey)),
		"/",
	)

	lastSlash := strings.LastIndex(clean, "/")
	if lastSlash <= 0 || lastSlash == len(clean)-1 {
		return "", "", fmt.Errorf(
			"remote key %q must use <engine-path>/<secret-name>",
			remoteKey,
		)
	}

	engine := clean[:lastSlash]
	secretPath := clean[lastSlash+1:]

	return engine, secretPath, nil
}

func (c *Client) request(
	ctx context.Context,
	method string,
	url string,
	token string,
	body any,
	out any,
	expected ...int,
) error {
	status, payload, err := c.rawRequest(
		ctx,
		method,
		url,
		token,
		body,
	)
	if err != nil {
		return err
	}

	for _, code := range expected {
		if status == code {
			if out != nil && len(payload) > 0 {
				if err := json.Unmarshal(payload, out); err != nil {
					return fmt.Errorf(
						"decode OpenBao response from %s: %w",
						url,
						err,
					)
				}
			}
			return nil
		}
	}

	return fmt.Errorf(
		"OpenBao %s %s returned %d: %s",
		method,
		url,
		status,
		strings.TrimSpace(string(payload)),
	)
}

func (c *Client) rawRequest(
	ctx context.Context,
	method string,
	url string,
	token string,
	body any,
) (int, []byte, error) {
	var reader io.Reader

	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return 0, nil, fmt.Errorf(
				"encode OpenBao request body: %w",
				err,
			)
		}
		reader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		method,
		url,
		reader,
	)
	if err != nil {
		return 0, nil, err
	}

	req.Header.Set("Accept", "application/json")

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if token != "" {
		req.Header.Set("X-Vault-Token", token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf(
			"OpenBao %s %s: %w",
			method,
			url,
			err,
		)
	}
	defer resp.Body.Close()

	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, fmt.Errorf(
			"read OpenBao response from %s: %w",
			url,
			err,
		)
	}

	return resp.StatusCode, payload, nil
}

func randomPassword(size int) (string, error) {
	if size <= 0 {
		return "", fmt.Errorf("password size must be greater than zero")
	}

	data := make([]byte, size)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}

	encoded := base64.RawURLEncoding.EncodeToString(data)
	if len(encoded) < size {
		return "", fmt.Errorf("generated password is shorter than requested size")
	}

	return encoded[:size], nil
}
