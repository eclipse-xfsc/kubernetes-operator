package modules

import (
	"encoding/json"
	"fmt"
	"strings"

	resourcesv1alpha1 "github.com/eclipse-xfsc/kubernetes-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

func SecretString(secret corev1.Secret, keys ...string) (string, error) {
	for _, key := range keys {
		if value := strings.TrimSpace(string(secret.Data[key])); value != "" {
			return value, nil
		}
	}
	return "", fmt.Errorf("secret %s/%s is missing one of keys %v", secret.Namespace, secret.Name, keys)
}

func OptionalSecretString(secret corev1.Secret, keys ...string) string {
	value, _ := SecretString(secret, keys...)
	return value
}

func DecodeParameters(claim resourcesv1alpha1.ResourceClaim, out any) error {
	if len(claim.Spec.Parameters.Raw) == 0 {
		return nil
	}
	if err := json.Unmarshal(claim.Spec.Parameters.Raw, out); err != nil {
		return fmt.Errorf("decode parameters for claim %s/%s: %w", claim.Namespace, claim.Name, err)
	}
	return nil
}
