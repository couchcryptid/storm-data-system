# Testing

Testing strategy for the storm data pipeline. Tests are organized in three tiers: unit tests within each service, integration tests using testcontainers, and end-to-end tests validating the full pipeline.

## Testing Pyramid

![Testing Pyramid](testing-pyramid.excalidraw.svg)

## Unit Tests

Fast, isolated tests with mocked dependencies. No Docker or external services required.

### Collector (TypeScript)

```sh
cd storm-data-collector
npm run test:unit
```

- Located in `src/**/*.test.ts` (excluding `*.integration.test.ts`)
- Uses Vitest with mocked Kafka, HTTP, and cron dependencies
- Coverage via `@vitest/coverage-v8`

### ETL (Go)

```sh
cd storm-data-etl
make test-unit    # go test ./... -count=1 -race
```

- Co-located `_test.go` files (idiomatic Go)
- Tests transformation logic, severity classification, location parsing, time bucketing
- Uses `clockwork` for deterministic time in tests
- Mock data in `data/mock/`

### API (Go)

```sh
cd storm-data-api
make test-unit    # go test ./internal/... -count=1 -race
```

- Tests model deserialization, enum validation, sort field behavior
- Mock data in `data/mock/` (30 reports: 10 hail, 10 tornado, 10 wind)
- No infrastructure dependencies

## Integration Tests

Test each service against real infrastructure using [testcontainers](https://testcontainers.com/).

### Collector

```sh
cd storm-data-collector
npm run test:integration
```

- Spins up real Kafka via testcontainers-node
- Starts a local HTTP mock server with test CSV data
- Verifies full CSV fetch -> parse -> batch publish -> Kafka consume cycle
- ~60-90 seconds to start containers

### ETL

```sh
cd storm-data-etl
make test-integration    # go test -tags=integration ./internal/integration/...
```

- Uses testcontainers-go for Kafka
- Tests the full extract -> transform -> load pipeline with real Kafka messages
- Uses `integration` build tag to separate from unit tests

### API

```sh
cd storm-data-api
make test-integration    # go test -tags=integration ./internal/integration/...
```

- Uses testcontainers-go for PostgreSQL and Kafka
- Tests store operations (insert, query, filter, aggregate, pagination)
- Tests GraphQL endpoint with real database
- Tests Kafka consumer integration (produce -> consume -> insert -> verify)
- ~1-2 minutes to start containers

### Container Images Used

| Service | Kafka Image | PostgreSQL Image |
|---------|-------------|------------------|
| Collector | KafkaJS testcontainers | -- |
| ETL | `apache/kafka:3.7.0` | -- |
| API | `confluentinc/confluent-local:7.6.0` | `postgres:16` |

Note: The API uses the Confluent Kafka image for integration tests because the testcontainers Kafka module requires Confluent's startup scripts.

## End-to-End Tests

System-level tests in this repository that validate the full pipeline using Docker Compose.

### Running

```sh
cd storm-data-system
make test-e2e        # Start stack + run tests
make test-e2e-only   # Run against already-running stack
```

### Architecture

The E2E tests run against the live Docker Compose stack:

```
Mock Server (CSV) --> Collector --> Kafka (raw) --> ETL --> Kafka (enriched) --> API --> PostgreSQL
                                                                                         |
                                                                                    E2E Tests
                                                                                  (GraphQL queries)
```

1. **Mock server** serves NOAA-format CSV fixtures over HTTP
2. **Collector** fetches CSVs on startup, publishes raw JSON to Kafka
3. **ETL** enriches events, produces to downstream topic
4. **API** consumes, persists to PostgreSQL, serves GraphQL
5. **E2E tests** query the GraphQL API to verify end-to-end correctness

### Data Propagation Gate

Tests use a `sync.Once` pattern to poll the GraphQL API until all 9 test records appear (up to 120 seconds), then run assertions. This avoids flaky timing-dependent tests.

```go
func ensureDataPropagated(t *testing.T) {
    dataReady.Do(func() {
        // Poll until 9 records appear or timeout
    })
}
```

### Test Suite

| Test | What It Verifies |
|------|-----------------|
| `TestServicesHealthy` | All services respond to `/healthz` |
| `TestDataPropagation` | Data flows through the full pipeline (9 records) |
| `TestReportCounts` | 3 hail + 3 tornado + 3 wind via `byType` aggregation |
| `TestStateAggregations` | TX=5, OK=3, NE=1 with county breakdowns |
| `TestReportEnrichment` | All reports have ID, unit, timeBucket, processedAt, geo |
| `TestSpotCheckHailReport` | San Saba TX hail: magnitude=1.25, unit=in, sourceOffice=SJT |
| `TestHourlyAggregation` | Hourly bucket counts sum to totalCount |
| `TestTypeFilter` | Filtering by `tornado` returns exactly 3 reports |
| `TestLastUpdated` | `lastUpdated` and `dataLagMinutes` are populated |

### Test Fixtures

CSV files in `mock-server/data/` follow the NOAA naming format `{YYMMDD}_rpts_{type}.csv`:

| File | Records | States | Description |
|------|---------|--------|-------------|
| `260101_rpts_hail.csv` | 3 | TX | Hail (100--175 hundredths of inch) |
| `260101_rpts_torn.csv` | 3 | OK, NE, TX | Tornado (UNK, EF1) |
| `260101_rpts_wind.csv` | 3 | OK, TX | Wind (UNK, 65, 70 mph) |

Total: **9 records** across **3 states** (TX, OK, NE).

### Environment Overrides

Tests default to `localhost` URLs. Override for non-standard ports:

| Variable | Default |
|----------|---------|
| `API_URL` | `http://localhost:8080` |
| `COLLECTOR_URL` | `http://localhost:3000` |
| `ETL_URL` | `http://localhost:8081` |

## Coverage

### Collector

```sh
npm run test:coverage         # All tests
npm run test:coverage:unit    # Unit only
```

### ETL

```sh
make test-cover    # Generates coverage.out + HTML report
```

### API

```sh
make test-cover    # Generates coverage.out + HTML report
```

## CI Integration

Each service runs unit tests and linting in CI on every push/PR to `main`. Integration tests are available but typically run locally due to Docker requirements. E2E tests can run in CI using published images (see [[Deployment]]).

| Tier | When | Duration | Docker Required |
|------|------|----------|----------------|
| Unit | Every push/PR | Seconds | No |
| Integration | Local / CI with Docker | 1-2 minutes | Yes |
| E2E | Local / CI with Docker | 2-5 minutes | Yes |
