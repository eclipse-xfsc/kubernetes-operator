package modules

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
)

type Registry struct {
	log     logr.Logger
	modules map[string]Provisioner
}

func NewRegistry(ms ...Provisioner) *Registry {
	r := &Registry{log: ctrl.Log.WithName("modules"), modules: make(map[string]Provisioner)}
	for _, module := range ms {
		if module == nil {
			continue
		}
		resourceType := strings.TrimSpace(strings.ToLower(module.Type()))
		if resourceType == "" {
			continue
		}
		r.modules[resourceType] = module
		r.log.Info("Registered provisioner", "type", resourceType, "implementation", fmt.Sprintf("%T", module))
	}
	r.log.Info("Provisioner registry initialized", "count", len(r.modules), "types", r.Types())
	return r
}

func (r *Registry) Get(resourceType string) (Provisioner, bool) {
	if r == nil {
		return nil, false
	}
	module, found := r.modules[strings.TrimSpace(strings.ToLower(resourceType))]
	return module, found
}

func (r *Registry) Provision(ctx context.Context, resourceType string, req ProvisionRequest) error {
	module, found := r.Get(resourceType)
	if !found {
		return fmt.Errorf("no provisioner registered for resource type %q", resourceType)
	}
	return module.Provision(ctx, req)
}

func (r *Registry) Types() []string {
	if r == nil {
		return nil
	}
	out := make([]string, 0, len(r.modules))
	for resourceType := range r.modules {
		out = append(out, resourceType)
	}
	sort.Strings(out)
	return out
}
