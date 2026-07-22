package postgres

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestIntegrationEnsureRoleAndDatabase(t *testing.T) {

	c, err := NewClient(corev1.Secret{Data: map[string][]byte{"host": []byte("127.0.0.1"), "port": []byte("15432"), "username": []byte("root"), "password": []byte("rootpass")}}, nil)
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
