package injection

import (
	"context"
	"strings"

	resourcesv1alpha1 "github.com/eclipse-xfsc/kubernetes-operator/api/v1alpha1"
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
	var list resourcesv1alpha1.ResourceProviderList
	if err := c.List(ctx, &list); err != nil {
		return nil, err
	}
	resolved := []resourcesv1alpha1.ResourceProvider{}
	for _, p := range list.Items {
		if !providerAllowed(p, namespace) {
			continue
		}
		if len(wantedProviders) > 0 {
			if contains(wantedProviders, p.Name) || contains(wantedProviders, p.Namespace+"/"+p.Name) {
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

func providerAllowed(p resourcesv1alpha1.ResourceProvider, namespace string) bool {
	if len(p.Spec.Allow.Namespaces) == 0 {
		return p.Namespace == namespace || p.Namespace == "xsfc-system"
	}
	return contains(p.Spec.Allow.Namespaces, namespace) || contains(p.Spec.Allow.Namespaces, "*")
}
func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}
