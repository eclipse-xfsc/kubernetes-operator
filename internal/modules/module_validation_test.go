package modules_test

import (
	"context"
	"strings"
	"testing"

	"github.com/eclipse-xfsc/kubernetes-operator/internal/modules"
	cassandra "github.com/eclipse-xfsc/kubernetes-operator/internal/modules/cassandra"
	nats "github.com/eclipse-xfsc/kubernetes-operator/internal/modules/nats"
	postgres "github.com/eclipse-xfsc/kubernetes-operator/internal/modules/postgres"
	redis "github.com/eclipse-xfsc/kubernetes-operator/internal/modules/redis"
	s3 "github.com/eclipse-xfsc/kubernetes-operator/internal/modules/s3"
	corev1 "k8s.io/api/core/v1"
)

type backend struct{ called bool }

func (b *backend) Provision(context.Context, modules.ProvisionRequest) error {
	b.called = true
	return nil
}
func TestModulesValidateAndDelegate(t *testing.T) {
	constructors := []struct {
		name string
		new  func(modules.Provisioner) modules.Provisioner
	}{
		{"postgres", func(_ modules.Provisioner) modules.Provisioner { return postgres.New(&backend{}) }},
		{"redis", func(_ modules.Provisioner) modules.Provisioner { return redis.New(&backend{}) }},
		{"cassandra", func(_ modules.Provisioner) modules.Provisioner { return cassandra.New(&backend{}) }},
		{"nats", func(_ modules.Provisioner) modules.Provisioner { return nats.New(&backend{}) }},
		{"s3", func(_ modules.Provisioner) modules.Provisioner { return s3.New(&backend{}) }},
	}
	for _, tc := range constructors {
		t.Run(tc.name, func(t *testing.T) {
			m := tc.new(nil)
			if err := m.Provision(context.Background(), modules.ProvisionRequest{}); err == nil || !strings.Contains(err.Error(), "admin secret") {
				t.Fatalf("unexpected %v", err)
			}
			req := modules.ProvisionRequest{AdminSecret: corev1.Secret{Data: map[string][]byte{"x": []byte("y")}}, ClaimSecret: corev1.Secret{Data: map[string][]byte{"x": []byte("y")}}}
			if err := m.Provision(context.Background(), req); err != nil {
				t.Fatal(err)
			}
		})
	}
}
