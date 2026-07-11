package injection

import (
	"context"
	"strings"

	resourcesv1alpha1 "github.com/eclipse-xfsc/kubernetes-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func WantsInjection(annotations map[string]string) bool {
	return annotations[AnnotationEnabled] == "true"
}

func SplitCSV(s string) []string {
	out := []string{}
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func ResolveProviders(ctx context.Context, c client.Client, namespace string, annotations map[string]string) ([]resourcesv1alpha1.ResourceProvider, error) {
	wantedTypes := SplitCSV(annotations[AnnotationNeeds])
	wantedProviders := SplitCSV(annotations[AnnotationProviders])

	var ns corev1.Namespace
	if err := c.Get(ctx, client.ObjectKey{Name: namespace}, &ns); err != nil {
		return nil, err
	}

	var list resourcesv1alpha1.ResourceProviderList
	if err := c.List(ctx, &list); err != nil {
		return nil, err
	}

	resolved := make([]resourcesv1alpha1.ResourceProvider, 0)
	for _, p := range list.Items {
		if !ProviderAllowed(p, namespace, ns.Labels) {
			continue
		}
		if len(wantedProviders) > 0 {
			if contains(wantedProviders, p.Name) {
				resolved = append(resolved, p)
			}
			continue
		}
		if len(wantedTypes) > 0 && contains(wantedTypes, p.Spec.Type) {
			resolved = append(resolved, p)
		}
	}
	return resolved, nil
}

func ProviderAllowed(p resourcesv1alpha1.ResourceProvider, namespace string, namespaceLabels map[string]string) bool {
	allow := p.Spec.Allow
	if len(allow.Namespaces) == 0 && len(allow.Selector) == 0 {
		return true
	}
	if contains(allow.Namespaces, "*") || contains(allow.Namespaces, namespace) {
		return true
	}
	if len(allow.Selector) > 0 {
		for key, value := range allow.Selector {
			if namespaceLabels[key] != value {
				return false
			}
		}
		return true
	}
	return false
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}
