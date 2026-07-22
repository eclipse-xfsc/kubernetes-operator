package injection

import (
	"encoding/json"
	"fmt"
	"strings"

	resourcesv1alpha1 "github.com/eclipse-xfsc/kubernetes-operator/api/v1alpha1"
	"github.com/eclipse-xfsc/kubernetes-operator/internal/render"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

func BuildConfigMaps(provider *resourcesv1alpha1.ResourceProvider, namespace, workload string) ([]*unstructured.Unstructured, error) {
	ctx := render.Context{Namespace: namespace, Workload: workload, Type: provider.Spec.Type, Provider: provider.Name, Tenant: namespace}
	out := make([]*unstructured.Unstructured, 0, len(provider.Spec.Outputs.Config))
	for i, cfg := range provider.Spec.Outputs.Config {
		name := render.Template(cfg.NameTemplate, ctx)
		if name == "" {
			name = fmt.Sprintf("%s-%s-config", workload, provider.Spec.Type)
		}
		data := map[string]any{}
		for k, v := range cfg.Data {
			data[k] = render.Template(v, ctx)
		}
		obj := &unstructured.Unstructured{Object: map[string]any{
			"apiVersion": "v1", "kind": "ConfigMap",
			"metadata": map[string]any{"name": name, "namespace": namespace, "labels": managedLabels(provider.Name)},
			"data":     data,
		}}
		if len(data) == 0 && len(cfg.Env) == 0 {
			return nil, fmt.Errorf("provider %s config[%d] is empty", provider.Name, i)
		}
		out = append(out, obj)
	}
	return out, nil
}

func BuildJobs(provider *resourcesv1alpha1.ResourceProvider, namespace, workload string) ([]*unstructured.Unstructured, error) {
	ctx := render.Context{Namespace: namespace, Workload: workload, Type: provider.Spec.Type, Provider: provider.Name, Tenant: namespace}
	out := make([]*unstructured.Unstructured, 0, len(provider.Spec.Outputs.Jobs))
	for i, job := range provider.Spec.Outputs.Jobs {
		rendered := render.Template(job.YAML, ctx)
		if strings.TrimSpace(rendered) == "" {
			return nil, fmt.Errorf("provider %s jobs[%d] has empty yaml", provider.Name, i)
		}
		rawJSON, err := yaml.YAMLToJSON([]byte(rendered))
		if err != nil {
			return nil, fmt.Errorf("provider %s jobs[%d]: %w", provider.Name, i, err)
		}
		var object map[string]any
		if err = json.Unmarshal(rawJSON, &object); err != nil {
			return nil, err
		}
		u := &unstructured.Unstructured{Object: object}
		if u.GetKind() != "Job" || u.GetAPIVersion() == "" {
			return nil, fmt.Errorf("provider %s jobs[%d] must contain a batch Job", provider.Name, i)
		}
		u.SetNamespace(namespace)
		if n := render.Template(job.NameTemplate, ctx); n != "" {
			u.SetName(n)
		}
		if u.GetName() == "" {
			return nil, fmt.Errorf("provider %s jobs[%d] has no metadata.name", provider.Name, i)
		}
		labels := u.GetLabels()
		if labels == nil {
			labels = map[string]string{}
		}
		for k, v := range managedStringLabels(provider.Name) {
			labels[k] = v
		}
		u.SetLabels(labels)
		out = append(out, u)
	}
	return out, nil
}

func managedLabels(provider string) map[string]any {
	return map[string]any{"app.kubernetes.io/managed-by": "xsfc-resource-operator", "resources.xfsc.io/provider": provider}
}
func managedStringLabels(provider string) map[string]string {
	return map[string]string{"app.kubernetes.io/managed-by": "xsfc-resource-operator", "resources.xfsc.io/provider": provider}
}
