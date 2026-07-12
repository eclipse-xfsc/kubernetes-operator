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
