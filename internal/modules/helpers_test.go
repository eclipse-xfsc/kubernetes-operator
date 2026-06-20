package modules

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestRequestedTypes(t *testing.T) {
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				InjectEnabledAnno: "true",
				InjectTypesAnno:   "telemetry,database.postgres, cache.redis",
			},
		},
	}

	got := RequestedTypes(dep)

	want := []string{"telemetry", "database.postgres", "cache.redis"}

	if len(got) != len(want) {
		t.Fatalf("expected %d types, got %d: %#v", len(want), len(got), got)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected %q at index %d, got %q", want[i], i, got[i])
		}
	}
}

func TestRequestedTypesDisabled(t *testing.T) {
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				InjectEnabledAnno: "false",
				InjectTypesAnno:   "telemetry",
			},
		},
	}

	got := RequestedTypes(dep)

	if got != nil {
		t.Fatalf("expected nil, got %#v", got)
	}
}

func TestIsProvider(t *testing.T) {
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				ProviderLabel:     "true",
				ResourceTypeLabel: "telemetry",
				ResourceNameLabel: "default",
			},
		},
	}

	if !IsProvider(dep, "telemetry") {
		t.Fatal("expected object to be telemetry provider")
	}

	if IsProvider(dep, "database.postgres") {
		t.Fatal("did not expect object to be postgres provider")
	}
}
