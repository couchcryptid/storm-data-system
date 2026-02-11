# Demo

How to run the storm data pipeline and explore it interactively. The unified Docker Compose stack includes a dashboard, Prometheus monitoring, and Kafka UI alongside the three application services.

## Quick Start

```sh
cd storm-data-system
make up          # Build all images from source, start stack
```

Once all services are healthy (typically 30--60 seconds), open:

| Tool | URL | Description |
|------|-----|-------------|
| Dashboard | [localhost:8000](http://localhost:8000) | Interactive storm data explorer |
| GraphQL API | [localhost:8080/query](http://localhost:8080/query) | Direct GraphQL endpoint |
| Prometheus | [localhost:9090](http://localhost:9090) | Metrics and monitoring |
| Kafka UI | [localhost:8082](http://localhost:8082) | Topic inspection and consumer groups |

## Dashboard

The dashboard is a single-page app served by nginx at port 8000. It queries the GraphQL API and renders storm data in real time.

### Features

- **Stats cards**: Total report count plus per-type breakdowns (hail, tornado, wind) with max magnitude
- **Activity timeline**: Stacked bar chart showing report counts by hour, color-coded by event type
- **Event map**: Leaflet map with circle markers sized by severity and colored by event type. Click markers for popup details (type, magnitude, location, county, time, comments)
- **Reports table**: Filterable list of all reports with columns for type, location, state, magnitude, severity, and time
- **Filters**: Dropdown filters for event type, state, and severity that update both the map and table
- **Data freshness**: Badge showing `dataLagMinutes` from the GraphQL `meta` response with color-coded severity (green/yellow/orange/red)
- **GraphQL query panel**: Expandable bottom drawer showing the live query. Enable "Edit" mode to modify and re-run queries against the API

### Demo Data

The mock NOAA server provides **271 records** across **11 states** (real NOAA SPC data from April 26, 2024):

| Type | Records |
|------|---------|
| Hail | 79 |
| Tornado | 149 |
| Wind | 43 |

The collector fetches these on startup, and data flows through the full pipeline (Collector -> Kafka -> ETL -> Kafka -> API -> PostgreSQL) within seconds.

## Observability Tools

### Prometheus (localhost:9090)

Pre-configured to scrape all three services:

- **Collector** (`storm_collector_*`): Job runs, rows processed/published, fetch duration, retry counts
- **ETL** (`storm_etl_*`): Messages consumed/produced, transform errors, processing duration, batch metrics, geocoding stats
- **API** (`storm_api_*`): HTTP requests, Kafka messages consumed, DB query duration, connection pool stats

Example queries to try:

```promql
up{job=~"storm-.*"}                                              # All services running
rate(storm_etl_messages_consumed_total[1m])                      # Pipeline throughput
histogram_quantile(0.99, rate(storm_api_http_request_duration_seconds_bucket[5m]))  # API p99 latency
```

See [[Observability]] for the full metrics reference and monitoring queries.

### Kafka UI (localhost:8082)

Browse Kafka topics, messages, and consumer groups:

- **Topics**: `raw-weather-reports` (collector output) and `transformed-weather-data` (ETL output)
- **Messages**: Inspect individual message payloads and headers
- **Consumer groups**: View `storm-data-etl` and `storm-data-api` consumer lag

## Running Tests

After starting the stack, you can run end-to-end and user acceptance tests:

```sh
make test-e2e-only   # 13 Go E2E tests against the GraphQL API
make test-uat-only   # 44 Playwright tests against the dashboard
```

See [[Testing]] for the full testing strategy.

## Tear Down

```sh
make down            # Stop containers
make clean           # Stop containers + remove volumes
```
