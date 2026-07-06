# XSFC Resource Operator MVP

Scope: no resource provisioning. External tenant management creates accounts and writes credentials into OpenBao. This operator creates ExternalSecret resources and injects static config plus secret refs into workloads.

Flow:
1. Tenant management creates ResourceBinding or the operator derives it from ResourceProvider + workload annotations.
2. Binding reconciler creates ExternalSecret in the consumer namespace.
3. Workload reconciler patches Deployment/StatefulSet/DaemonSet/Job/CronJob pod template.
4. Static config becomes env values. Secret data becomes env valueFrom secretKeyRef.

Security:
- Never read or write secret values.
- Only create ExternalSecret pointing at OpenBao.
- Cross-namespace provider use requires allowNamespaces or explicit binding.
