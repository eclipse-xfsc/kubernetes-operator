package index

import (
	"testing"

	"github.com/eclipse-xfsc/kubernetes-operator/internal/modules"
)

func TestInventoryProvidersSortedByType(t *testing.T) {
	inv := NewInventory()

	inv.UpsertProvider(modules.Provider{
		Type:      "database.postgres",
		Name:      "main",
		Namespace: "database",
	})

	inv.UpsertProvider(modules.Provider{
		Type:      "telemetry",
		Name:      "default",
		Namespace: "observability",
	})

	got := inv.Providers()

	if len(got) != 2 {
		t.Fatalf("expected 2 providers, got %d", len(got))
	}

	if got[0].Type != "database.postgres" {
		t.Fatalf("expected postgres first, got %s", got[0].Type)
	}
}

func TestAccountsByConsumer(t *testing.T) {
	inv := NewInventory()

	inv.UpsertAccount(modules.Account{
		Name:              "db-user",
		Namespace:         "app",
		Type:              "database.postgres",
		ConsumerNamespace: "app",
		ConsumerName:      "example-api",
	})

	inv.UpsertAccount(modules.Account{
		Name:              "other-user",
		Namespace:         "other",
		Type:              "database.postgres",
		ConsumerNamespace: "other",
		ConsumerName:      "other-api",
	})

	got := inv.AccountsByConsumer("app", "example-api")

	if len(got) != 1 {
		t.Fatalf("expected 1 account, got %d", len(got))
	}

	if got[0].Name != "db-user" {
		t.Fatalf("expected db-user, got %s", got[0].Name)
	}
}
