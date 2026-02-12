# Troubleshooting

Common issues and their solutions when running the storm data pipeline.

## Stack Startup

### Kafka fails to start or keeps restarting

**Symptom**: `kafka` container exits or health check never passes.

**Cause**: Kafka requires ~512MB--1GB of memory. Docker Desktop may have insufficient memory allocated.

**Fix**: Ensure Docker Desktop has at least 4GB of memory. Check with `docker stats`. The `compose.yml` reserves 512MB and limits to 1GB for the Kafka container.

### Services start before Kafka is ready

**Symptom**: Collector or ETL logs show Kafka connection errors on startup, then recover.

**Cause**: Normal behavior. Docker Compose health checks enforce startup ordering, but services may attempt connections during the health check interval. All services use `restart: unless-stopped` and will retry.

**Fix**: No action needed -- services self-heal. If startup is too slow, check that `kafka-init` completed successfully: `docker logs kafka-init`.

### PostgreSQL "role does not exist" or "database does not exist"

**Symptom**: API fails to start with a database connection error.

**Fix**: Ensure `.env.postgres` exists with `POSTGRES_USER=storm`, `POSTGRES_PASSWORD=storm`, `POSTGRES_DB=stormdata`. If the volume has stale data, run `make clean` to remove volumes and restart.

## Data Flow

### Data isn't appearing in the API

**Symptom**: The GraphQL API returns zero results after the stack starts.

**Checklist**:

1. **Collector**: Check `docker logs storm-data-collector` for successful CSV fetch and Kafka publish
2. **ETL**: Check `docker logs storm-data-etl` for consumed/produced message counts
3. **API**: Check `docker logs storm-data-api` for consumed messages and database inserts
4. **Topics**: Verify topics exist: `docker exec kafka /opt/kafka/bin/kafka-topics.sh --bootstrap-server localhost:9092 --list`
5. **Consumer groups**: Check consumer lag: `docker exec kafka /opt/kafka/bin/kafka-consumer-groups.sh --bootstrap-server localhost:9092 --group storm-data-api --describe`

**Common causes**:

- Topic name mismatch between services (check `KAFKA_TOPIC` / `KAFKA_SOURCE_TOPIC` / `KAFKA_SINK_TOPIC`)
- The collector hasn't run yet (check `CRON_SCHEDULE` or trigger manually)
- The mock server isn't serving the expected CSV files (check `docker logs mock-server`)

### E2E tests time out waiting for data

**Symptom**: `TestDataPropagation` times out after 120 seconds.

**Fix**: Check that all services are healthy: `make ps`. Look at each service's logs for errors. The most common cause is the collector not fetching data -- check `docker logs storm-data-collector` for the job run.

### Duplicate records or missing records

**Symptom**: Record counts don't match expectations.

**Cause**: At-least-once delivery means duplicates are possible but handled by `ON CONFLICT (id) DO NOTHING`. If records are missing, check ETL transform errors (malformed messages are logged and skipped).

**Fix**: Check `docker logs storm-data-etl | grep -i error` for skipped messages.

## Development

### `make generate` fails in the API

**Symptom**: `go generate ./...` (gqlgen) fails with schema errors.

**Fix**: Ensure the GraphQL schema in `internal/graph/schema.graphqls` is valid. Common issues: missing type definitions, invalid enum values, field type mismatches with `gqlgen.yml` model bindings.

### golangci-lint fails with unfamiliar errors

**Symptom**: Pre-commit or `make lint` reports issues from `gocritic`, `revive`, or `gocyclo`.

**Common fixes**:

- `exitAfterDefer`: Extract the body of `main()` into a `run()` function
- `rangeValCopy`: Use `for i := range slice` with `slice[i]` instead of `for _, v := range slice` for large structs
- `gocyclo` (complexity > 15): Split large functions into smaller ones
- `nestingReduce`: Invert `if` conditions and use early `continue`/`return`

### Mock data out of sync

**Symptom**: Unit tests fail after changing ETL transform logic.

**Fix**: Regenerate mock data with `genmock` and validate with `validate`. See [[Common Tasks]] for the commands.

## Docker

### Port conflicts

**Symptom**: `docker compose up` fails with "port already in use".

**Ports used by the stack**:

| Port | Service |
|------|---------|
| 29092 | Kafka (host access) |
| 5432 | PostgreSQL |
| 8090 | Mock server |
| 3000 | Collector |
| 8081 | ETL |
| 8080 | API |
| 8000 | Dashboard |
| 9090 | Prometheus |
| 8082 | Kafka UI |

**Fix**: Stop the conflicting process or change the host port in `compose.yml`.

### Stale images after code changes

**Symptom**: Changes to service code aren't reflected after `docker compose up`.

**Fix**: Use `make up` (which includes `--build`) to rebuild images. Or explicitly: `docker compose build --no-cache <service>`.

### Container uses too much memory

**Symptom**: Container killed by OOM or Docker Desktop becomes slow.

**Fix**: The `compose.yml` sets memory limits. If you need to reduce total usage, stop services you don't need: `docker compose up kafka postgres api` (skip collector, ETL, monitoring).

## Related

- [[Observability]] -- metrics and logs for diagnosing issues
- [[Configuration]] -- environment variables and validation
- [[Common Tasks]] -- step-by-step guides for common operations
- [[Deployment]] -- Docker Compose setup and port mapping
