# Eclipse XFSC Kubernetes Operator

The XFSC Kubernetes Operator provides two complementary capabilities:

1. **Resource Injection**
2. **Resource Provisioning**

These concerns are intentionally separated.

- **Resource Providers** describe how applications consume infrastructure.
- **Resource Claims** describe dedicated resources that must be provisioned for an application.

The operator automatically injects connection information into workloads while independently provisioning tenant-specific resources such as databases, users, keyspaces, buckets or ACLs.

---

# Architecture

```
                    +----------------------+
                    |  Resource Provider   |
                    +----------+-----------+
                               |
                               |
                needs=redis,nats,...
                               |
                               v
                    +----------------------+
                    | Workload Controller  |
                    +----------+-----------+
                               |
                 +-------------+--------------+
                 |                            |
                 v                            v
         External Secrets               ConfigMaps
                 |                            |
                 +-------------+--------------+
                               |
                               v
                    Environment Injection
                               |
                               v
                        Application Pods


                    +----------------------+
                    |  Resource Claim      |
                    +----------+-----------+
                               |
                               v
                  ResourceClaim Controller
                               |
                 +-------------+--------------+
                 |                            |
                 v                            v
          Tenant Secret                Root Secret
                 |                            |
                 +-------------+--------------+
                               |
                               v
                        Provision Module
                               |
                               v
            Redis / PostgreSQL / Cassandra /
                 NATS / S3 / Custom Services
```

---

# Resource Injection

Applications declare their infrastructure dependencies using annotations.

```yaml
metadata:
  annotations:
    inject.xfsc.io/enabled: "true"
    inject.xfsc.io/needs: "redis,postgres"
    inject.xfsc.io/env-prefix: "APP"
```

The operator resolves every entry inside `needs`.

For every referenced `ResourceProvider` it automatically generates

- ExternalSecret
- ConfigMap
- Environment Variables
- Secret References

The application itself never needs to know where credentials are stored.

---

# Environment Prefix

Optionally every injected variable can be prefixed.

Example:

```yaml
inject.xfsc.io/env-prefix: APP
```

Results in

```
APP_REDIS_HOST
APP_REDIS_PORT
APP_REDIS_USERNAME
APP_REDIS_PASSWORD
```

without modifying the underlying Kubernetes Secret.

---

# Resource Providers

A Resource Provider describes how applications consume a service.

Example:

```yaml
apiVersion: resources.xfsc.io/v1alpha1
kind: ResourceProvider

metadata:
  name: redis-main

spec:

  type: redis

  config:

    REDIS_HOST: redis.redis.svc.cluster.local
    REDIS_PORT: "6379"

  adminSecretRef:

    namespace: infrastructure
    name: redis-root

  secrets:

    - secretKey: username
      env: REDIS_USERNAME

    - secretKey: password
      env: REDIS_PASSWORD
```

A Resource Provider **does not provision anything**.

It only describes

- connection parameters
- configuration
- secret mappings
- root credentials used by the provisioner

---

# Resource Claims

Applications requiring dedicated resources create a ResourceClaim.

Typical examples are

- dedicated PostgreSQL database
- dedicated Redis user
- Cassandra keyspace
- S3 bucket
- NATS account

Example

```yaml
apiVersion: resources.xfsc.io/v1alpha1
kind: ResourceClaim

metadata:
  name: redis-db

spec:

  type: redis

  provider: redis-main

  secretRef:
    name: redis-db-credentials

  parameters:

    database: 1

    acl:

      keys:

      - wallet:*

      commands:

      - +get

      - +set
```

The claim **does not create credentials**.

It only describes the desired resource.

---

# Credential Generation

Credential generation is handled by the application Helm Chart.

Every chart ships an initialization Job.

```
Helm Install

        │

        ▼

OpenBao Init Job

        │

        ├── generate username

        ├── generate password

        └── write KV

        │

        ▼

External Secret

        │

        ▼

Kubernetes Secret
```

This means credentials already exist before the operator provisions the resource.

---

# Provisioning

The ResourceClaim controller watches all ResourceClaims.

Whenever a new claim appears it performs the following steps.

## 1. Read Tenant Secret

The secret referenced by

```yaml
spec:
  secretRef:
```

is loaded.

This secret contains

```
username
password
```

generated by the Helm chart.

---

## 2. Read Root Secret

The operator loads the root credentials defined by the ResourceProvider.

Example

```yaml
adminSecretRef:

  namespace: infrastructure

  name: redis-root
```

Only the operator requires access to these credentials.

---

## 3. Execute Provisioner

The corresponding module provisions the requested resource.

Examples

Redis

- create ACL user
- assign password
- configure permissions

PostgreSQL

- create role
- create database
- grant privileges
- create schemas
- install extensions

Cassandra

- create role
- create keyspace
- grant permissions

NATS

- create account
- create user
- configure permissions

S3

- create user
- create bucket
- create policy

---

## 4. Update Status

Finally the claim is updated.

Example

```yaml
status:

  phase: Ready

  conditions:

  - type: Ready

    status: "True"
```

---

# Separation of Responsibilities

## Helm Chart

Responsible for

- generating credentials
- writing OpenBao secrets
- creating ExternalSecrets
- creating ResourceClaims

The chart never provisions infrastructure resources.

---

## Resource Provider

Responsible for

- describing infrastructure
- environment variables
- configuration
- secret mappings
- root credential location

---

## Resource Claim

Responsible for describing

- desired database
- bucket
- keyspace
- ACL
- permissions
- tenant specific configuration

---

## Operator

Responsible for

- workload injection
- ExternalSecret generation
- ConfigMap generation
- environment injection
- resource provisioning

The operator never generates credentials.

---

# Module Architecture

Every infrastructure type is implemented as a module.

```
internal/modules

    redis/

    postgres/

    cassandra/

    nats/

    s3/
```

Each module consists of two independent parts.

## Injector

Responsible for

- ExternalSecrets
- ConfigMaps
- environment variables

## Provisioner

Responsible for

- databases
- users
- ACLs
- buckets
- keyspaces
- permissions

This separation keeps workload injection independent from infrastructure provisioning.

---

# Typical Workflow

```
Helm Install

    │

    ├── OpenBao Init Job

    ├── ExternalSecret

    ├── ResourceClaim

    └── Deployment

                │

                ▼

        Workload Controller

                │

                ├── ConfigMap

                ├── ExternalSecret

                └── Environment Injection

                │

                ▼

       ResourceClaim Controller

                │

                ├── Read Tenant Secret

                ├── Read Root Secret

                ├── Provision Resource

                └── Ready
```

---

# Benefits

- Complete separation of injection and provisioning
- Credentials generated before provisioning
- No direct dependency on OpenBao inside modules
- Kubernetes-native secret handling
- Extensible module architecture
- Idempotent provisioning
- Fully declarative infrastructure
- Independent application lifecycle
- Easy support for additional resource types