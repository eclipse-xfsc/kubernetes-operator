package injection

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	resourcesv1alpha1 "github.com/eclipse-xfsc/kubernetes-operator/api/v1alpha1"
	"github.com/eclipse-xfsc/kubernetes-operator/internal/render"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	AnnotationEnabled   = "inject.xfsc.io/enabled"
	AnnotationNeeds     = "inject.xfsc.io/needs"
	AnnotationProviders = "inject.xfsc.io/providers"
	AnnotationHash      = "inject.xfsc.io/hash"
)

func PatchWorkload(obj *unstructured.Unstructured, providers []resourcesv1alpha1.ResourceProvider) error {
	containers, found, err := unstructured.NestedSlice(obj.Object, "spec", "template", "spec", "containers")
	if err != nil || !found {
		return fmt.Errorf("workload has no pod template containers")
	}

	for i := range containers {
		c, ok := containers[i].(map[string]any)
		if !ok {
			continue
		}
		env, _ := c["env"].([]any)
		for _, p := range providers {
			ctx := render.Context{Namespace: obj.GetNamespace(), Workload: obj.GetName(), Type: p.Spec.Type, Provider: p.Name, Tenant: obj.GetNamespace()}
			for k, v := range p.Spec.Outputs.Env {
				env = upsertEnv(env, k, map[string]any{"name": k, "value": render.Template(v, ctx)})
			}
			for _, es := range p.Spec.Outputs.ExternalSecrets {
				target := render.Template(es.TargetSecretNameTemplate, ctx)
				if target == "" {
					target = fmt.Sprintf("%s-%s", obj.GetName(), p.Spec.Type)
				}
				for _, d := range es.Data {
					env = upsertEnv(env, d.EnvName, map[string]any{"name": d.EnvName, "valueFrom": map[string]any{"secretKeyRef": map[string]any{"name": target, "key": d.EnvName}}})
				}
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
	b, _ := json.Marshal(providers)
	sum := sha256.Sum256(b)
	ann[AnnotationHash] = hex.EncodeToString(sum[:])
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
