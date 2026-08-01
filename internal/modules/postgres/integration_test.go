package postgres

import (
	"context"
	"testing"

	resourcesv1alpha1 "github.com/eclipse-xfsc/kubernetes-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

func TestIntegrationEnsureRoleAndDatabase(t *testing.T) {
	provider := resourcesv1alpha1.ResourceProvider{
		Spec: resourcesv1alpha1.ResourceProviderSpec{
			Outputs: resourcesv1alpha1.ProviderOutputs{
				Env: map[string]string{
					"POSTGRES_HOST":    "127.0.0.1",
					"POSTGRES_PORT":    "15432",
					"POSTGRES_SSLMODE": "disable",
				},
			},
		},
	}
	secret := corev1.Secret{Data: map[string][]byte{
		"username": []byte("root"),
		"password": []byte("rootpass"),
	}}

	c, err := NewClient(provider, secret, nil)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	if err = c.EnsureRole(ctx, "xfsc_test", "testpass"); err != nil {
		t.Fatal(err)
	}
	if err = c.EnsureDatabase(ctx, "xfsc_test", "xfsc_test"); err != nil {
		t.Fatal(err)
	}
}
