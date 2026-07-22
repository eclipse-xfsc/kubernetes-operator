package redis

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestIntegrationEnsureUser(t *testing.T) {
	c, err := NewClient(corev1.Secret{Data: map[string][]byte{"host": []byte("127.0.0.1"), "port": []byte("16379"), "password": []byte("rootpass")}})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if err := c.EnsureUser(context.Background(), "xfsc_test", []string{"reset", "on", ">testpass", "~*", "+@all"}); err != nil {
		t.Fatal(err)
	}
}
