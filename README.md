# Storm Data System

System-level orchestration, end-to-end tests, and documentation for the storm data pipeline. Brings together the collector, ETL, and GraphQL API services into a Kubernetes stack (minikube) with a mock NOAA server for testing.

## How It Works

The stack runs the full data pipeline against mock data:

1. **Mock Server** -- Serves NOAA-format CSV fixtures (hail, tornado, wind) over HTTP
2. **Collector** -- Fetches CSVs from the mock server, parses them, and publishes JSON to Kafka
3. **ETL** -- Consumes raw reports, enriches them (severity, location parsing, time bucketing), and publishes to a downstream Kafka topic
4. **API** -- Consumes enriched events, persists to PostgreSQL, and serves a GraphQL API

```
Mock Server (CSV) --> Collector --> Kafka (raw-weather-reports)
                                        |
                                        v
                                       ETL --> Kafka (transformed-weather-data)
                                                        |
                                                        v
                                                       API --> PostgreSQL --> GraphQL (/query)
```

E2E tests query the GraphQL API to verify that data flows correctly through the entire pipeline.

The stack also includes a **dashboard** (interactive Leaflet map with filters and timeline), **Prometheus** for metrics, and **Kafka UI** for topic inspection.

## Quick Start

### Prerequisites

- [minikube](https://minikube.sigs.k8s.io/docs/start/) (4GB memory, 2 CPUs)
- [kubectl](https://kubernetes.io/docs/tasks/tools/)
- Docker (for building local images)

### Run the full stack

```sh
make up
```

This starts a minikube cluster, installs the Strimzi Kafka operator, builds local images, deploys infrastructure (Kafka + Postgres), and applies application manifests. The collector runs its job once on startup, fetching CSVs from the mock server. Once healthy (~60-90 seconds):

```sh
make port-forward
```

This forwards all services to localhost. In a separate terminal:

| Tool | URL | Description |
|------|-----|-------------|
| Dashboard | [localhost:8000](http://localhost:8000) | Interactive storm data explorer |
| GraphQL API | [localhost:8080/query](http://localhost:8080/query) | Direct GraphQL endpoint |
| Prometheus | [localhost:9090](http://localhost:9090) | Metrics and monitoring |
| Kafka UI | [localhost:8082](http://localhost:8082) | Topic inspection and consumer groups |

### Run E2E tests

E2E tests require `make port-forward` running in a separate terminal.

```sh
make test-e2e          # Reset DB + run tests
make test-e2e-only     # Run tests against an already-running stack
```

### Tear down

```sh
make down              # Delete workloads but keep cluster
make clean             # Delete workloads + PVCs
make stop              # Stop minikube (preserves state)
make destroy           # Delete minikube cluster entirely
```

## Services

| Service        | K8s Resource   | Image                                          | Container Port | Health Check |
| -------------- | -------------- | ---------------------------------------------- | -------------- | ------------ |
| `kafka`        | Strimzi Kafka  | Strimzi-managed (Kafka 4.1.1)                  | 9092           | Strimzi readiness |
| `postgres`     | StatefulSet    | `postgres:16`                                  | 5432           | `pg_isready` |
| `mock-server`  | Deployment     | `storm-data-mock-server:latest` (local build)  | 8080           | `/healthz`   |
| `collector`    | Deployment     | `brendanvinson/storm-data-collector:latest`     | 3000           | `/healthz`   |
| `etl`          | Deployment     | `brendanvinson/storm-data-etl:latest`           | 8080           | `/healthz`   |
| `api`          | Deployment     | `brendanvinson/storm-data-api:latest`           | 8080           | `/healthz`   |
| `dashboard`    | Deployment     | `nginx:1-alpine`                               | 80             | HTTP GET `/` |
| `prometheus`   | Deployment     | `prom/prometheus:v3.2.1`                        | 9090           | `/-/healthy` |
| `kafka-ui`     | Deployment     | `provectuslabs/kafka-ui:latest`                 | 8080           | `/actuator/health` |

## Kafka Topics

| Topic                       | Producer    | Consumer | Description                          |
| --------------------------- | ----------- | -------- | ------------------------------------ |
| `raw-weather-reports`       | Collector   | ETL      | Flat CSV JSON with capitalized keys  |
| `transformed-weather-data`  | ETL         | API      | Enriched storm events with severity, location, time bucketing |

Topics are managed as Strimzi `KafkaTopic` custom resources in `k8s/base/kafka/`.

## Mock Server

A lightweight Go HTTP server that mimics the NOAA Storm Prediction Center CSV endpoint. It matches request URLs by suffix (`_rpts_hail.csv`, `_rpts_torn.csv`, `_rpts_wind.csv`) and serves the corresponding fixture from `mock-server/data/`.

The collector's `REPORTS_BASE_URL` is configured via ConfigMap to point to the mock server's ClusterIP Service. CSV fixtures are named using the NOAA format: `{YYMMDD}_rpts_{type}.csv`.

### Test Fixtures

| File                       | Records | Description                |
| -------------------------- | ------- | -------------------------- |
| `240426_rpts_hail.csv`     | 79      | Hail reports |
| `240426_rpts_torn.csv`     | 149     | Tornado reports |
| `240426_rpts_wind.csv`     | 43      | Wind reports |

Total: **271 records** across **11 states** (real NOAA SPC data from April 26, 2024).

## E2E Tests

Go test suite in `e2e/` that runs against the live stack. Tests use `sync.Once` to poll the GraphQL API for data propagation before running assertions. Queries are scoped to the fixture date (2024-04-26) so stale data from other dates doesn't affect assertions.

| Test                       | Description                                                |
| -------------------------- | ---------------------------------------------------------- |
| `TestServicesHealthy`      | All services respond to `/healthz`                         |
| `TestDataPropagation`      | Data flows through the full pipeline (polls until 271 records appear) |
| `TestReportCounts`         | 79 hail + 149 tornado + 43 wind via `byType` aggregation  |
| `TestStateAggregations`    | State and county breakdowns match expected counts          |
| `TestReportEnrichment`     | All reports have ID, unit, timeBucket, processedAt, geo    |
| `TestSpotCheckHailReport`  | San Saba TX hail: magnitude=1.25, unit=in, sourceOffice=SJT |
| `TestHourlyAggregation`    | Hourly bucket counts sum to totalCount                     |
| `TestEventTypeFilter`      | Filtering by `tornado` returns only tornado reports        |
| `TestMeta`                 | `lastUpdated` and `dataLagMinutes` are populated           |
| `TestPagination`           | Limit/offset pagination returns distinct pages             |
| `TestSeverityFilter`       | Filtering by severity narrows results correctly            |
| `TestSortByMagnitude`      | Reports sort descending by magnitude                       |
| `TestGeoRadiusFilter`      | Geo radius filter returns nearby reports only              |

### Environment Overrides

Tests default to `localhost` URLs. Override with environment variables:

| Variable        | Default                  |
| --------------- | ------------------------ |
| `API_URL`       | `http://localhost:8080`  |
| `COLLECTOR_URL` | `http://localhost:3000`  |
| `ETL_URL`       | `http://localhost:8081`  |

## Development

To build from local source instead of published images, clone the sibling service repos alongside this repo:

```
storm-data-collector/
storm-data-etl/
storm-data-api/
storm-data-shared/          <-- shared Go library (config, observability, retry)
storm-data-system/          <-- this repo
```

### Makefile Targets

```
make start               # Start minikube cluster
make stop                # Stop minikube (preserves state)
make destroy             # Delete minikube cluster entirely
make install-strimzi     # Install Strimzi Kafka operator
make apply-infra         # Deploy Kafka + Postgres
make build-local         # Build and load local images into minikube
make apply-apps          # Deploy application services (dev overlay)
make apply-apps-ci       # Deploy application services (CI overlay)
make up                  # Full stack from nothing (start + infra + apps)
make down                # Delete workloads but keep cluster
make clean               # Delete workloads + PVCs
make reset-db            # Truncate storm_reports and restart collector
make test-e2e            # Reset DB + run E2E tests
make test-e2e-only       # Run E2E tests against running stack
make status              # Show all pods across namespaces
make logs                # Tail all app service logs
make logs-collector      # Tail collector logs
make logs-etl            # Tail ETL logs
make logs-api            # Tail API logs
make port-forward        # Forward all services to localhost
make help                # Show all available targets
```

## K8s Manifest Structure

```
k8s/
  base/
    kustomization.yaml          Kustomize base -- assembles all hailtrace-namespace resources
    namespace.yaml              Namespace definition (hailtrace)

    kafka/
      kafka-cluster.yaml        Strimzi Kafka + KafkaNodePool CRs (deployed to kafka namespace)
      topic-raw.yaml            KafkaTopic: raw-weather-reports
      topic-transformed.yaml    KafkaTopic: transformed-weather-data

    postgres/
      statefulset.yaml          StatefulSet with 1Gi PVC
      service.yaml              Headless Service (clusterIP: None)
      secret.yaml               Credentials (POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB)

    collector/
      deployment.yaml           Deployment (1 replica)
      service.yaml              ClusterIP Service
      configmap.yaml            Kafka brokers, topic, base URL, cron schedule

    etl/
      deployment.yaml           Deployment (1 replica)
      service.yaml              ClusterIP Service
      configmap.yaml            Kafka brokers, source/sink topics, batch config

    api/
      deployment.yaml           Deployment (1 replica)
      service.yaml              ClusterIP Service
      configmap.yaml            Kafka brokers, topic, batch config
      secret.yaml               DATABASE_URL connection string

    mock-server/
      deployment.yaml           Deployment (imagePullPolicy: Never for local image)
      service.yaml              ClusterIP Service

    monitoring/
      deployment.yaml           Prometheus Deployment
      service.yaml              ClusterIP Service
      prometheus-config.yaml    ConfigMap with scrape config

    dashboard/
      deployment.yaml           nginx Deployment with ConfigMap volume mount
      service.yaml              ClusterIP Service
      configmap.yaml            Dashboard HTML served via ConfigMap

    kafka-ui/
      deployment.yaml           Kafka UI Deployment
      service.yaml              ClusterIP Service
      configmap.yaml            Bootstrap server config

  overlays/
    dev/
      kustomization.yaml        Dev overlay -- patches mock-server for local image
    ci/
      kustomization.yaml        CI overlay -- pins published image tags
```

## Design Decisions

### Strimzi for Kafka

Kafka is managed by the [Strimzi operator](https://strimzi.io/) rather than a raw StatefulSet. Strimzi provides `Kafka`, `KafkaNodePool`, and `KafkaTopic` custom resources that handle broker lifecycle, storage, and topic management declaratively. This replaces the `kafka-init` container from the Docker Compose setup -- topics are now Kubernetes resources that the Strimzi entity operator reconciles automatically.

**Why**: Strimzi handles the operational complexity of Kafka on Kubernetes (storage provisioning, rolling updates, listener configuration) through its operator pattern. Topic creation is declarative rather than imperative, eliminating the need for init containers or startup scripts.

### Raw StatefulSet for Postgres

PostgreSQL uses a vanilla StatefulSet with a PersistentVolumeClaim rather than an operator (e.g., CloudNativePG, Zalando).

**Why**: The database has a single replica with no replication, failover, or backup requirements in development. A StatefulSet with a PVC provides stable storage identity without the complexity of a full database operator. The headless Service (`clusterIP: None`) gives the pod a stable DNS name (`postgres-0.postgres.hailtrace.svc`).

### Kustomize overlays

The `k8s/base/` directory contains the canonical resource definitions. Overlays customize for environment:

- **dev**: Patches the mock-server to use a locally-built image (`imagePullPolicy: Never`) loaded into minikube's Docker daemon via `eval $(minikube docker-env)`.
- **ci**: Pins published Docker Hub image tags for the three application services.

**Why**: Kustomize is built into kubectl (no additional tooling). Overlays replace the `compose.ci.yml` override pattern from Docker Compose, providing the same "base + environment patches" model natively in Kubernetes.

### Namespace separation

Infrastructure and application resources are deployed to two namespaces:

- **`kafka`** -- Strimzi operator, Kafka broker, KafkaNodePool, and KafkaTopic resources. Strimzi requires its operator and managed resources in the same namespace.
- **`hailtrace`** -- Everything else: Postgres, application services, monitoring, dashboard. All application ConfigMaps reference Kafka via its cross-namespace DNS name (`kafka-kafka-bootstrap.kafka.svc.cluster.local:9092`).

**Why**: Separating Kafka into its own namespace isolates operator RBAC permissions and CRD lifecycle from application resources. The `hailtrace` namespace owns the application boundary. This mirrors a production pattern where shared infrastructure (message brokers, service meshes) lives in dedicated namespaces.

### Credentials in Secrets

Database credentials and connection strings are stored in Kubernetes Secrets (`postgres-credentials`, `api-credentials`) rather than ConfigMaps. Non-sensitive configuration (Kafka brokers, topic names, log levels) lives in ConfigMaps.

**Why**: Kubernetes Secrets provide the standard separation between sensitive and non-sensitive configuration. Pods reference Secrets via `envFrom.secretRef`, keeping credential management consistent with production patterns where Secrets would be backed by an external secrets store (Vault, AWS Secrets Manager, etc.).

## CI

The CI overlay (`k8s/overlays/ci/`) replaces the dev overlay's local image patches with published Docker Hub image references (`brendanvinson/storm-data-*:latest`). This is the Kustomize equivalent of the former `compose.ci.yml` override.

```sh
make up                  # or: start + install-strimzi + apply-infra + apply-apps-ci
make port-forward        # in a separate terminal
make test-e2e-only
make down
```

## Documentation

See the [project wiki](../../wiki) for detailed documentation:

- [Architecture](../../wiki/Architecture) -- Pipeline design, deployment topology, and improvement roadmap
- [Development](../../wiki/Development) -- Multi-repo workflow, CI conventions, and cross-service patterns

## Project Structure

```
k8s/                    Kubernetes manifests (Kustomize base + overlays)
Makefile                Convenience targets for cluster and stack management

mock-server/
  main.go               Go HTTP server mimicking NOAA CSV endpoints
  Dockerfile            Multi-stage build
  data/                 NOAA-format CSV test fixtures

dashboard/
  index.html            Single-page dashboard (Leaflet map, filters, timeline)

monitoring/
  prometheus/
    prometheus.yml      Scrape config for all three services

e2e/
  e2e_test.go           E2E test suite (13 tests)
  helpers_test.go       GraphQL client, health polling, data propagation gate
```
