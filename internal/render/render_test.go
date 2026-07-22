package render

import "testing"

func TestTemplateRendersGoTemplateFields(t *testing.T) {
	ctx := Context{
		Namespace: "wallet",
		Workload:  "api",
		Type:      "postgres",
		Provider:  "postgres-main",
		Tenant:    "tenant-a",
	}

	got := Template("{{ .Namespace }}/{{ .Workload }}/{{ .Type }}/{{ .Provider }}/{{ .Tenant }}", ctx)
	want := "wallet/api/postgres/postgres-main/tenant-a"
	if got != want {
		t.Fatalf("Template() = %q, want %q", got, want)
	}
}

func TestTemplateKeepsLegacySyntaxCompatible(t *testing.T) {
	ctx := Context{Namespace: "wallet", Workload: "api"}
	got := Template("{{ namespace }}/{{ workload }}/{{ consumer.namespace }}/{{ consumer.name }}", ctx)
	want := "wallet/api/wallet/api"
	if got != want {
		t.Fatalf("Template() = %q, want %q", got, want)
	}
}

func TestTemplateRendersMultilineJobYAML(t *testing.T) {
	ctx := Context{Namespace: "wallet", Workload: "api"}
	input := "apiVersion: batch/v1\nkind: Job\nmetadata:\n  name: migrate-{{ .Workload }}\n  namespace: {{ .Namespace }}\n"
	got := Template(input, ctx)
	want := "apiVersion: batch/v1\nkind: Job\nmetadata:\n  name: migrate-api\n  namespace: wallet\n"
	if got != want {
		t.Fatalf("Template() = %q, want %q", got, want)
	}
}

func TestRenderExactProviderContext(t *testing.T) {
	ctx := Context{
		Namespace: "wallet",
		Workload:  "wallet-api",
		Type:      "redis",
		Provider:  "redis-main",
		Tenant:    "tenant-a",
	}

	got, err := Render("{{ .Namespace }}/{{ .Workload }}/{{ .Type }}/{{ .Provider }}/{{ .Tenant }}", ctx)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	want := "wallet/wallet-api/redis/redis-main/tenant-a"
	if got != want {
		t.Fatalf("Render() = %q, want %q", got, want)
	}
}
