# Design

The operator is intentionally annotation-first for Helm users.

## Removed ResourceBinding

`ResourceBinding` was removed because it forced product charts or tenant tooling to create one extra object for the common case. Most bindings are deterministic:

```text
namespace + workload + needed type -> provider + remoteKeyTemplate
```

## Runtime flow

```mermaid
sequenceDiagram
    autonumber
    participant Helm
    participant API as Kubernetes API Server
    participant WH as XSFC Mutating Webhook
    participant OP as XSFC Workload Controller
    participant ESO as External Secrets Operator
    participant Bao as OpenBao
    participant Pod

    Helm->>API: apply ResourceProvider
    Helm->>API: apply Deployment with inject.xfsc.io/needs
    API->>WH: AdmissionReview
    WH->>API: list ResourceProviders
    WH->>WH: resolve provider and patch env
    WH-->>API: JSONPatch
    OP->>API: reconcile Deployment
    OP->>API: create/update ExternalSecret
    ESO->>Bao: read rendered remoteKey
    ESO->>API: create/update Kubernetes Secret
    API->>Pod: start pod with env and secretKeyRef
```

## Provider resolution

The webhook and controller use the same resolution function:

1. Read pod template annotations.
2. If `inject.xfsc.io/providers` is set, match by provider cluster-scoped provider name.
3. Otherwise match `inject.xfsc.io/needs` against `spec.type`.
4. Enforce `spec.allow.namespaces`.
5. Patch static env and secret-backed env.
6. The controller creates ExternalSecrets.

## Secret rule

The operator never creates secret values. It creates only ExternalSecret objects. Credentials must already exist in OpenBao, usually created by tenant management.

## Environment prefixing

Consumer annotations may define `inject.xfsc.io/env-prefix`. Prefixing is applied only to the consumer-facing environment variable name. Secret keys and remote properties are not renamed. Managed-state ownership stores the effective prefixed names, allowing upgrades to remove obsolete variables when the prefix or provider output changes.

## Optional modules

The generic injection pipeline dispatches an optional module after provider resolution and before generated resources are applied. Modules are keyed by `ResourceProvider.spec.type`, are required to be idempotent, and are invoked only by the controller. Admission never invokes modules or performs external side effects.
