package nats

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	"reflect"
	"testing"
)

type recordingRunner struct {
	name string
	args []string
}

func (r *recordingRunner) Run(_ context.Context, name string, args, env []string) error {
	r.name = name
	r.args = append([]string(nil), args...)
	return nil
}
func TestEnsureAccountAndUserCommands(t *testing.T) {
	r := &recordingRunner{}
	c, err := NewClient(corev1.Secret{Data: map[string][]byte{"operator": []byte("XFSC")}}, r)
	if err != nil {
		t.Fatal(err)
	}
	if err = c.EnsureAccount(context.Background(), "wallet"); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(r.args, []string{"add", "account", "wallet", "--operator", "XFSC"}) {
		t.Fatalf("args=%v", r.args)
	}
	if err = c.EnsureUser(context.Background(), "wallet", "api", "secret", []string{"events.>"}, []string{"commands.>"}); err != nil {
		t.Fatal(err)
	}
	if r.name != "nsc" {
		t.Fatalf("name=%s", r.name)
	}
}
