package index

import (
	"sort"
	"sync"

	"github.com/eclipse-xfsc/kubernetes-operator/internal/modules"
)

type Manifest struct {
	APIVersion, Kind, Name, Namespace string
	RequestedTypes                    []string
	Annotations, Labels               map[string]string
}
type Inventory struct {
	mu         sync.RWMutex
	providers  map[string]modules.Provider
	consumers  map[string]modules.Consumer
	injections map[string]modules.InjectionResult
	accounts   map[string]modules.Account
	manifests  map[string]Manifest
}

func NewInventory() *Inventory {
	return &Inventory{providers: map[string]modules.Provider{}, consumers: map[string]modules.Consumer{}, injections: map[string]modules.InjectionResult{}, accounts: map[string]modules.Account{}, manifests: map[string]Manifest{}}
}
func key(p ...string) string {
	s := ""
	for i, x := range p {
		if i > 0 {
			s += "/"
		}
		s += x
	}
	return s
}
func (i *Inventory) UpsertProvider(p modules.Provider) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.providers[key(p.Namespace, p.Type, p.Name, p.Kind, p.Resource)] = p
}
func (i *Inventory) UpsertConsumer(c modules.Consumer) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.consumers[key(c.Namespace, c.Kind, c.Name, c.Type)] = c
}
func (i *Inventory) UpsertInjection(x modules.InjectionResult) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.injections[key(x.ConsumerNamespace, x.ConsumerKind, x.ConsumerName, x.Container)] = x
}
func (i *Inventory) UpsertAccount(a modules.Account) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.accounts[key(a.Namespace, a.Name, a.ConsumerNamespace, a.ConsumerName)] = a
}
func (i *Inventory) UpsertManifest(m Manifest) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.manifests[key(m.Namespace, m.Kind, m.Name)] = m
}
func (i *Inventory) Providers() []modules.Provider {
	i.mu.RLock()
	defer i.mu.RUnlock()
	out := []modules.Provider{}
	for _, p := range i.providers {
		out = append(out, p)
	}
	sort.Slice(out, func(a, b int) bool {
		if out[a].Type == out[b].Type {
			return out[a].Name < out[b].Name
		}
		return out[a].Type < out[b].Type
	})
	return out
}
func (i *Inventory) Consumers() []modules.Consumer {
	i.mu.RLock()
	defer i.mu.RUnlock()
	out := []modules.Consumer{}
	for _, c := range i.consumers {
		out = append(out, c)
	}
	sort.Slice(out, func(a, b int) bool {
		if out[a].Namespace == out[b].Namespace {
			return out[a].Name < out[b].Name
		}
		return out[a].Namespace < out[b].Namespace
	})
	return out
}
func (i *Inventory) Injections() []modules.InjectionResult {
	i.mu.RLock()
	defer i.mu.RUnlock()
	out := []modules.InjectionResult{}
	for _, x := range i.injections {
		out = append(out, x)
	}
	return out
}
func (i *Inventory) Accounts() []modules.Account {
	i.mu.RLock()
	defer i.mu.RUnlock()
	out := []modules.Account{}
	for _, a := range i.accounts {
		out = append(out, a)
	}
	sort.Slice(out, func(x, y int) bool {
		if out[x].ConsumerNamespace == out[y].ConsumerNamespace {
			return out[x].ConsumerName < out[y].ConsumerName
		}
		return out[x].ConsumerNamespace < out[y].ConsumerNamespace
	})
	return out
}
func (i *Inventory) AccountsByConsumer(ns, name string) []modules.Account {
	i.mu.RLock()
	defer i.mu.RUnlock()
	out := []modules.Account{}
	for _, a := range i.accounts {
		if a.ConsumerNamespace == ns && a.ConsumerName == name {
			out = append(out, a)
		}
	}
	return out
}
func (i *Inventory) ManifestsRequestingInjection() []Manifest {
	i.mu.RLock()
	defer i.mu.RUnlock()
	out := []Manifest{}
	for _, m := range i.manifests {
		out = append(out, m)
	}
	return out
}
