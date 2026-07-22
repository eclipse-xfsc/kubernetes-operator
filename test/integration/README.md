# Module integration environment

Start the backing services:

```bash
docker compose -f test/integration/docker-compose.yaml up -d --wait
```

Run all unit tests:

```bash
go test ./...
```

Run integration tests when the required CLIs are installed (`psql`, `nsc`, `mc`):

```bash
XFSC_INTEGRATION=1 go test -tags=integration ./internal/modules/...
```

Inspect created resources:

```bash
docker compose -f test/integration/docker-compose.yaml exec postgres psql -U root -d postgres -c '\du'
docker compose -f test/integration/docker-compose.yaml exec redis redis-cli -a rootpass ACL LIST
docker compose -f test/integration/docker-compose.yaml exec cassandra cqlsh -u cassandra -p cassandra -e 'LIST ROLES'
```

Stop and remove the environment:

```bash
docker compose -f test/integration/docker-compose.yaml down -v
```
