# XSFC Resource Operator

Annotation based runtime injection for Helm charts.

A product Helm chart only declares what it needs, for example Redis or NATS. The operator resolves those needs against `ResourceProvider` objects, injects static environment variables, and creates `ExternalSecret` resources that pull credentials from OpenBao through External Secrets Operator.

`ResourceBinding` has been removed. The default contract is now:

```text
Helm chart annotations -> ResourceProvider -> ExternalSecret -> injected env
```

## Prerequisites

- Kubernetes
- cert-manager
- External Secrets Operator
- OpenBao/Vault
- A `ClusterSecretStore` named `openbao`

## Build and deploy

Build and push an image:

```bash
make docker-build IMG=ghcr.io/eclipse-xfsc/resource-operator:dev
make docker-push IMG=ghcr.io/eclipse-xfsc/resource-operator:dev
```

For kind:

```bash
make docker-build IMG=ghcr.io/eclipse-xfsc/resource-operator:dev
kind load docker-image ghcr.io/eclipse-xfsc/resource-operator:dev
```

Deploy:

```bash
kubectl apply -k config/default
```

Verify:

```bash
kubectl get pods -n xsfc-system
```

## Two apply smoke test

### 1. Apply the producer/platform side

This creates a Redis `ResourceProvider`. It contains static config and the OpenBao/ESO mapping, but no secret values.

```bash
kubectl apply -f examples/redis-provider.yaml
```

The provider says:

```yaml
spec:
  type: redis
  outputs:
    env:
      REDIS_HOST: redis-master.redis.svc.cluster.local
      REDIS_PORT: "6379"
    externalSecrets:
      - targetSecretNameTemplate: "{{ workload }}-redis"
        remoteKeyTemplate: "tenants/{{ namespace }}/redis/{{ workload }}"
```

The external tenant management system must already have written credentials to OpenBao at:

```text
tenants/tenant-a/redis/wallet-api
```

with properties like:

```json
{
  "username": "tenant-a-wallet-api",
  "password": "..."
}
```

### 2. Apply the consumer Helm output

```bash
kubectl apply -f examples/workload.yaml
```

The Deployment only declares its needs on the pod template:

```yaml
spec:
  template:
    metadata:
      annotations:
        inject.xfsc.io/enabled: "true"
        inject.xfsc.io/needs: "redis"
```

## Expected result

The admission webhook patches the Deployment with:

- `REDIS_HOST`
- `REDIS_PORT`
- `REDIS_TLS`
- `REDIS_CLUSTER`
- `REDIS_USERNAME` from `secretKeyRef`
- `REDIS_PASSWORD` from `secretKeyRef`

The workload controller creates the matching `ExternalSecret`:

```bash
kubectl get externalsecret -n tenant-a
kubectl get deployment wallet-api -n tenant-a -o yaml
```

Expected ExternalSecret name:

```text
wallet-api-redis-eso
```

Expected target Kubernetes Secret name:

```text
wallet-api-redis
```

## Helm chart contract

Product charts should not create `ResourceProvider` objects. They should only declare needs:

```yaml
spec:
  template:
    metadata:
      annotations:
        inject.xfsc.io/enabled: "true"
        inject.xfsc.io/needs: "redis,nats,service.xy"
```

Platform charts install providers:

```yaml
kind: ResourceProvider
spec:
  type: nats
  outputs:
    env:
      NATS_URL: nats://nats.infra.svc.cluster.local:4222
    externalSecrets:
      - targetSecretNameTemplate: "{{ workload }}-nats"
        remoteKeyTemplate: "tenants/{{ namespace }}/nats/{{ workload }}"
```

If a chart needs a specific provider instead of type based resolution:

```yaml
inject.xfsc.io/providers: "infra/redis-default"
```

## Responsibilities

Tenant management:

- creates Redis/NATS/Postgres/S3 accounts
- rotates credentials
- writes secrets to OpenBao

XSFC operator:

- resolves `inject.xfsc.io/needs`
- renders provider templates
- creates `ExternalSecret`
- patches workload env

External Secrets Operator:

- reads OpenBao
- creates the Kubernetes Secret

## Watch filtering and lifecycle logs

The Deployment controller only queues workloads where injection is enabled through
`inject.xfsc.io/enabled: "true"`. Deployments without XSFC injection annotations are
filtered at the watch level and produce no consumer logs. Status-only Deployment
updates are ignored.

Lifecycle logs distinguish the operation explicitly:

- `consumer created`, `consumer changed`, `consumer deleted`
- `resource provider created`, `resource provider updated`, `resource provider deleted`
- `generated resource created`, `generated resource updated`

Unchanged generated resources and already reconciled consumers are logged only at
verbosity level 1 (`--zap-log-level=1`) and do not appear in normal info logs.

## Cluster-scoped ResourceProviders

`ResourceProvider` is cluster-scoped. Create it without `metadata.namespace`:

```yaml
apiVersion: resources.xfsc.io/v1alpha1
kind: ResourceProvider
metadata:
  name: hello-provider
spec:
  type: hello
  outputs:
    env:
      HELLO_MESSAGE: "Hello XSFC!"
```

A provider is available to all namespaces by default. Restrict it with either an allow-list:

```yaml
spec:
  allow:
    namespaces:
      - tenant-a
      - tenant-b
```

or a namespace-label selector:

```yaml
spec:
  allow:
    selector:
      resources.xfsc.io/injection-enabled: "true"
```

The selector is matched against labels on the consumer namespace.

### Migration from namespaced providers

Kubernetes does not support changing a CRD scope in place reliably. Back up existing providers, remove the old CRD, install the new cluster-scoped CRD, and recreate providers without a namespace:

```bash
kubectl get resourceproviders -A -o yaml > resourceproviders-backup.yaml
kubectl delete crd resourceproviders.resources.xfsc.io
kubectl apply -f config/crd/resources.xfsc.io_resourceproviders.yaml
kubectl apply -f examples/redis-provider.yaml
```

## Consumer-specific environment prefixes

A consumer can optionally prefix every environment variable injected by the operator. Set the prefix on the pod template together with the other injection annotations:

```yaml
spec:
  template:
    metadata:
      annotations:
        inject.xfsc.io/enabled: "true"
        inject.xfsc.io/needs: "vault,redis"
        inject.xfsc.io/env-prefix: "WALLET"
```

Given a provider that exports `VAULT_ADDR`, `REDIS_HOST`, and `REDIS_PASSWORD`, the resulting container receives:

```text
WALLET_VAULT_ADDR
WALLET_REDIS_HOST
WALLET_REDIS_PASSWORD
```

The prefix applies to static values and secret-backed values. Kubernetes Secret keys remain unchanged; only the environment variable name in the consumer is prefixed. Without `inject.xfsc.io/env-prefix`, environment variables are injected under their original names.

Prefixes must be valid environment-variable identifiers. Leading and trailing underscores are normalized, so `WALLET_` is treated as `WALLET`.

The prefix is included in the managed injection lifecycle. Changing or removing it during a Helm upgrade removes the previously managed environment variables and injects the new names.

## Resource-specific modules

Environment and ExternalSecret injection remains generic. Optional resource-specific behavior is dispatched through the module registry by `ResourceProvider.spec.type`.

```text
Consumer needs redis
        |
        v
Resolve ResourceProvider(type=redis)
        |
        +--> Redis module registered? --> reconcile account/provisioning actions
        |
        +--> Generate ExternalSecret
        |
        +--> Inject environment variables
```

A module implements the following contract:

```go
type Module interface {
    Type() string
    Reconcile(context.Context, modules.Request) (modules.Result, error)
}
```

Modules must be idempotent because the controller may reconcile the same consumer repeatedly. A module can:

- call an external API to create or update an account;
- create resource-specific Kubernetes CRs;
- return generated Kubernetes resources in `modules.Result.Resources`.

Returned resources participate in the operator's ownership and cleanup lifecycle. If a provider stops returning a resource, or the provider is removed, the operator deletes that managed resource.

Modules are registered in `cmd/manager/main.go`:

```go
moduleRegistry := modules.NewRegistry(
    redisModule,
    natsModule,
)
```

When no module is registered for a provider type, module dispatch is a no-op and the normal injection flow continues. Module execution happens in the workload controller, not in the admission webhook, so admission remains side-effect free.

A Redis adapter is included in `internal/modules/redis`. The operator can register it when an `AccountProvisioner` implementation is available:

```go
moduleRegistry := modules.NewRegistry(
    redis.New(redisAccountProvisioner),
)
```

The provisioner decides whether it creates the account directly or returns an account-request CR. When no Redis provisioner is wired, Redis remains a normal injection-only provider.
