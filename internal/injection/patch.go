package injection

import (
	"encoding/json"
	"fmt"

	resourcesv1alpha1 "github.com/eclipse-xfsc/kubernetes-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	AnnotationEnabled = "inject.xfsc.io/enabled"
	AnnotationTypes   = "inject.xfsc.io/types"
	AnnotationHash    = "inject.xfsc.io/hash"
)

func PatchWorkload(obj *unstructured.Unstructured, bindings []resourcesv1alpha1.ResourceBinding, providers map[string]resourcesv1alpha1.ResourceProvider) error {
	containers, found, err := unstructured.NestedSlice(obj.Object, "spec", "template", "spec", "containers")
	if err != nil || !found {
		return fmt.Errorf("workload has no pod template containers")
	}

	for i := range containers {
		c, ok := containers[i].(map[string]any)
		if !ok {
			continue
		}
		name, _ := c["name"].(string)
		env, _ := c["env"].([]any)
		for _, b := range bindings {
			if !containerSelected(name, b.Spec.Injection.Containers) {
				continue
			}
			provider := providers[providerKey(b.Spec.ProviderRef.Namespace, b.Spec.ProviderRef.Name)]
			for k, v := range provider.Spec.Config.Env {
				env = upsertEnv(env, k, map[string]any{"name": k, "value": v})
			}
			for k, v := range b.Spec.Config.Env {
				env = upsertEnv(env, k, map[string]any{"name": k, "value": v})
			}
			targetSecretName := b.Spec.Secret.TargetSecretName
			if targetSecretName == "" {
				targetSecretName = b.Name + "-secret"
			}
			for _, d := range b.Spec.Secret.Data {
				env = upsertEnv(env, d.EnvName, map[string]any{"name": d.EnvName, "valueFrom": map[string]any{"secretKeyRef": map[string]any{"name": targetSecretName, "key": d.EnvName}}})
			}
		}
		c["env"] = env
		containers[i] = c
	}
	if err := unstructured.SetNestedSlice(obj.Object, containers, "spec", "template", "spec", "containers"); err != nil {
		return err
	}

	ann, _, _ := unstructured.NestedStringMap(obj.Object, "spec", "template", "metadata", "annotations")
	if ann == nil {
		ann = map[string]string{}
	}
	b, _ := json.Marshal(bindings)
	ann[AnnotationHash] = fmt.Sprintf("%x", b)
	return unstructured.SetNestedStringMap(obj.Object, ann, "spec", "template", "metadata", "annotations")
}

func upsertEnv(env []any, name string, item map[string]any) []any {
	for i, e := range env {
		if m, ok := e.(map[string]any); ok && m["name"] == name {
			env[i] = item
			return env
		}
	}
	return append(env, item)
}
func containerSelected(name string, selected []string) bool {
	if len(selected) == 0 {
		return true
	}
	for _, s := range selected {
		if s == name {
			return true
		}
	}
	return false
}
func providerKey(ns, name string) string { return ns + "/" + name }
