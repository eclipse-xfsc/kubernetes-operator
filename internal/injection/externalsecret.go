package injection

import (
	"fmt"
	"strings"

	resourcesv1alpha1 "github.com/eclipse-xfsc/kubernetes-operator/api/v1alpha1"
	"github.com/eclipse-xfsc/kubernetes-operator/internal/render"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type ExternalSecretBuildResult struct {
	Object           *unstructured.Unstructured
	TargetSecretName string
}

func BuildExternalSecrets(provider *resourcesv1alpha1.ResourceProvider, namespace, workload string) ([]ExternalSecretBuildResult, error) {
	ctx := render.Context{Namespace: namespace, Workload: workload, Type: provider.Spec.Type, Provider: provider.Name, Tenant: namespace}
	results := make([]ExternalSecretBuildResult, 0, len(provider.Spec.Outputs.ExternalSecrets))
	for i, out := range provider.Spec.Outputs.ExternalSecrets {
		store := out.SecretStoreRef
		if store.Kind == "" {
			store.Kind = "ClusterSecretStore"
		}
		if store.Name == "" {
			return nil, fmt.Errorf("provider %s externalSecrets[%d] has no secretStoreRef.name", provider.Name, i)
		}
		name := render.Template(out.NameTemplate, ctx)
		if name == "" {
			name = fmt.Sprintf("%s-%s-eso", workload, provider.Spec.Type)
		}
		target := render.Template(out.TargetSecretNameTemplate, ctx)
		if target == "" {
			target = fmt.Sprintf("%s-%s", workload, provider.Spec.Type)
		}
		refresh := out.RefreshInterval
		if refresh == "" {
			refresh = "1h"
		}
		remoteKey := render.Template(out.RemoteKeyTemplate, ctx)
		if strings.TrimSpace(remoteKey) == "" {
			return nil, fmt.Errorf("provider %s externalSecrets[%d] has empty remoteKeyTemplate", provider.Name, i)
		}

		data := make([]any, 0, len(out.Data))
		for _, d := range out.Data {
			data = append(data, map[string]any{"secretKey": d.EnvName, "remoteRef": map[string]any{"key": remoteKey, "property": d.Property}})
		}
		es := &unstructured.Unstructured{Object: map[string]any{
			"apiVersion": "external-secrets.io/v1",
			"kind":       "ExternalSecret",
			"metadata":   map[string]any{"name": name, "namespace": namespace, "labels": map[string]any{"app.kubernetes.io/managed-by": "xsfc-resource-operator", "resources.xfsc.io/provider": provider.Name}},
			"spec":       map[string]any{"refreshInterval": refresh, "secretStoreRef": map[string]any{"kind": store.Kind, "name": store.Name}, "target": map[string]any{"name": target, "creationPolicy": "Owner"}, "data": data},
		}}
		es.SetOwnerReferences([]metav1.OwnerReference{})
		results = append(results, ExternalSecretBuildResult{Object: es, TargetSecretName: target})
	}
	return results, nil
}
