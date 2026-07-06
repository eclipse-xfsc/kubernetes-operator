# XSFC Resource Operator MVP

This scaffold implements the reduced runtime-injection model:

- External tenant management provisions accounts and writes credentials to OpenBao.
- Tenant management creates `ResourceBinding` objects in workload namespaces.
- `ResourceProvider` describes shared infrastructure and static config.
- The operator creates `ExternalSecret` resources and injects static env + secret env refs into workloads.
- A mutating admission webhook injects on CREATE/UPDATE of Deployments/StatefulSets/DaemonSets/Jobs/CronJobs.
- A controller reconciles bindings and can re-patch existing workloads.

No product-specific module system is required for the happy path. Provider-specific provisioning belongs outside this operator or in separate optional provisioners.

## Important labels / annotations

Consumer workloads opt in with:

```yaml
metadata:
  annotations:
    inject.xfsc.io/enabled: "true"
    inject.xfsc.io/types: "redis,nats"
```

Bindings select the concrete consumer and provider.

## Secret flow

```text
Tenant Management -> OpenBao KV -> ExternalSecret -> Kubernetes Secret -> Workload env
```

The operator never writes Kubernetes Secret data and never handles secret values.

## Layout

```text
api/                    Go API types
internal/controller/    ResourceBinding reconciler
internal/injection/     Patch/injection logic
internal/webhook/       Admission webhook
config/                 Kustomize manifests, CRDs, RBAC, webhook
examples/               Example providers, bindings, workloads
```

## Getting Started

Prerequisites

The operator assumes the following components are already installed:

* Kubernetes >= 1.30
* cert-manager
* External Secrets Operator
* OpenBao (or Hashicorp Vault)
* A ClusterSecretStore named openbao

Build the operator image:
```
make docker-build IMG=ghcr.io/eclipse-xfsc/resource-operator:latest
make docker-push IMG=ghcr.io/eclipse-xfsc/resource-operator:latest
```
Deploy the CRDs:
```
kubectl apply -f config/crd/
```
Deploy the operator:
```
kubectl apply -k config/default
```
Verify that the controller is running:
```
kubectl get pods -n xsfc-system
```
Expected output:

NAME                                          READY
xsfc-resource-operator-controller-manager     1/1

⸻

Quick Start

The quickest way to verify the operator is working is to create

1. one ResourceProvider
2. one application (Deployment) consuming it

No additional objects are required.

⸻

1. Create a ResourceProvider
```
apiVersion: resources.xfsc.io/v1alpha1
kind: ResourceProvider
metadata:
  name: redis-default
spec:
  type: redis
  outputs:
    env:
      REDIS_HOST: redis.redis.svc.cluster.local
      REDIS_PORT: "6379"
      REDIS_TLS: "false"
    externalSecrets:
      - name: redis-secret
        secretStoreRef:
          kind: ClusterSecretStore
          name: openbao
        remoteKey: tenants/default/redis/demo
        data:
          - secretKey: REDIS_USERNAME
            property: username
          - secretKey: REDIS_PASSWORD
            property: password
```
Apply:
```
kubectl apply -f provider.yaml
```
⸻

2. Create a Consumer
```
apiVersion: apps/v1
kind: Deployment
metadata:
  name: demo
spec:
  replicas: 1
  selector:
    matchLabels:
      app: demo
  template:
    metadata:
      labels:
        app: demo
      annotations:
        inject.xfsc.io/enabled: "true"
        inject.xfsc.io/provider: redis-default
    spec:
      containers:
      - name: app
        image: nginx
```
Apply:
```
kubectl apply -f consumer.yaml
```
⸻

What should happen?

The admission webhook intercepts the Deployment creation.

It performs the following actions:

1. Resolves the referenced ResourceProvider.
2. Renders all templates.
3. Creates an ExternalSecret.
4. Injects static environment variables.
5. Injects secret-backed environment variables.
6. Adds an annotation containing the rendered configuration hash.

The resulting Pod receives:

REDIS_HOST
REDIS_PORT
REDIS_TLS
REDIS_USERNAME
REDIS_PASSWORD

without the application having to know anything about OpenBao or External Secrets.

⸻

Verify the Injection

The generated ExternalSecret should exist:
```
kubectl get externalsecret
```
Inspect it:
```
kubectl describe externalsecret redis-secret
```
The Deployment should contain the injected environment variables:
```
kubectl get deployment demo -o yaml
```
or inspect the Pod:
```
kubectl exec deploy/demo -- env | grep REDIS
```
Expected output:

REDIS_HOST=redis.redis.svc.cluster.local
REDIS_PORT=6379
REDIS_TLS=false
REDIS_USERNAME=...
REDIS_PASSWORD=...

⸻

How Production Works

In production, credentials are not created by the operator.

Instead, an external Tenant Management system performs the lifecycle:

Tenant created
        │
        ▼
Create Redis/Postgres/NATS account
        │
        ▼
Write credentials to OpenBao
        │
        ▼
Create ResourceProvider (or update binding)
        │
        ▼
Application Deployment
        │
        ▼
Admission Webhook
        │
        ▼
ExternalSecret
        │
        ▼
Kubernetes Secret
        │
        ▼
Injected environment variables

This keeps the operator focused solely on runtime injection while account provisioning, credential rotation, auditing and tenant lifecycle remain external responsibilities.

⸻

Removing the Example
```
kubectl delete deployment demo
kubectl delete resourceprovider redis-default
```