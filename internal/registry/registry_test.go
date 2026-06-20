package registry

import (
	"testing"

	postgresmodule "github.com/eclipse-xfsc/kubernetes-operator/internal/modules/postgres"
	telemetrymodule "github.com/eclipse-xfsc/kubernetes-operator/internal/modules/telemetry"
)

func TestRegistryListsTypes(t *testing.T) {
	reg := New()
	reg.MustRegister(
		telemetrymodule.New(),
		postgresmodule.New(),
	)

	got := reg.Types()

	assertContains(t, got, "telemetry")
	assertContains(t, got, "database.postgres")
}

func TestRegistryFindsModuleByType(t *testing.T) {
	reg := New()
	reg.MustRegister(telemetrymodule.New())

	mod, ok := reg.ForType("telemetry")
	if !ok {
		t.Fatal("expected telemetry module")
	}

	if mod.Name() != "telemetry" {
		t.Fatalf("expected telemetry module, got %s", mod.Name())
	}
}

func assertContains(t *testing.T, values []string, expected string) {
	t.Helper()

	for _, v := range values {
		if v == expected {
			return
		}
	}

	t.Fatalf("expected %#v to contain %q", values, expected)
}
