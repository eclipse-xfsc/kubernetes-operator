package s3

import (
	"context"
	"testing"

	resourcesv1alpha1 "github.com/eclipse-xfsc/kubernetes-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

func TestIntegrationEnsureBucketAndUser(t *testing.T) {
	provider := resourcesv1alpha1.ResourceProvider{
		Spec: resourcesv1alpha1.ResourceProviderSpec{
			Outputs: resourcesv1alpha1.ProviderOutputs{
				Env: map[string]string{
					"S3_ENDPOINT": "http://127.0.0.1:19000",
				},
			},
		},
	}
	secret := corev1.Secret{Data: map[string][]byte{
		"username": []byte("rootaccess"),
		"password": []byte("rootsecret123"),
	}}

	c, err := NewClient(provider, secret, nil)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	if err = c.EnsureBucket(ctx, "xfsc-test"); err != nil {
		t.Fatal(err)
	}
	if err = c.EnsureUser(ctx, "xfsc-test", "testsecret123"); err != nil {
		t.Fatal(err)
	}
}
