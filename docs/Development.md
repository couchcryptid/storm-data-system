# Development

Multi-repo development workflow for the storm data pipeline. Each service is an independent repository with its own toolchain, tests, and CI. This repo provides the unified Kubernetes stack for E2E testing and system-level validation.

For service-specific development guides:

- [Collector Development](https://github.com/couchcryptid/storm-data-collector/wiki/Development)
- [ETL Development](https://github.com/couchcryptid/storm-data-etl/wiki/Development)
- [API Development](https://github.com/couchcryptid/storm-data-api/wiki/Development)
- [Shared Library Development](https://github.com/couchcryptid/storm-data-shared/wiki/Development)

## Prerequisites

| Tool | Version | Used By |
|------|---------|---------|
| Go | 1.25+ | ETL, API, E2E tests, mock server |
| Node.js | 24+ (LTS) | Collector |
| Docker | Latest | Building images, minikube driver |
| minikube | Latest | Local Kubernetes cluster |
| kubectl | Latest | Cluster management, deployments |
| golangci-lint | Latest | ETL, API linting |
| pre-commit | Latest | Git hooks (optional) |

## Repository Layout

```
~/Projects/hailtrace/
  storm-data-collector/       # TypeScript, Confluent Kafka JS, Vitest
  storm-data-etl/             # Go, hexagonal architecture, kafka-go
  storm-data-api/             # Go, gqlgen, pgx, chi
  storm-data-shared/          # Go, shared library (config, observability, retry)
  storm-data-system/          # Unified K8s stack, E2E tests, docs
```

All five repos should be cloned as siblings under the same parent directory. The mock server is built locally and loaded into minikube's Docker daemon. The shared library is consumed as a Go module dependency, not via relative paths.

## Working on a Single Service

Each service has its own Docker Compose for isolated development:

```sh
# Collector
cd storm-data-collector
cp .env.example .env
docker compose up

# ETL
cd storm-data-etl
cp .env.example .env
docker compose up --build

# API
cd storm-data-api
make docker-up
make run
```

Each service's Compose starts only the infrastructure it needs (Kafka for collector/ETL, Kafka + PostgreSQL for API). This remains the fastest path for iterating on a single service without running the full cluster.

## Working on the Full Stack

Use this repo to bring up the complete pipeline in minikube:

```sh
cd storm-data-system
make up              # Full stack from nothing (cluster + infra + apps)
make status          # Check pod readiness across both namespaces
make port-forward    # Forward all services to localhost (run in separate terminal)
```

Once port-forward is running, the dashboard, GraphQL API, Prometheus, and Kafka UI are accessible at their usual localhost URLs (see README).

Common operations:

```sh
make apply-apps      # Re-deploy application services (dev overlay)
make apply-apps-ci   # Re-deploy using published images (CI overlay)
make reset-db        # Truncate storm_reports and restart collector
make test-e2e        # Reset DB + run E2E tests
make test-e2e-only   # Run tests against running stack (requires port-forward)
make down            # Delete workloads but keep cluster
make clean           # Delete workloads + PVCs
make stop            # Stop minikube (preserves state for later)
make destroy         # Delete minikube cluster entirely
```

See the Makefile for the complete command reference.

## Building Local Images

The mock server is built locally and loaded into minikube's Docker daemon:

```sh
make build-local
```

This runs `docker build` inside minikube's Docker environment (via `eval $(minikube docker-env)`), making the image available to the cluster without a registry push. The mock-server Deployment uses `imagePullPolicy: Never` to reference this local image.

Application service images (collector, ETL, API) are pulled from Docker Hub by default. To test local changes to a service image:

```sh
# Build inside minikube's Docker daemon
eval $(minikube docker-env)
docker build -t brendanvinson/storm-data-api:latest ../storm-data-api

# Restart the deployment to pick up the new image
kubectl rollout restart deployment/api -n hailtrace
```

## Useful kubectl Commands

### Pod status and logs

```sh
kubectl get pods -n hailtrace              # App pods
kubectl get pods -n kafka                  # Kafka pods (Strimzi-managed)
kubectl get pods -A                        # All pods across namespaces

kubectl logs -f deployment/collector -n hailtrace   # Follow collector logs
kubectl logs -f deployment/etl -n hailtrace         # Follow ETL logs
kubectl logs -f deployment/api -n hailtrace         # Follow API logs
kubectl logs -f -l app -n hailtrace --max-log-requests=10  # All app logs
```

### Debugging

```sh
kubectl describe pod <pod-name> -n hailtrace   # Events, conditions, mounts
kubectl describe kafka/kafka -n kafka          # Strimzi Kafka cluster status

kubectl exec -it -n hailtrace postgres-0 -- psql -U storm -d stormdata
kubectl exec -it -n hailtrace deployment/api -- sh
```

### Port forwarding (individual services)

```sh
kubectl port-forward -n hailtrace deployment/api 8080:8080
kubectl port-forward -n hailtrace deployment/dashboard 8000:80
kubectl port-forward -n hailtrace deployment/prometheus 9090:9090
```

### Rolling restarts

```sh
kubectl rollout restart deployment/collector -n hailtrace
kubectl rollout restart deployment/etl -n hailtrace
kubectl rollout restart deployment/api -n hailtrace
```

### Strimzi resources

```sh
kubectl get kafka -n kafka                 # Kafka cluster status
kubectl get kafkatopic -n kafka            # Topic list
kubectl get kafkanodepool -n kafka         # Broker pool status
```

## Shared Library

The [storm-data-shared](https://github.com/couchcryptid/storm-data-shared) Go module provides common infrastructure code used by both Go services. It is imported as a regular Go module dependency (`github.com/couchcryptid/storm-data-shared`).

| Package | What It Provides | Used By |
|---------|-----------------|---------|
| `config` | `EnvOrDefault`, `ParseBrokers`, `ParseBatchSize`, `ParseBatchFlushInterval`, `ParseShutdownTimeout` | ETL, API |
| `observability` | `NewLogger` (slog), `LivenessHandler`, `ReadinessHandler`, `ReadinessChecker` interface | ETL, API |
| `retry` | `NextBackoff`, `SleepWithContext` | ETL, API |

Each service wraps shared functions in thin service-specific adapters. For example, the ETL's `observability.NewLogger(cfg)` calls `sharedobs.NewLogger(cfg.LogLevel, cfg.LogFormat)`, keeping the shared library free of service-specific types.

See the [Shared Library wiki](https://github.com/couchcryptid/storm-data-shared/wiki) for architecture details and configuration reference.

## Cross-Service Conventions

These conventions are shared across all services. They were standardized to ensure consistency across the TypeScript and Go codebases.

### Health Endpoints

All three services expose the same operational endpoints:

| Endpoint | Response (200) | Response (503) | Purpose |
|----------|---------------|----------------|---------|
| `GET /healthz` | `{"status": "healthy"}` | -- | Liveness probe |
| `GET /readyz` | `{"status": "ready"}` | `{"status": "not ready"}` | Readiness probe |
| `GET /metrics` | Prometheus exposition format | -- | Metrics scraping |

These map directly to Kubernetes liveness and readiness probes in each Deployment manifest.

### Readiness Semantics

| Service | Ready When |
|---------|------------|
| Collector | Kafka producer connected |
| ETL | At least one message processed |
| API | PostgreSQL pool responds to ping |

### Metric Namespaces

| Service | Prefix | Examples |
|---------|--------|----------|
| Collector | `storm_collector_` | `storm_collector_job_runs_total` |
| ETL | `storm_etl_` | `storm_etl_messages_consumed_total` |
| API | `storm_api_` | `storm_api_http_requests_total` |

### Logging

| Convention | Collector | ETL | API |
|-----------|-----------|-----|-----|
| Library | Pino | `log/slog` via [storm-data-shared](https://github.com/couchcryptid/storm-data-shared) | `log/slog` via [storm-data-shared](https://github.com/couchcryptid/storm-data-shared) |
| Format control | `LOG_LEVEL` | `LOG_LEVEL` + `LOG_FORMAT` | `LOG_LEVEL` + `LOG_FORMAT` |
| Production format | JSON (default) | JSON (default) | JSON (default) |
| Structured fields | Object first argument | Key-value pairs | Key-value pairs |

### Configuration

| Convention | Collector | ETL | API |
|-----------|-----------|-----|-----|
| Source | Environment variables | Environment variables | Environment variables |
| Shared parsers | -- | [storm-data-shared/config](https://github.com/couchcryptid/storm-data-shared) | [storm-data-shared/config](https://github.com/couchcryptid/storm-data-shared) |
| Validation | Zod schema | `config.Load() (*Config, error)` | `config.Load() (*Config, error)` |
| Failure mode | Exit with Zod error | Exit with error message | Exit with error message |

In Kubernetes, non-sensitive configuration lives in ConfigMaps and sensitive values (database URLs, credentials) in Secrets. Pods load both via `envFrom`.

### Graceful Shutdown

All services follow the same shutdown pattern:

1. Receive `SIGINT` or `SIGTERM`
2. Stop accepting new work (close HTTP server, stop cron/consumer)
3. Drain in-flight operations within `SHUTDOWN_TIMEOUT`
4. Close infrastructure connections (Kafka, database)
5. Log "shutdown complete"

Kubernetes sends `SIGTERM` on pod termination, giving the `terminationGracePeriodSeconds` (default 30s) for shutdown to complete.

### Linting and Pre-commit

| Tool | Collector | ETL | API |
|------|-----------|-----|-----|
| Linter | ESLint + `@typescript-eslint` | golangci-lint (14 linters) | golangci-lint (15 linters) |
| Formatter | Prettier | gofmt + goimports | gofmt + goimports |
| Pre-commit | Husky + lint-staged | `.pre-commit-config.yaml` | `.pre-commit-config.yaml` |
| Secret detection | -- | gitleaks | gitleaks |

All three Go projects (ETL, API, and [shared library](https://github.com/couchcryptid/storm-data-shared)) share a `.golangci.yml` configuration with `gocritic` (diagnostic/style/performance) and `revive` (exported) enabled.

### CI Pipelines

All services have the same CI structure in `.github/workflows/ci.yml`:

| Job | Collector | ETL | API |
|-----|-----------|-----|-----|
| Unit tests | `npm run test:unit` | `make test-unit` | `make test-unit` |
| Lint | `npm run lint` + `typecheck` | `make lint` | `make lint` |
| Build | `npm run build` | `make build` | `make build` |
| SonarCloud | Unit tests with coverage + SonarCloud scan | Unit tests with coverage + SonarCloud scan | Unit tests with coverage + SonarCloud scan |

A separate `release.yml` workflow handles versioning, GitHub releases, and Docker image publishing.

## Making Cross-Service Changes

When a change spans multiple services (e.g., adding a new field to the Kafka message):

1. Update the **ETL** domain types and transform logic
2. Update the **API** model, database migration, store queries, and GraphQL schema
3. Update the **collector** CSV parsing (if the field comes from NOAA data)
4. Update **E2E test fixtures** in `storm-data-system/mock-server/data/`
5. Update **E2E test assertions** in `storm-data-system/e2e/`
6. Open coordinated PRs across affected repos
7. Run `make test-e2e` in this repo to validate the full pipeline

## Related

- [Collector Development](https://github.com/couchcryptid/storm-data-collector/wiki/Development) -- TypeScript build, test, and lint
- [ETL Development](https://github.com/couchcryptid/storm-data-etl/wiki/Development) -- Go build, test, and lint
- [API Development](https://github.com/couchcryptid/storm-data-api/wiki/Development) -- Go build, test, lint, and code generation
- [Shared Development](https://github.com/couchcryptid/storm-data-shared/wiki/Development) -- shared library build and test
- [[Architecture]] -- system design, data flow, and tradeoffs
