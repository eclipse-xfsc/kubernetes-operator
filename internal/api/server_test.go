package api

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/eclipse-xfsc/kubernetes-operator/internal/index"
	"github.com/eclipse-xfsc/kubernetes-operator/internal/logging"
	"github.com/eclipse-xfsc/kubernetes-operator/internal/registry"
	"github.com/eclipse-xfsc/kubernetes-operator/internal/runtimeinfo"

	telemetrymodule "github.com/eclipse-xfsc/kubernetes-operator/internal/modules/telemetry"
)

func TestVersionEndpoint(t *testing.T) {
	srv := NewServer(ServerConfig{
		Address: ":0",
		Version: runtimeinfo.Info{
			OperatorVersion: "test-version",
			GitCommit:       "test-commit",
			BuildDate:       "test-date",
		},
		Inventory: index.NewInventory(),
		Registry:  registry.New(),
		Logger:    logging.New(),
	})

	req := httptest.NewRequest("GET", "/version", nil)
	rec := httptest.NewRecorder()

	srv.Handler.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var body runtimeinfo.Info
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}

	if body.OperatorVersion != "test-version" {
		t.Fatalf("expected test-version, got %s", body.OperatorVersion)
	}
}

func TestModulesEndpoint(t *testing.T) {
	reg := registry.New()
	reg.MustRegister(telemetrymodule.New())

	srv := NewServer(ServerConfig{
		Address:   ":0",
		Version:   runtimeinfo.Info{},
		Inventory: index.NewInventory(),
		Registry:  reg,
		Logger:    logging.New(),
	})

	req := httptest.NewRequest("GET", "/modules", nil)
	rec := httptest.NewRecorder()

	srv.Handler.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var body []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}

	if len(body) != 1 {
		t.Fatalf("expected 1 module, got %d", len(body))
	}

	if body[0]["name"] != "telemetry" {
		t.Fatalf("expected telemetry module, got %#v", body[0]["name"])
	}
}
