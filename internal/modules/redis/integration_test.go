package redis

import (
	"context"
	"testing"

	resourcesv1alpha1 "github.com/eclipse-xfsc/kubernetes-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

func TestIntegrationEnsureUser(t *testing.T) {
	provider := resourcesv1alpha1.ResourceProvider{
		Spec: resourcesv1alpha1.ResourceProviderSpec{
			Outputs: resourcesv1alpha1.ProviderOutputs{
				Env: map[string]string{
					"REDIS_HOSTS": "127.0.0.1:16379",
					"REDIS_PORT":  "16379",
					"REDIS_TLS":   "false",
				},
			},
		},
	}
	secret := corev1.Secret{Data: map[string][]byte{
		"password": []byte("rootpass"),
	}}

	c, err := NewClient(provider, secret)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	ctx := context.Background()
	if err = c.EnsureUser(ctx, "xfsc_test", []string{"reset", "on", ">testpass", "~*", "+@all"}); err != nil {
		t.Fatal(err)
	}
}
