package registry

import (
	"fmt"
	"sort"

	"github.com/eclipse-xfsc/kubernetes-operator/internal/modules"
)

type Registry struct {
	modulesByName map[string]modules.Module
	modulesByType map[string]modules.Module
}

func New() *Registry { return &Registry{map[string]modules.Module{}, map[string]modules.Module{}} }
func (r *Registry) MustRegister(items ...modules.Module) {
	for _, i := range items {
		if err := r.Register(i); err != nil {
			panic(err)
		}
	}
}
func (r *Registry) Register(item modules.Module) error {
	if _, ok := r.modulesByName[item.Name()]; ok {
		return fmt.Errorf("module %q already registered", item.Name())
	}
	r.modulesByName[item.Name()] = item
	for _, t := range item.Types() {
		if _, ok := r.modulesByType[t]; ok {
			return fmt.Errorf("type %q already registered", t)
		}
		r.modulesByType[t] = item
	}
	return nil
}
func (r *Registry) Modules() []modules.Module {
	out := []modules.Module{}
	for _, m := range r.modulesByName {
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name() < out[j].Name() })
	return out
}
func (r *Registry) Types() []string {
	out := []string{}
	for t := range r.modulesByType {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}
func (r *Registry) ForType(t string) (modules.Module, bool) {
	m, ok := r.modulesByType[t]
	return m, ok
}
