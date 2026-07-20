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
	modules map[string]Module
}

func NewRegistry(ms ...Module) *Registry {
	r := &Registry{
		log:     ctrl.Log.WithName("modules"),
		modules: make(map[string]Module),
	}

	for _, module := range ms {
		if module == nil {
			continue
		}

		resourceType := strings.TrimSpace(strings.ToLower(module.Type()))
		if resourceType == "" {
			r.log.Info(
				"Skipping module with empty type",
				"implementation", fmt.Sprintf("%T", module),
			)
			continue
		}

		if existing, exists := r.modules[resourceType]; exists {
			r.log.Info(
				"Replacing already registered module",
				"type", resourceType,
				"previousImplementation", fmt.Sprintf("%T", existing),
				"implementation", fmt.Sprintf("%T", module),
			)
		}

		r.modules[resourceType] = module

		r.log.Info(
			"Registered module",
			"type", resourceType,
			"implementation", fmt.Sprintf("%T", module),
		)
	}

	r.log.Info(
		"Module registry initialized",
		"count", len(r.modules),
		"types", r.Types(),
	)

	return r
}

func (r *Registry) Get(resourceType string) (Module, bool) {
	if r == nil {
		return nil, false
	}

	resourceType = strings.TrimSpace(strings.ToLower(resourceType))
	module, found := r.modules[resourceType]

	return module, found
}

func (r *Registry) Reconcile(
	ctx context.Context,
	resourceType string,
	req *Request,
) (Result, error) {
	return r.Dispatch(ctx, resourceType, req)
}

func (r *Registry) Dispatch(
	ctx context.Context,
	resourceType string,
	req *Request,
) (Result, error) {
	resourceType = strings.TrimSpace(strings.ToLower(resourceType))

	module, found := r.Get(resourceType)
	if !found {
		r.log.V(1).Info(
			"No module registered for provider type",
			"type", resourceType,
		)

		return Result{}, nil
	}

	if req == nil {
		return Result{}, fmt.Errorf(
			"cannot execute module %q with nil request",
			resourceType,
		)
	}

	r.log.Info(
		"Executing provider module",
		"type", resourceType,
		"implementation", fmt.Sprintf("%T", module),
	)

	result, err := module.Reconcile(ctx, *req)
	if err != nil {
		r.log.Error(
			err,
			"Provider module failed",
			"type", resourceType,
			"implementation", fmt.Sprintf("%T", module),
		)

		return Result{}, err
	}

	r.log.Info(
		"Provider module completed",
		"type", resourceType,
		"implementation", fmt.Sprintf("%T", module),
		"resourceCount", len(result.Resources),
	)

	return result, nil
}

func (r *Registry) Types() []string {
	if r == nil {
		return nil
	}

	types := make([]string, 0, len(r.modules))
	for resourceType := range r.modules {
		types = append(types, resourceType)
	}

	sort.Strings(types)

	return types
}
