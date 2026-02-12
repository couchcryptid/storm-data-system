# Observability

Unified view of health checks, Prometheus metrics, and structured logging across all storm data services. Each service exposes the same operational endpoints and follows consistent patterns.

## Health Endpoints

All three services expose identical operational endpoints:

| Endpoint | Method | Healthy | Unhealthy | Purpose |
|----------|--------|---------|-----------|---------|
| `/healthz` | GET | 200 `{"status": "healthy"}` | -- | Liveness probe (always 200) |
| `/readyz` | GET | 200 `{"status": "ready"}` | 503 `{"status": "not ready"}` | Readiness probe |
| `/metrics` | GET | 200 (Prometheus format) | -- | Prometheus scrape endpoint |

### Readiness Semantics

| Service | Host Port | Ready When |
|---------|-----------|------------|
| Collector | `localhost:3000` | Kafka producer connected |
| ETL | `localhost:8081` | At least one message processed |
| API | `localhost:8080` | PostgreSQL pool responds to ping |

The Go services (ETL and API) implement health endpoints using the [storm-data-shared](https://github.com/couchcryptid/storm-data-shared) `observability` package, which provides `LivenessHandler()`, `ReadinessHandler()`, and the `ReadinessChecker` interface. The collector implements the same contract independently in TypeScript.

In the unified Docker Compose stack, readiness determines whether dependent services start (via health check conditions).

## Prometheus Metrics

Each service uses a namespaced prefix to avoid metric collisions:

| Service | Prefix | Library | Full reference |
|---------|--------|---------|----------------|
| Collector | `storm_collector_` | prom-client (Node.js) | [Collector README](https://github.com/couchcryptid/storm-data-collector#prometheus-metrics) |
| ETL | `storm_etl_` | prometheus/client_golang | [ETL README](https://github.com/couchcryptid/storm-data-etl#prometheus-metrics) |
| API | `storm_api_` | prometheus/client_golang | [API README](https://github.com/couchcryptid/storm-data-api#prometheus-metrics) |

### Scrape Configuration

```yaml
scrape_configs:
  - job_name: 'storm-collector'
    scrape_interval: 60s
    static_configs:
      - targets: ['localhost:3000']

  - job_name: 'storm-etl'
    scrape_interval: 15s
    static_configs:
      - targets: ['localhost:8081']

  - job_name: 'storm-api'
    scrape_interval: 15s
    static_configs:
      - targets: ['localhost:8080']
```

The collector uses a 60-second interval because it only runs once daily. The ETL and API use 15-second intervals for finer-grained visibility.

## Compose Stack Tooling

The unified Docker Compose stack includes Prometheus and Kafka UI for local observability out of the box.

### Prometheus

Available at [http://localhost:9090](http://localhost:9090) when the stack is running. Pre-configured to scrape all three services using `monitoring/prometheus/prometheus.yml`. Use the query examples below or the built-in expression browser to explore metrics.

### Kafka UI

Available at [http://localhost:8082](http://localhost:8082). Provides a web interface for inspecting Kafka topics, messages, consumer groups, and broker configuration. Useful for verifying that the collector is publishing raw events and the ETL is producing enriched events.

## Key Monitoring Queries

### System Health

```promql
# All services running
up{job=~"storm-.*"}

# Pipeline throughput (messages/second)
rate(storm_etl_messages_consumed_total[1m])

# API request rate
rate(storm_api_http_requests_total[1m])
```

### Latency

```promql
# ETL batch processing (p99)
histogram_quantile(0.99, rate(storm_etl_batch_processing_duration_seconds_bucket[5m]))

# API HTTP request latency (p99)
histogram_quantile(0.99, rate(storm_api_http_request_duration_seconds_bucket[5m]))

# API database query latency (p99)
histogram_quantile(0.99, rate(storm_api_db_query_duration_seconds_bucket[5m]))
```

### Error Rates

```promql
# ETL transform error rate
rate(storm_etl_transform_errors_total[5m]) / rate(storm_etl_messages_consumed_total[5m])

# API Kafka consumer error rate
rate(storm_api_kafka_consumer_errors_total[5m])

# Collector job failure rate
rate(storm_collector_job_runs_total{status="failure"}[1h])
```

### Resource Utilization

```promql
# API database connection pool utilization
storm_api_db_pool_connections{state="active"} / storm_api_db_pool_connections{state="total"}

# Geocoding cache hit rate (ETL, when enabled)
rate(storm_etl_geocode_cache_total{result="hit"}[5m])
  / rate(storm_etl_geocode_cache_total[5m])
```

## Structured Logging

All services support configurable log levels via `LOG_LEVEL`:

| Level | Usage |
|-------|-------|
| `debug` | Detailed diagnostics (message offsets, individual inserts) |
| `info` | General events (default) -- job runs, request handling, startup/shutdown |
| `warn` | Retries, backoff, skipped messages, degraded geocoding |
| `error` | Failed operations |

| Service | Library | Format Control |
|---------|---------|---------------|
| Collector | Pino | `LOG_LEVEL` (JSON default, pino-pretty in dev) |
| ETL | `log/slog` via [storm-data-shared](https://github.com/couchcryptid/storm-data-shared) | `LOG_LEVEL` + `LOG_FORMAT` (`json` or `text`) |
| API | `log/slog` via [storm-data-shared](https://github.com/couchcryptid/storm-data-shared) | `LOG_LEVEL` + `LOG_FORMAT` (`json` or `text`) |

## Data Lag Monitoring

The API exposes `dataLagMinutes` via the GraphQL `stormReports` query. This represents the time since the most recent `processed_at` timestamp in PostgreSQL. During storm season, a lag exceeding 2 hours may indicate a pipeline problem.

See [[Architecture]] for the improvement roadmap, including Prometheus alerting on data lag.

## Related

- [Shared Architecture](https://github.com/couchcryptid/storm-data-shared/wiki/Architecture) -- shared observability package (logging, health endpoints)
- [[Architecture]] -- system design and improvement roadmap
- [[Configuration]] -- environment variables including LOG_LEVEL and LOG_FORMAT
- [[Troubleshooting]] -- diagnosing issues using metrics and logs
- [[Demo]] -- accessing Prometheus and Kafka UI
