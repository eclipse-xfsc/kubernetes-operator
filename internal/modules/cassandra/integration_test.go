package cassandra

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestIntegrationEnsureRoleAndKeyspace(t *testing.T) {
	c, err := NewClient(corev1.Secret{Data: map[string][]byte{"host": []byte("127.0.0.1"), "port": []byte("9042"), "username": []byte("cassandra"), "password": []byte("cassandra")}})
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
