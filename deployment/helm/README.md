# XFSC Resource Operator Helm Chart

Installs the XFSC Resource Injection and Provisioning Operator.

## Install

From the repository root:

```bash
helm upgrade --install xfsc-resource-operator ./deployment/helm \
  --namespace xfsc-system \
  --create-namespace \
  --set image.repository=harbor.example.com/xfsc/resource-operator \
  --set image.tag=0.1.0
```

## Module configuration

The chart stores module enablement in the Secret `<release>-resource-operator-modules` and mounts it as `/etc/xfsc/modules.yaml`.
Backend endpoints and administrative credentials are not stored here. They are read from the Kubernetes Secret referenced by `ResourceProvider.spec.adminSecretRef`.

```yaml
moduleConfig:
  postgres:
    enabled: true
  redis:
    enabled: true
  cassandra:
    enabled: true
  nats:
    enabled: true
  s3:
    enabled: true
  vault:
    enabled: false
```

## Render

```bash
helm template xfsc-resource-operator ./deployment/helm --namespace xfsc-system
```
