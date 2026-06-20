package main

import (
	"context"

	inventory "github.com/eclipse-xfsc/kubernetes-operator/cmd/apigen/gen/inventory"
	"goa.design/clue/log"
)

// inventory service example implementation.
// The example methods log the requests and return zero values.
type inventorysrvc struct{}

// NewInventory returns the inventory service implementation.
func NewInventory() inventory.Service {
	return &inventorysrvc{}
}

// Version implements version.
func (s *inventorysrvc) Version(ctx context.Context) (res *inventory.XsfcVersion, err error) {
	res = &inventory.XsfcVersion{}
	log.Printf(ctx, "inventory.version")
	return
}

// Modules implements modules.
func (s *inventorysrvc) Modules(ctx context.Context) (res []*inventory.XsfcModule, err error) {
	log.Printf(ctx, "inventory.modules")
	return
}

// Types implements types.
func (s *inventorysrvc) Types(ctx context.Context) (res []string, err error) {
	log.Printf(ctx, "inventory.types")
	return
}

// Providers implements providers.
func (s *inventorysrvc) Providers(ctx context.Context) (res []*inventory.XsfcProvider, err error) {
	log.Printf(ctx, "inventory.providers")
	return
}

// Consumers implements consumers.
func (s *inventorysrvc) Consumers(ctx context.Context) (res []*inventory.XsfcConsumer, err error) {
	log.Printf(ctx, "inventory.consumers")
	return
}

// Injections implements injections.
func (s *inventorysrvc) Injections(ctx context.Context) (res []*inventory.XsfcInjection, err error) {
	log.Printf(ctx, "inventory.injections")
	return
}

// Accounts implements accounts.
func (s *inventorysrvc) Accounts(ctx context.Context) (res []*inventory.XsfcAccount, err error) {
	log.Printf(ctx, "inventory.accounts")
	return
}

// AccountsByConsumer implements accountsByConsumer.
func (s *inventorysrvc) AccountsByConsumer(ctx context.Context, p *inventory.AccountsByConsumerPayload) (res []*inventory.XsfcAccount, err error) {
	log.Printf(ctx, "inventory.accountsByConsumer")
	return
}

// ManifestsRequestingInjection implements manifestsRequestingInjection.
func (s *inventorysrvc) ManifestsRequestingInjection(ctx context.Context) (res []*inventory.XsfcManifest, err error) {
	log.Printf(ctx, "inventory.manifestsRequestingInjection")
	return
}

// Healthz implements healthz.
func (s *inventorysrvc) Healthz(ctx context.Context) (res string, err error) {
	log.Printf(ctx, "inventory.healthz")
	return
}

// Readyz implements readyz.
func (s *inventorysrvc) Readyz(ctx context.Context) (res string, err error) {
	log.Printf(ctx, "inventory.readyz")
	return
}
