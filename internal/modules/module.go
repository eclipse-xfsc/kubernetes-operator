package modules

import (
	"context"

	resourcesv1alpha1 "github.com/eclipse-xfsc/kubernetes-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Request contains all information a resource-specific module needs to
// reconcile provider-side actions for one consumer. Implementations must be
// idempotent because Reconcile can be called repeatedly.
type Request struct {
	Client      client.Client
	Provider    resourcesv1alpha1.ResourceProvider
	Namespace   string
	Workload    string
	Annotations map[string]string
}

// Result allows a module to return Kubernetes resources that should be owned
// and reconciled together with the provider injection. A module may also carry
// out external API calls directly and return an empty result.
type Result struct {
	Resources []*unstructured.Unstructured
}

// Module implements resource-specific behavior, for example creating a Redis
// account, a NATS user or a PostgreSQL role. Type must match
// ResourceProvider.spec.type.
type Module interface {
	Type() string
	Reconcile(context.Context, Request) (Result, error)
}
