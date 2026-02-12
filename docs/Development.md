# Development

Multi-repo development workflow for the storm data pipeline. Each service is an independent repository with its own toolchain, tests, and CI. This repo provides the unified stack for E2E testing and system-level validation.

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
| Docker | Latest | All services, infrastructure |
| Docker Compose | v2+ | Stack orchestration |
| golangci-lint | Latest | ETL, API linting |
| pre-commit | Latest | Git hooks (optional) |

## Repository Layout

```
~/Projects/hailtrace/
  storm-data-collector/       # TypeScript, Confluent Kafka JS, Vitest
  storm-data-etl/             # Go, hexagonal architecture, kafka-go
  storm-data-api/             # Go, gqlgen, pgx, chi
  storm-data-shared/          # Go, shared library (config, observability, retry)
  storm-data-system/          # Unified stack, E2E tests, docs
```

All five repos should be cloned as siblings under the same parent directory. The unified `compose.yml` references sibling repos via relative paths (`../storm-data-collector`, etc.). The shared library is consumed as a Go module dependency, not via relative paths.

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

Each service's Compose starts only the infrastructure it needs (Kafka for collector/ETL, Kafka + PostgreSQL for API).

## Working on the Full Stack

Use this repo to bring up everything together:

```sh
cd storm-data-system
make up          # Build all images from source, start stack
make test-e2e    # Start stack + reset DB + run E2E tests
make reset-db    # Truncate storm_reports and restart collector
make down        # Tear down
```

See the Makefile for the complete command reference.

## Shared Library

The [storm-data-shared](https://github.com/couchcryptid/storm-data-shared) Go module provides common infrastructure code used by both Go services. It is imported as a regular Go module dependency (`github.com/couchcryptid/storm-data-shared`).

| Package | What It Provides | Used By |
|---------|-----------------|---------|
| `config` | `EnvOrDefault`, `ParseBrokers`, `ParseBatchSize`, `ParseBatchFlushInterval`, `ParseShutdownTimeout` | ETL, API |
| `observability` | `NewLogger` (slog), `LivenessHandler`, `ReadinessHandler`, `ReadinessChecker` interface | ETL, API |
| `retry` | `NextBackoff`, `SleepWithContext` | ETL |

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

### Graceful Shutdown

All services follow the same shutdown pattern:

1. Receive `SIGINT` or `SIGTERM`
2. Stop accepting new work (close HTTP server, stop cron/consumer)
3. Drain in-flight operations within `SHUTDOWN_TIMEOUT`
4. Close infrastructure connections (Kafka, database)
5. Log "shutdown complete"

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
