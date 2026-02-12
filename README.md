# Storm Data System

System-level orchestration, end-to-end tests, and documentation for the storm data pipeline. Brings together the collector, ETL, and GraphQL API services into a single Docker Compose stack with a mock NOAA server for testing.

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

- Docker and Docker Compose

### Run the full stack

```sh
make up-ci
```

This pulls published service images from Docker Hub and starts the full stack. The collector runs its job once on startup, fetching CSVs from the mock server. Once healthy (~30-60 seconds):

| Tool | URL | Description |
|------|-----|-------------|
| Dashboard | [localhost:8000](http://localhost:8000) | Interactive storm data explorer |
| GraphQL API | [localhost:8080/query](http://localhost:8080/query) | Direct GraphQL endpoint |
| Prometheus | [localhost:9090](http://localhost:9090) | Metrics and monitoring |
| Kafka UI | [localhost:8082](http://localhost:8082) | Topic inspection and consumer groups |

### Run E2E tests

```sh
make test-e2e-ci     # Starts stack (published images) + runs tests
make test-e2e-only   # Runs tests against an already-running stack
```

### Tear down

```sh
make down            # Stop containers
make clean           # Stop containers + remove volumes
```

## Services

| Service        | Image                                          | Host Port | Internal Port | Health Check |
| -------------- | ---------------------------------------------- | --------- | ------------- | ------------ |
| `kafka`        | `apache/kafka:3.7.0`                           | 29092     | 9092          | Topic list   |
| `postgres`     | `postgres:16`                                  | 5432      | 5432          | `pg_isready` |
| `mock-server`  | `./mock-server`                                | 8090      | 8080          | `/healthz`   |
| `collector`    | `brendanvinson/storm-data-collector:latest`    | 3000      | 3000          | `/healthz`   |
| `etl`          | `brendanvinson/storm-data-etl:latest`          | 8081      | 8080          | `/healthz`   |
| `api`          | `brendanvinson/storm-data-api:latest`          | 8080      | 8080          | `/healthz`   |
| `dashboard`    | `nginx:1-alpine`                               | 8000      | 80            | HTTP GET `/` |
| `prometheus`   | `prom/prometheus:v3.2.1`                       | 9090      | 9090          | `promtool check healthy` |
| `kafka-ui`     | `provectuslabs/kafka-ui:latest`                | 8082      | 8080          | `/actuator/health` |

## Kafka Topics

| Topic                       | Producer    | Consumer | Description                          |
| --------------------------- | ----------- | -------- | ------------------------------------ |
| `raw-weather-reports`       | Collector   | ETL      | Flat CSV JSON with capitalized keys  |
| `transformed-weather-data`  | ETL         | API      | Enriched storm events with severity, location, time bucketing |

## Mock Server

A lightweight Go HTTP server that mimics the NOAA Storm Prediction Center CSV endpoint. It matches request URLs by suffix (`_rpts_hail.csv`, `_rpts_torn.csv`, `_rpts_wind.csv`) and serves the corresponding fixture from `mock-server/data/`.

The collector's `REPORTS_BASE_URL` is configured to point to the mock server. CSV fixtures are named using the NOAA format: `{YYMMDD}_rpts_{type}.csv`.

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

Then use `make up` to build from source:

```
make up              # Start the full stack (builds from local source)
make up-ci           # Start the full stack using published images
make down            # Stop and remove all containers
make clean           # Stop, remove containers, volumes, and orphans
make build           # Build all service images
make test-e2e        # Start stack (from source) + reset DB + run E2E tests
make test-e2e-ci     # Start stack (published images) + reset DB + run E2E tests
make test-e2e-only   # Run E2E tests against an already-running stack
make reset-db        # Truncate storm_reports and restart collector
make ps              # Show running services
make logs            # Tail logs from all services
make logs-collector  # Tail collector logs
make logs-etl        # Tail ETL logs
make logs-api        # Tail API logs
make help            # Show all available targets
```

## CI

The `compose.ci.yml` override replaces local `build:` directives with `image:` references to published Docker Hub images (`brendanvinson/storm-data-*:latest`). Quick Start uses this by default via `make up-ci`. CI pipelines run E2E tests against these same images:

```sh
make up-ci
cd e2e && go test -v -count=1 -timeout 5m ./...
make down
```

## Documentation

See the [project wiki](../../wiki) for detailed documentation:

- [Architecture](../../wiki/Architecture) -- Pipeline design, deployment topology, and improvement roadmap
- [Development](../../wiki/Development) -- Multi-repo workflow, CI conventions, and cross-service patterns

## Project Structure

```
compose.yml             Unified Docker Compose stack (local dev, builds from source)
compose.ci.yml          CI override (published images from Docker Hub)
Makefile                Convenience targets for stack management and testing
.env.*                  Service and infrastructure environment files

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
