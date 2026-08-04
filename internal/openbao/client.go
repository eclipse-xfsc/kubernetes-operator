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
	return &Client{cfg: cfg, http: &http.Client{Timeout: 20 * time.Second}}
}

func (c *Client) Enabled() bool {
	return strings.TrimSpace(c.cfg.Address) != "" && strings.TrimSpace(c.cfg.Role) != ""
}

func (c *Client) EnsureCredentials(ctx context.Context, remoteKey, username string) error {
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
	payload := map[string]any{"data": map[string]string{"username": username, "password": password}}
	return c.request(ctx, http.MethodPost, c.kvURL(engine, "data", secretPath), token, payload, nil, http.StatusOK, http.StatusNoContent)
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
	err = c.request(ctx, http.MethodDelete, c.kvURL(engine, "metadata", secretPath), token, nil, nil, http.StatusNoContent, http.StatusOK, http.StatusNotFound)
	return err
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
	url := strings.TrimRight(c.cfg.Address, "/") + "/v1/auth/" + strings.Trim(c.cfg.AuthMount, "/") + "/login"
	if err := c.request(ctx, http.MethodPost, url, "", map[string]string{"role": c.cfg.Role, "jwt": strings.TrimSpace(string(jwt))}, &response, http.StatusOK); err != nil {
		return "", err
	}
	if response.Auth.ClientToken == "" {
		return "", fmt.Errorf("OpenBao login returned no client token")
	}
	return response.Auth.ClientToken, nil
}

func (c *Client) ensureKVEngine(ctx context.Context, token, engine string) error {
	url := strings.TrimRight(c.cfg.Address, "/") + "/v1/sys/mounts/" + strings.Trim(engine, "/")
	var current map[string]any
	err := c.request(ctx, http.MethodGet, url, token, nil, &current, http.StatusOK, http.StatusNotFound)
	if err != nil {
		return err
	}
	if len(current) > 0 {
		return nil
	}
	return c.request(ctx, http.MethodPost, url, token, map[string]any{"type": "kv", "options": map[string]string{"version": "2"}}, nil, http.StatusNoContent, http.StatusOK)
}

func (c *Client) exists(ctx context.Context, token, engine, secretPath string) (bool, error) {
	var response map[string]any
	err := c.request(ctx, http.MethodGet, c.kvURL(engine, "data", secretPath), token, nil, &response, http.StatusOK, http.StatusNotFound)
	if err != nil {
		return false, err
	}
	return len(response) > 0, nil
}

func (c *Client) kvURL(engine, part, secretPath string) string {
	clean := strings.TrimPrefix(path.Clean("/"+secretPath), "/")
	return strings.TrimRight(c.cfg.Address, "/") + "/v1/" + strings.Trim(engine, "/") + "/" + part + "/" + clean
}

func splitRemoteKey(remoteKey string) (string, string, error) {
	clean := strings.TrimPrefix(path.Clean("/"+strings.TrimSpace(remoteKey)), "/")
	parts := strings.SplitN(clean, "/", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", fmt.Errorf("remote key %q must use <engine>/<secret-path>", remoteKey)
	}
	return parts[0], parts[1], nil
}

func (c *Client) request(ctx context.Context, method, url, token string, body any, out any, expected ...int) error {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return err
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
		return fmt.Errorf("OpenBao %s %s: %w", method, url, err)
	}
	defer resp.Body.Close()
	payload, _ := io.ReadAll(resp.Body)
	ok := false
	for _, code := range expected {
		if resp.StatusCode == code {
			ok = true
			break
		}
	}
	if !ok {
		return fmt.Errorf("OpenBao %s %s returned %d: %s", method, url, resp.StatusCode, strings.TrimSpace(string(payload)))
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if out != nil && len(payload) > 0 {
		if err := json.Unmarshal(payload, out); err != nil {
			return err
		}
	}
	return nil
}

func randomPassword(size int) (string, error) {
	data := make([]byte, size)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(data)[:size], nil
}
