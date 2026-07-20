package modules

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

func Parameters(req Request) (map[string]any, error) {
	out := map[string]any{}
	if req.Claim == nil || len(req.Claim.Spec.Parameters.Raw) == 0 {
		return out, nil
	}
	if err := json.Unmarshal(req.Claim.Spec.Parameters.Raw, &out); err != nil {
		return nil, fmt.Errorf("decode claim parameters: %w", err)
	}
	return out, nil
}

func StringParameter(params map[string]any, key, fallback string) string {
	if value, ok := params[key].(string); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return fallback
}

func ProviderEnv(req Request, key, fallback string) string {
	if value := strings.TrimSpace(req.Provider.Spec.Outputs.Env[key]); value != "" {
		return value
	}
	return fallback
}

func RandomPassword(bytes int) ([]byte, error) {
	buf := make([]byte, bytes)
	if _, err := rand.Read(buf); err != nil {
		return nil, fmt.Errorf("generate password: %w", err)
	}
	encoded := make([]byte, base64.RawURLEncoding.EncodedLen(len(buf)))
	base64.RawURLEncoding.Encode(encoded, buf)
	return encoded, nil
}

func ClaimBaseName(req Request) string {
	if req.Claim != nil && req.Claim.Name != "" {
		return req.Claim.Name
	}
	if req.Workload != "" {
		return req.Workload
	}
	return "consumer"
}
