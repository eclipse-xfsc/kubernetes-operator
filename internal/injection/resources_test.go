package injection

import (
	"testing"

	resourcesv1alpha1 "github.com/eclipse-xfsc/kubernetes-operator/api/v1alpha1"
)

func TestBuildConfigMaps(t *testing.T) {
	p := &resourcesv1alpha1.ResourceProvider{}
	p.Name = "redis-main"
	p.Spec.Type = "redis"
	p.Spec.Outputs.Config = []resourcesv1alpha1.ConfigMapOutput{{NameTemplate: "{{ .Workload }}-redis", Data: map[string]string{"endpoint": "{{ .Namespace }}.svc"}}}
	got, err := BuildConfigMaps(p, "wallet", "api")
	if err != nil {
		t.Fatal(err)
	}
	if got[0].GetName() != "api-redis" {
		t.Fatalf("name=%s", got[0].GetName())
	}
	data, _, _ := unstructuredNestedStringMap(got[0].Object, "data")
	if data["endpoint"] != "wallet.svc" {
		t.Fatalf("data=%v", data)
	}
}

func TestBuildJobsForcesTargetNamespace(t *testing.T) {
	p := &resourcesv1alpha1.ResourceProvider{}
	p.Name = "postgres-main"
	p.Spec.Type = "postgres"
	p.Spec.Outputs.Jobs = []resourcesv1alpha1.JobOutput{{YAML: "apiVersion: batch/v1\nkind: Job\nmetadata:\n  name: migrate-{{ .Workload }}\nspec:\n  template:\n    spec:\n      restartPolicy: Never\n      containers:\n      - name: migrate\n        image: busybox\n"}}
	got, err := BuildJobs(p, "wallet", "api")
	if err != nil {
		t.Fatal(err)
	}
	if got[0].GetNamespace() != "wallet" || got[0].GetName() != "migrate-api" {
		t.Fatalf("metadata=%s/%s", got[0].GetNamespace(), got[0].GetName())
	}
}

func TestBuildJobsRejectsNonJob(t *testing.T) {
	p := &resourcesv1alpha1.ResourceProvider{}
	p.Name = "x"
	p.Spec.Outputs.Jobs = []resourcesv1alpha1.JobOutput{{YAML: "apiVersion: v1\nkind: Pod\nmetadata:\n  name: no\n"}}
	if _, err := BuildJobs(p, "n", "w"); err == nil {
		t.Fatal("expected error")
	}
}

func unstructuredNestedStringMap(obj map[string]any, field string) (map[string]string, bool, error) {
	raw, ok := obj[field].(map[string]any)
	if !ok {
		return nil, false, nil
	}
	out := map[string]string{}
	for k, v := range raw {
		out[k] = v.(string)
	}
	return out, true, nil
}
