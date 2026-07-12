# XSFC Resource Operator Helm Chart

Installs the XSFC Resource Injection Operator.

## Install

```bash
helm install xsfc-resource-operator ./charts/resource-operator \
  --namespace xsfc-system \
  --create-namespace \
  --set image.repository=harbor.example.com/xsfc/resource-operator \
  --set image.tag=0.1.0
```

## Upgrade

```bash
helm upgrade --install xsfc-resource-operator ./charts/resource-operator \
  --namespace xsfc-system \
  --create-namespace \
  --set image.repository=harbor.example.com/xsfc/resource-operator \
  --set image.tag=0.1.0
```

## Test

```bash
kubectl apply -f examples/hello-provider.yaml
kubectl apply -f examples/hello-deployment.yaml
kubectl exec deploy/hello -- env | grep HELLO_MESSAGE
```

Expected:

```text
HELLO_MESSAGE=Hello XSFC!
```
