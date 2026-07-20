package modules

import (
	"context"
	"fmt"
	"sync"

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
// account, a NATS user or a Postgres role. Type must match ResourceProvider.spec.type.
type Module interface {
	Type() string
	Reconcile(context.Context, Request) (Result, error)
}

type Registry struct {
	mu      sync.RWMutex
	modules map[string]Module
}

func NewRegistry(mods ...Module) *Registry {
	r := &Registry{modules: map[string]Module{}}
	for _, m := range mods {
		r.Register(m)
	}
	return r
}

func (r *Registry) Register(module Module) {
	if r == nil || module == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.modules[module.Type()] = module
}

// Reconcile invokes the module registered for the provider type. Missing
// modules are intentionally treated as a no-op so pure injection providers do
// not need a module.
func (r *Registry) Reconcile(ctx context.Context, req Request) (Result, bool, error) {
	if r == nil {
		return Result{}, false, nil
	}
	r.mu.RLock()
	module, found := r.modules[req.Provider.Spec.Type]
	r.mu.RUnlock()
	if !found {
		return Result{}, false, nil
	}
	result, err := module.Reconcile(ctx, req)
	if err != nil {
		return Result{}, true, fmt.Errorf("module %q failed for provider %q: %w", module.Type(), req.Provider.Name, err)
	}
	return result, true, nil
}
