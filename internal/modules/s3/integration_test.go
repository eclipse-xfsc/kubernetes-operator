package s3

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestIntegrationEnsureBucketAndUser(t *testing.T) {

	c, err := NewClient(corev1.Secret{Data: map[string][]byte{"endpoint": []byte("http://127.0.0.1:19000"), "username": []byte("rootaccess"), "password": []byte("rootsecret123")}}, nil)
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
