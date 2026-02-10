# Configuration

Consolidated configuration reference for the entire storm data stack. Each service is configured via environment variables with sensible defaults for local development. The unified Docker Compose stack in this repository wires all services together with the correct inter-container addresses.

For service-specific configuration details, see each service's wiki:
- [Collector Configuration](https://github.com/couchcryptid/storm-data-collector/wiki/Configuration)
- [ETL Configuration](https://github.com/couchcryptid/storm-data-etl-service/wiki/Configuration)
- [API Configuration](https://github.com/couchcryptid/storm-data-graphql-api/wiki/Configuration)

## Collector (TypeScript)

| Variable | Default | Description |
|----------|---------|-------------|
| `KAFKA_BROKERS` | `localhost:9092` | Kafka broker addresses (comma-separated) |
| `KAFKA_CLIENT_ID` | `csv-producer` | Unique client identifier |
| `KAFKA_TOPIC` | `raw-weather-reports` | Target Kafka topic |
| `REPORTS_BASE_URL` | `https://example.com/` | NOAA CSV base URL (must be valid URL) |
| `REPORT_TYPES` | `torn,hail,wind` | CSV report types to fetch (comma-separated) |
| `CRON_SCHEDULE` | `0 0 * * *` | Cron expression for fetch schedule |
| `LOG_LEVEL` | `info` | `fatal`, `error`, `warn`, `info`, `debug` |

Configuration is validated at startup using **Zod**. Invalid values cause an immediate exit with a descriptive error.

## ETL (Go)

| Variable | Default | Description |
|----------|---------|-------------|
| `KAFKA_BROKERS` | `localhost:9092` | Kafka broker addresses (comma-separated) |
| `KAFKA_SOURCE_TOPIC` | `raw-weather-reports` | Topic to consume raw reports from |
| `KAFKA_SINK_TOPIC` | `transformed-weather-data` | Topic to produce enriched events to |
| `KAFKA_GROUP_ID` | `storm-data-etl` | Consumer group ID |
| `HTTP_ADDR` | `:8080` | Health/metrics HTTP server address |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |
| `LOG_FORMAT` | `json` | `json` or `text` |
| `SHUTDOWN_TIMEOUT` | `10s` | Graceful shutdown deadline (Go duration) |

### Mapbox Geocoding (Optional)

| Variable | Default | Description |
|----------|---------|-------------|
| `MAPBOX_TOKEN` | *(none)* | Mapbox API token; auto-enables geocoding if set |
| `MAPBOX_ENABLED` | auto-detected | Explicit override (`true`/`false`) |
| `MAPBOX_TIMEOUT` | `5s` | HTTP timeout for Mapbox API requests |
| `MAPBOX_CACHE_SIZE` | `1000` | Max LRU cache entries |

Configuration is loaded in `internal/config/config.go` and returns `(*Config, error)` with validation.

## API (Go)

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP server port |
| `DATABASE_URL` | `postgres://storm:storm@localhost:5432/stormdata?sslmode=disable` | PostgreSQL connection string |
| `KAFKA_BROKERS` | `localhost:29092` | Kafka broker addresses |
| `KAFKA_TOPIC` | `transformed-weather-data` | Topic to consume enriched events from |
| `KAFKA_GROUP_ID` | `storm-data-graphql-api` | Consumer group ID |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |
| `LOG_FORMAT` | `json` | `json` or `text` |
| `SHUTDOWN_TIMEOUT` | `10s` | Graceful shutdown deadline |

## Unified Stack Overrides

The `compose.yml` in this repository overrides defaults for inter-container networking:

| Variable | Unified Stack Value | Why |
|----------|-------------------|-----|
| Collector `KAFKA_BROKERS` | `kafka:9092` | Internal Docker network (not host-mapped `29092`) |
| Collector `REPORTS_BASE_URL` | `http://mock-server:8080/` | Points to mock NOAA server (E2E) or real NOAA (production) |
| Collector `CRON_SCHEDULE` | `0 0 1 1 *` | Far-future cron; collector runs once on startup for E2E |
| ETL `KAFKA_BROKERS` | `kafka:9092` | Internal Docker network |
| API `KAFKA_BROKERS` | `kafka:9092` | Internal Docker network |
| API `DATABASE_URL` | `postgres://storm:storm@postgres:5432/stormdata?sslmode=disable` | Docker service name `postgres` |

## Environment Files

| File | Used By | Contents |
|------|---------|----------|
| `.env.kafka` | `kafka` container | KRaft broker config (listeners, controller, replication) |
| `.env.postgres` | `postgres` container | `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB` |

These files are committed to the repository (they contain only local dev credentials).

## Validation Conventions

| Service | Validation Approach | Failure Behavior |
|---------|-------------------|------------------|
| Collector | Zod schema at startup | Exit with structured error |
| ETL | `config.Load()` returns `(*Config, error)` | Exit with error message |
| API | `config.Load()` returns `(*Config, error)` | Exit with error message |

All services fail fast on invalid configuration. No service silently uses zero values for required fields.
