package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadOrDefaultReturnsDefaultIfFileMissing(t *testing.T) {
	cfg, err := LoadOrDefault("/does/not/exist/config.yaml")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.ResourceTypes["telemetry"].Module != "telemetry" {
		t.Fatalf("expected telemetry module")
	}
}

func TestLoadOrDefaultReadsConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	content := `
resourceTypes:
  telemetry:
    module: telemetry
    description: test telemetry
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadOrDefault(path)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	got := cfg.ResourceTypes["telemetry"].Description
	if got != "test telemetry" {
		t.Fatalf("expected custom config, got %q", got)
	}
}
