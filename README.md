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

## Quick Start

### Prerequisites

- Go 1.25+
- Docker and Docker Compose
- Sibling service repos cloned alongside this repo:

```
~/Projects/hailtrace/
  storm-data-collector/
  storm-data-etl-service/
  storm-data-graphql-api/
  storm-data-system/          <-- this repo
```

### Run the full stack

```sh
make up
```

This builds all service images from source and starts the full stack. The collector runs its job once on startup, fetching CSVs from the mock server.

### Run E2E tests

```sh
make test-e2e        # Starts stack + runs tests
make test-e2e-only   # Runs tests against an already-running stack
```

### Tear down

```sh
make down            # Stop containers
make clean           # Stop containers + remove volumes
```

## Services

| Service        | Image / Build                | Host Port | Internal Port | Health Check |
| -------------- | ---------------------------- | --------- | ------------- | ------------ |
| `kafka`        | `apache/kafka:3.7.0`         | 29092     | 9092          | Topic list   |
| `postgres`     | `postgres:16`                | 5432      | 5432          | `pg_isready` |
| `mock-server`  | `./mock-server`              | 8090      | 8080          | `/healthz`   |
| `collector`    | `../storm-data-collector`    | 3000      | 3000          | `/healthz`   |
| `etl`          | `../storm-data-etl-service`  | 8081      | 8080          | `/healthz`   |
| `api`          | `../storm-data-graphql-api`  | 8080      | 8080          | `/healthz`   |

## Kafka Topics

| Topic                       | Producer    | Consumer | Description                          |
| --------------------------- | ----------- | -------- | ------------------------------------ |
| `raw-weather-reports`       | Collector   | ETL      | Flat CSV JSON with capitalized keys  |
| `transformed-weather-data`  | ETL         | API      | Enriched storm events with severity, location, time bucketing |

## Mock Server

A lightweight Go HTTP server that mimics the NOAA Storm Prediction Center CSV endpoint. It matches request URLs by suffix (`_rpts_hail.csv`, `_rpts_torn.csv`, `_rpts_wind.csv`) and serves the corresponding fixture from `mock-server/data/`.

The collector's `REPORTS_BASE_URL` is configured to point to the mock server. CSV fixtures are named using the NOAA format: `{YYMMDD}_rpts_{type}.csv`.

### Test Fixtures

| File                       | Records | States  | Description                |
| -------------------------- | ------- | ------- | -------------------------- |
| `260101_rpts_hail.csv`     | 3       | TX      | Hail reports (100--175 hundredths of inch) |
| `260101_rpts_torn.csv`     | 3       | OK, NE, TX | Tornado reports (UNK, EF1) |
| `260101_rpts_wind.csv`     | 3       | OK, TX  | Wind reports (UNK, 65, 70 mph) |

Total: **9 records** across **3 states** (TX, OK, NE).

## E2E Tests

Go test suite in `e2e/` that runs against the live stack. Tests use `sync.Once` to poll the GraphQL API for data propagation before running assertions.

| Test                       | Description                                                |
| -------------------------- | ---------------------------------------------------------- |
| `TestServicesHealthy`      | All services respond to `/healthz`                         |
| `TestDataPropagation`      | Data flows through the full pipeline (polls until 9 records appear) |
| `TestReportCounts`         | 3 hail + 3 tornado + 3 wind via `byType` aggregation      |
| `TestStateAggregations`    | TX=5, OK=3, NE=1 with county breakdowns                   |
| `TestReportEnrichment`     | All reports have ID, unit, timeBucket, processedAt, geo    |
| `TestSpotCheckHailReport`  | San Saba TX hail: magnitude=1.25, unit=in, sourceOffice=SJT |
| `TestHourlyAggregation`    | Hourly bucket counts sum to totalCount                     |
| `TestTypeFilter`           | Filtering by `tornado` returns exactly 3 reports           |
| `TestLastUpdated`          | `lastUpdated` and `dataLagMinutes` are populated           |

### Environment Overrides

Tests default to `localhost` URLs. Override with environment variables:

| Variable        | Default                  |
| --------------- | ------------------------ |
| `API_URL`       | `http://localhost:8080`  |
| `COLLECTOR_URL` | `http://localhost:3000`  |
| `ETL_URL`       | `http://localhost:8081`  |

## Development

```
make up              # Start the full stack (builds from local source)
make up-ci           # Start the full stack using published images
make down            # Stop and remove all containers
make clean           # Stop, remove containers, volumes, and orphans
make build           # Build all service images
make test-e2e        # Start stack + run E2E tests
make test-e2e-only   # Run E2E tests against an already-running stack
make ps              # Show running services
make logs            # Tail logs from all services
make logs-collector  # Tail collector logs
make logs-etl        # Tail ETL logs
make logs-api        # Tail API logs
make help            # Show all available targets
```

## CI

For CI environments where service images are published to a container registry:

```sh
docker compose -f compose.yml -f compose.ci.yml up -d --wait
cd e2e && go test -v -count=1 -timeout 5m ./...
docker compose -f compose.yml -f compose.ci.yml down -v
```

The `compose.ci.yml` override replaces `build:` directives with `image:` references to `ghcr.io/couchcryptid/storm-data-*:latest`.

## Project Structure

```
compose.yml             Unified Docker Compose stack (local dev, builds from source)
compose.ci.yml          CI override (published images from GHCR)
Makefile                Convenience targets for stack management and testing
.env.kafka              Kafka KRaft broker configuration
.env.postgres           PostgreSQL credentials

mock-server/
  main.go               Go HTTP server mimicking NOAA CSV endpoints
  Dockerfile            Multi-stage build (distroless + busybox)
  data/                 NOAA-format CSV test fixtures

e2e/
  e2e_test.go           E2E test suite (9 tests)
  helpers_test.go       GraphQL client, health polling, data propagation gate
  go.mod                Go module
```
