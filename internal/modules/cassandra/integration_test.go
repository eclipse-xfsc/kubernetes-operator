package cassandra

import (
	"context"
	"testing"

	resourcesv1alpha1 "github.com/eclipse-xfsc/kubernetes-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

func TestIntegrationEnsureRoleAndKeyspace(t *testing.T) {
	provider := resourcesv1alpha1.ResourceProvider{
		Spec: resourcesv1alpha1.ResourceProviderSpec{
			Outputs: resourcesv1alpha1.ProviderOutputs{
				Env: map[string]string{
					"CASSANDRA_HOST": "127.0.0.1",
					"CASSANDRA_PORT": "9042",
				},
			},
		},
	}
	secret := corev1.Secret{Data: map[string][]byte{
		"username": []byte("cassandra"),
		"password": []byte("cassandra"),
	}}

	c, err := NewClient(provider, secret)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	ctx := context.Background()
	if err = c.EnsureRole(ctx, "xfsc_test", "testpass"); err != nil {
		t.Fatal(err)
	}
	if err = c.EnsureKeyspace(ctx, "xfsc_test", "", 1); err != nil {
		t.Fatal(err)
	}
	if err = c.GrantAllOnKeyspace(ctx, "xfsc_test", "xfsc_test"); err != nil {
		t.Fatal(err)
	}
}
