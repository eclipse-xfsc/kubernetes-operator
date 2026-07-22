package injection

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
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
	AnnotationEnvPrefix    = "inject.xfsc.io/env-prefix"
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
	Providers map[string]ManagedProviderState `json:"providers,omitempty"` // cluster-scoped provider name
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

var envPrefixPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// EnvPrefix reads the optional consumer-specific prefix. A value such as
// "WALLET" transforms VAULT_ADDR into WALLET_VAULT_ADDR. Empty means that
// environment variable names are injected unchanged.
func EnvPrefix(annotations map[string]string) (string, error) {
	prefix := strings.Trim(strings.TrimSpace(annotations[AnnotationEnvPrefix]), "_")
	if prefix == "" {
		return "", nil
	}
	if !envPrefixPattern.MatchString(prefix) {
		return "", fmt.Errorf("annotation %s contains invalid env prefix %q", AnnotationEnvPrefix, prefix)
	}
	return prefix, nil
}

func PrefixEnvName(prefix, name string) string {
	if prefix == "" {
		return name
	}
	return prefix + "_" + name
}

func PatchWorkload(obj *unstructured.Unstructured, providers []resourcesv1alpha1.ResourceProvider, resources map[string][]string, missing []string) (ManagedState, error) {
	containers, found, err := unstructured.NestedSlice(obj.Object, "spec", "template", "spec", "containers")
	if err != nil || !found {
		return ManagedState{}, fmt.Errorf("workload has no pod template containers")
	}
	old := ReadManagedState(obj)
	consumerAnnotations := map[string]string{}
	for key, value := range obj.GetAnnotations() {
		consumerAnnotations[key] = value
	}
	templateAnnotations, _, _ := unstructured.NestedStringMap(obj.Object, "spec", "template", "metadata", "annotations")
	for key, value := range templateAnnotations {
		consumerAnnotations[key] = value
	}
	prefix, err := EnvPrefix(consumerAnnotations)
	if err != nil {
		return ManagedState{}, err
	}
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
			key := p.Name
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
				injectedName := PrefixEnvName(prefix, k)
				env = upsertEnv(env, injectedName, map[string]any{"name": injectedName, "value": render.Template(v, ctx)})
				names[injectedName] = struct{}{}
			}
			for _, cfg := range p.Spec.Outputs.Config {
				configName := render.Template(cfg.NameTemplate, ctx)
				if configName == "" {
					configName = fmt.Sprintf("%s-%s-config", obj.GetName(), p.Spec.Type)
				}
				for envName, key := range cfg.Env {
					injectedName := PrefixEnvName(prefix, envName)
					env = upsertEnv(env, injectedName, map[string]any{"name": injectedName, "valueFrom": map[string]any{"configMapKeyRef": map[string]any{"name": configName, "key": key}}})
					names[injectedName] = struct{}{}
				}
			}
			for _, es := range p.Spec.Outputs.ExternalSecrets {
				target := render.Template(es.TargetSecretNameTemplate, ctx)
				if target == "" {
					target = fmt.Sprintf("%s-%s", obj.GetName(), p.Spec.Type)
				}
				for _, d := range es.Data {
					injectedName := PrefixEnvName(prefix, d.EnvName)
					env = upsertEnv(env, injectedName, map[string]any{"name": injectedName, "valueFrom": map[string]any{"secretKeyRef": map[string]any{"name": target, "key": d.EnvName}}})
					names[injectedName] = struct{}{}
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
