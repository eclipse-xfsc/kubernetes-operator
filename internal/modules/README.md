# Resource modules

The workload injection pipeline remains provider based and is unchanged. The
`ResourceClaimReconciler` invokes these modules only for tenant-specific
provisioning.

Each module follows the same boundary:

```
ResourceClaimReconciler -> Module -> Backend -> Client -> external service
```

`module.go` validates the common request and delegates to `backend.go`.
`backend.go` decodes claim parameters and implements the provisioning flow.
`client.go` contains backend-specific protocol or CLI integration.

Default backends:

- Redis: native RESP connection and `ACL SETUSER`
- Cassandra: gocql role/keyspace/grant operations
- PostgreSQL: `psql` command integration
- NATS: `nsc` command integration
- S3/MinIO: `mc` command integration

The operator image must contain `psql`, `nsc`, and `mc` when those default
backends are enabled. Custom backends can be injected through `New(backend)`.
