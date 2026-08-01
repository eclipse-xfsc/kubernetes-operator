package config

import (
	"fmt"
	"os"

	"sigs.k8s.io/yaml"
)

type Module struct {
	Enabled *bool `json:"enabled,omitempty" yaml:"enabled,omitempty"`
}

type Modules struct {
	Postgres  Module `json:"postgres,omitempty" yaml:"postgres,omitempty"`
	Redis     Module `json:"redis,omitempty" yaml:"redis,omitempty"`
	Cassandra Module `json:"cassandra,omitempty" yaml:"cassandra,omitempty"`
	NATS      Module `json:"nats,omitempty" yaml:"nats,omitempty"`
	S3        Module `json:"s3,omitempty" yaml:"s3,omitempty"`
	Vault     Module `json:"vault,omitempty" yaml:"vault,omitempty"`
}

func LoadModules(path string) (Modules, error) {
	cfg := defaultModules()
	if path == "" {
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return Modules{}, fmt.Errorf("read module config %q: %w", path, err)
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Modules{}, fmt.Errorf("parse module config %q: %w", path, err)
	}
	return cfg, nil
}

func (m Module) IsEnabled() bool {
	return m.Enabled == nil || *m.Enabled
}

func defaultModules() Modules {
	postgres := true
	redis := true
	cassandra := true
	nats := true
	s3 := true
	vault := true

	return Modules{
		Postgres:  Module{Enabled: &postgres},
		Redis:     Module{Enabled: &redis},
		Cassandra: Module{Enabled: &cassandra},
		NATS:      Module{Enabled: &nats},
		S3:        Module{Enabled: &s3},
		Vault:     Module{Enabled: &vault},
	}
}
