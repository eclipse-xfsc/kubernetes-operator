package config

import (
	"errors"
	"os"

	"sigs.k8s.io/yaml"
)

type OperatorConfig struct {
	ExcludedNamespaces []string                `json:"excludedNamespaces,omitempty" yaml:"excludedNamespaces,omitempty"`
	ResourceTypes      map[string]ResourceType `json:"resourceTypes,omitempty" yaml:"resourceTypes,omitempty"`
}
type ResourceType struct {
	Module      string            `json:"module,omitempty" yaml:"module,omitempty"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`
	Env         map[string]EnvDef `json:"env,omitempty" yaml:"env,omitempty"`
}
type EnvDef struct {
	Required bool   `json:"required,omitempty" yaml:"required,omitempty"`
	Default  string `json:"default,omitempty" yaml:"default,omitempty"`
	Secret   bool   `json:"secret,omitempty" yaml:"secret,omitempty"`
}

func LoadOrDefault(path string) (*OperatorConfig, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Default(), nil
		}
		return nil, err
	}
	var cfg OperatorConfig
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
func Default() *OperatorConfig {
	return &OperatorConfig{ExcludedNamespaces: []string{"kube-system", "kube-public", "kube-node-lease"}, ResourceTypes: map[string]ResourceType{"telemetry": {Module: "telemetry", Description: "OpenTelemetry env config"}, "database.postgres": {Module: "postgres", Description: "PostgreSQL connection"}, "cache.redis": {Module: "redis", Description: "Redis connection"}, "objectstore.s3": {Module: "s3", Description: "S3 object storage"}, "vault": {Module: "vault", Description: "OpenBao/Vault access"}}}
}
