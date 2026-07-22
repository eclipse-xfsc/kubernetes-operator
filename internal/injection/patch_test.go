package injection

import (
	"testing"

	resourcesv1alpha1 "github.com/eclipse-xfsc/kubernetes-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestPatchWorkloadAppliesEnvPrefix(t *testing.T) {
	obj := testDeployment(map[string]string{AnnotationEnabled: "true", AnnotationEnvPrefix: "PREFIX"})
	provider := resourcesv1alpha1.ResourceProvider{}
	provider.Name = "vault"
	provider.Spec.Type = "vault"
	provider.Spec.Outputs.Env = map[string]string{"VAULT_ADDR": "http://vault:8200"}
	provider.Spec.Outputs.ExternalSecrets = []resourcesv1alpha1.ExternalSecretOutput{{
		TargetSecretNameTemplate: "app-vault",
		RemoteKeyTemplate:        "tenants/default/vault/app",
		Data:                     []resourcesv1alpha1.ExternalSecretKey{{EnvName: "VAULT_TOKEN", Property: "token"}},
	}}

	if _, err := PatchWorkload(obj, []resourcesv1alpha1.ResourceProvider{provider}, nil, nil); err != nil {
		t.Fatalf("PatchWorkload() error = %v", err)
	}

	env := containerEnv(t, obj)
	assertEnvName(t, env, "PREFIX_VAULT_ADDR")
	assertEnvName(t, env, "PREFIX_VAULT_TOKEN")
	if _, found := env["VAULT_ADDR"]; found {
		t.Fatalf("unprefixed VAULT_ADDR was injected")
	}
}

func TestPatchWorkloadWithoutPrefixKeepsNames(t *testing.T) {
	obj := testDeployment(map[string]string{AnnotationEnabled: "true"})
	provider := resourcesv1alpha1.ResourceProvider{}
	provider.Name = "vault"
	provider.Spec.Type = "vault"
	provider.Spec.Outputs.Env = map[string]string{"VAULT_ADDR": "http://vault:8200"}

	if _, err := PatchWorkload(obj, []resourcesv1alpha1.ResourceProvider{provider}, nil, nil); err != nil {
		t.Fatalf("PatchWorkload() error = %v", err)
	}
	assertEnvName(t, containerEnv(t, obj), "VAULT_ADDR")
}

func TestPatchWorkloadPrefixChangeRemovesOldManagedEnv(t *testing.T) {
	obj := testDeployment(map[string]string{AnnotationEnabled: "true", AnnotationEnvPrefix: "OLD"})
	provider := resourcesv1alpha1.ResourceProvider{}
	provider.Name = "vault"
	provider.Spec.Type = "vault"
	provider.Spec.Outputs.Env = map[string]string{"VAULT_ADDR": "http://vault:8200"}

	if _, err := PatchWorkload(obj, []resourcesv1alpha1.ResourceProvider{provider}, nil, nil); err != nil {
		t.Fatalf("first PatchWorkload() error = %v", err)
	}
	annotations, _, _ := unstructured.NestedStringMap(obj.Object, "spec", "template", "metadata", "annotations")
	annotations[AnnotationEnvPrefix] = "NEW"
	if err := unstructured.SetNestedStringMap(obj.Object, annotations, "spec", "template", "metadata", "annotations"); err != nil {
		t.Fatal(err)
	}
	if _, err := PatchWorkload(obj, []resourcesv1alpha1.ResourceProvider{provider}, nil, nil); err != nil {
		t.Fatalf("second PatchWorkload() error = %v", err)
	}

	env := containerEnv(t, obj)
	if _, found := env["OLD_VAULT_ADDR"]; found {
		t.Fatalf("old prefixed env was not removed")
	}
	assertEnvName(t, env, "NEW_VAULT_ADDR")
}

func testDeployment(annotations map[string]string) *unstructured.Unstructured {
	ann := map[string]any{}
	for k, v := range annotations {
		ann[k] = v
	}
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata": map[string]any{
			"name":      "app",
			"namespace": "default",
		},
		"spec": map[string]any{
			"template": map[string]any{
				"metadata": map[string]any{"annotations": ann},
				"spec":     map[string]any{"containers": []any{map[string]any{"name": "app"}}},
			},
		},
	}}
}

func containerEnv(t *testing.T, obj *unstructured.Unstructured) map[string]map[string]any {
	t.Helper()
	containers, found, err := unstructured.NestedSlice(obj.Object, "spec", "template", "spec", "containers")
	if err != nil || !found || len(containers) != 1 {
		t.Fatalf("containers not found: found=%v err=%v", found, err)
	}
	container := containers[0].(map[string]any)
	values := map[string]map[string]any{}
	for _, raw := range container["env"].([]any) {
		item := raw.(map[string]any)
		values[item["name"].(string)] = item
	}
	return values
}

func assertEnvName(t *testing.T, env map[string]map[string]any, name string) {
	t.Helper()
	if _, found := env[name]; !found {
		t.Fatalf("expected env %q, got %#v", name, env)
	}
}
