# Resource modules

This package contains optional provider-specific reconciliation hooks. The
generic workload injection pipeline remains responsible for resolving
`ResourceProvider` objects, creating `ExternalSecret` resources and patching
workloads.

Registered module types:

- `redis`
- `postgres`
- `cassandra`
- `nats`
- `s3`
- `vault`

All modules currently contain only their public provisioning boundary and act
as no-ops when no concrete provisioner is supplied. Provider-specific behavior
can therefore be implemented incrementally without changing the injection
controller.
