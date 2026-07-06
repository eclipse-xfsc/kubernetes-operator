# Design

## Decision

The operator intentionally does not provision backing accounts. Account creation and credential rotation are owned by an external tenant-management/provisioning system.

The handover contract is `ResourceBinding`:

- points to the concrete consumer workload
- points to the selected provider
- names the OpenBao remote key
- defines the secret properties that should become env vars
- may add binding-specific static env values

## Why not modules?

A module per product quickly duplicates the same tasks: find provider, render values, create ExternalSecret, patch workload. The only variable part is the declared contract. For runtime injection a generic controller is enough.

Provider-specific logic should exist only in provisioners, not in the injector.

## Admission webhook vs controller

Both are included:

- The webhook mutates workloads at creation/update time.
- The binding controller creates ExternalSecret and also patches existing workloads when a binding changes.

This avoids depending solely on restart/redeploy order.

## Production gaps to finish

- Strong CRD validation with CEL rules.
- Namespace authorization enforcement for `provider.spec.allow`.
- Template rendering for remote keys and env values.
- CronJob pod-template patch path.
- Server-side apply instead of update.
- Proper deep-copy generation via controller-gen.
- Integration tests with envtest.
- Helm chart packaging.
