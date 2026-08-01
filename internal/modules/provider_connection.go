package modules

import (
	"fmt"
	"net"
	"net/url"
	"strings"

	resourcesv1alpha1 "github.com/eclipse-xfsc/kubernetes-operator/api/v1alpha1"
)

func ProviderEnv(provider resourcesv1alpha1.ResourceProvider, keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(provider.Spec.Outputs.Env[key]); value != "" {
			return value
		}
	}
	return ""
}

func ProviderRequiredEnv(provider resourcesv1alpha1.ResourceProvider, keys ...string) (string, error) {
	if value := ProviderEnv(provider, keys...); value != "" {
		return value, nil
	}
	return "", fmt.Errorf("resource provider %q is missing one of output env keys %v", provider.Name, keys)
}

func SplitHostPort(value, defaultPort string) (string, string) {
	value = strings.TrimSpace(value)
	if parsed, err := url.Parse(value); err == nil && parsed.Host != "" {
		value = parsed.Host
	}
	if host, port, err := net.SplitHostPort(value); err == nil {
		return host, port
	}
	return strings.TrimSuffix(value, ":"+defaultPort), defaultPort
}
