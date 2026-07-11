package injection

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	resourcesv1alpha1 "github.com/eclipse-xfsc/kubernetes-operator/api/v1alpha1"
	"github.com/eclipse-xfsc/kubernetes-operator/internal/render"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	AnnotationEnabled      = "inject.xfsc.io/enabled"
	AnnotationNeeds        = "inject.xfsc.io/needs"
	AnnotationProviders    = "inject.xfsc.io/providers"
	AnnotationHash         = "inject.xfsc.io/hash"
	AnnotationManagedState = "inject.xfsc.io/managed-state"
	AnnotationWarning      = "inject.xfsc.io/warning"
)

type ManagedProviderState struct {
	Type      string   `json:"type"`
	Env       []string `json:"env,omitempty"`
	Resources []string `json:"resources,omitempty"` // apiVersion|kind|namespace|name
}

type ManagedState struct {
	Providers map[string]ManagedProviderState `json:"providers,omitempty"` // namespace/name
}

func ReadManagedState(obj *unstructured.Unstructured) ManagedState {
	state := ManagedState{Providers: map[string]ManagedProviderState{}}
	ann, _, _ := unstructured.NestedStringMap(obj.Object, "spec", "template", "metadata", "annotations")
	if raw := ann[AnnotationManagedState]; raw != "" {
		_ = json.Unmarshal([]byte(raw), &state)
	}
	if state.Providers == nil {
		state.Providers = map[string]ManagedProviderState{}
	}
	return state
}

func PatchWorkload(obj *unstructured.Unstructured, providers []resourcesv1alpha1.ResourceProvider, resources map[string][]string, missing []string) (ManagedState, error) {
	containers, found, err := unstructured.NestedSlice(obj.Object, "spec", "template", "spec", "containers")
	if err != nil || !found {
		return ManagedState{}, fmt.Errorf("workload has no pod template containers")
	}
	old := ReadManagedState(obj)
	managedNames := map[string]struct{}{}
	for _, ps := range old.Providers {
		for _, n := range ps.Env {
			managedNames[n] = struct{}{}
		}
	}

	state := ManagedState{Providers: map[string]ManagedProviderState{}}
	for i := range containers {
		c, ok := containers[i].(map[string]any)
		if !ok {
			continue
		}
		env, _ := c["env"].([]any)
		env = removeManagedEnv(env, managedNames)
		for _, p := range providers {
			key := p.Namespace + "/" + p.Name
			ps := state.Providers[key]
			ps.Type = p.Spec.Type
			if resources != nil {
				ps.Resources = resources[key]
			} else if oldPS, ok := old.Providers[key]; ok {
				ps.Resources = oldPS.Resources
			}
			ctx := render.Context{Namespace: obj.GetNamespace(), Workload: obj.GetName(), Type: p.Spec.Type, Provider: p.Name, Tenant: obj.GetNamespace()}
			names := map[string]struct{}{}
			for k, v := range p.Spec.Outputs.Env {
				env = upsertEnv(env, k, map[string]any{"name": k, "value": render.Template(v, ctx)})
				names[k] = struct{}{}
			}
			for _, es := range p.Spec.Outputs.ExternalSecrets {
				target := render.Template(es.TargetSecretNameTemplate, ctx)
				if target == "" {
					target = fmt.Sprintf("%s-%s", obj.GetName(), p.Spec.Type)
				}
				for _, d := range es.Data {
					env = upsertEnv(env, d.EnvName, map[string]any{"name": d.EnvName, "valueFrom": map[string]any{"secretKeyRef": map[string]any{"name": target, "key": d.EnvName}}})
					names[d.EnvName] = struct{}{}
				}
			}
			ps.Env = sortedKeys(names)
			state.Providers[key] = ps
		}
		c["env"] = env
		containers[i] = c
	}
	if err := unstructured.SetNestedSlice(obj.Object, containers, "spec", "template", "spec", "containers"); err != nil {
		return state, err
	}
	ann, _, _ := unstructured.NestedStringMap(obj.Object, "spec", "template", "metadata", "annotations")
	if ann == nil {
		ann = map[string]string{}
	}
	raw, _ := json.Marshal(state)
	ann[AnnotationManagedState] = string(raw)
	b, _ := json.Marshal(providers)
	sum := sha256.Sum256(b)
	ann[AnnotationHash] = hex.EncodeToString(sum[:])
	if len(missing) > 0 {
		ann[AnnotationWarning] = "Unavailable ResourceProviders: " + joinSorted(missing)
	} else {
		delete(ann, AnnotationWarning)
	}
	if err := unstructured.SetNestedStringMap(obj.Object, ann, "spec", "template", "metadata", "annotations"); err != nil {
		return state, err
	}
	meta := obj.GetAnnotations()
	if meta == nil {
		meta = map[string]string{}
	}
	if len(missing) > 0 {
		meta[AnnotationWarning] = "Unavailable ResourceProviders: " + joinSorted(missing)
	} else {
		delete(meta, AnnotationWarning)
	}
	obj.SetAnnotations(meta)
	return state, nil
}

func removeManagedEnv(env []any, managed map[string]struct{}) []any {
	out := make([]any, 0, len(env))
	for _, e := range env {
		if m, ok := e.(map[string]any); ok {
			if n, _ := m["name"].(string); n != "" {
				if _, owned := managed[n]; owned {
					continue
				}
			}
		}
		out = append(out, e)
	}
	return out
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
func sortedKeys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
func joinSorted(xs []string) string {
	sort.Strings(xs)
	return strings.Join(xs, ", ")
}
