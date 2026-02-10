# Observability

Unified view of health checks, Prometheus metrics, and structured logging across all storm data services. Each service exposes the same operational endpoints and follows consistent patterns for metrics and logging.

For service-specific observability details:

- [Collector Metrics](https://github.com/couchcryptid/storm-data-collector/wiki/Metrics) and [Logging](https://github.com/couchcryptid/storm-data-collector/wiki/Logging)
- [ETL Performance](https://github.com/couchcryptid/storm-data-etl/wiki/Performance) (includes monitoring queries)
- [API Performance](https://github.com/couchcryptid/storm-data-api/wiki/Performance) (includes monitoring queries)

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

In the unified Docker Compose stack, readiness determines whether dependent services start (via health check conditions).

## Prometheus Metrics

### Collector (`storm_collector_*`)

**Package**: prom-client (Node.js)

#### Counters

| Metric | Labels | Description |
|--------|--------|-------------|
| `storm_collector_job_runs_total` | `status` (`success`, `failure`) | Total scheduled job runs |
| `storm_collector_rows_processed_total` | `report_type` (`torn`, `hail`, `wind`) | CSV rows parsed |
| `storm_collector_rows_published_total` | `report_type` | Rows published to Kafka |
| `storm_collector_retry_total` | `report_type` | HTTP 5xx retry attempts |
| `storm_collector_kafka_publish_retries_total` | `topic` | Kafka publish retry attempts |

#### Histograms

| Metric | Labels | Buckets (s) | Description |
|--------|--------|-------------|-------------|
| `storm_collector_job_duration_seconds` | -- | 1, 5, 10, 30, 60, 120 | Full job duration |
| `storm_collector_csv_fetch_duration_seconds` | `report_type` | 0.5, 1, 2, 5, 10, 30 | Single CSV fetch + process |

Also includes default Node.js runtime metrics (`nodejs_*`, `process_*`).

### ETL (`storm_etl_*`)

**Package**: prometheus/client_golang

#### Pipeline Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `storm_etl_messages_consumed_total` | Counter | `topic` | Messages consumed from source topic |
| `storm_etl_messages_produced_total` | Counter | `topic` | Messages produced to sink topic |
| `storm_etl_transform_errors_total` | Counter | `error_type` | Transform failures |
| `storm_etl_processing_duration_seconds` | Histogram | -- | Per-message processing time (1ms--5s buckets) |
| `storm_etl_pipeline_running` | Gauge | -- | 1 when pipeline loop is active |

#### Geocoding Metrics (when enabled)

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `storm_etl_geocode_enabled` | Gauge | -- | 1 if geocoding feature is active |
| `storm_etl_geocode_requests_total` | Counter | `method`, `outcome` | API requests by method (forward/reverse) and outcome |
| `storm_etl_geocode_api_duration_seconds` | Histogram | `method` | Mapbox API latency (10ms--5s buckets) |
| `storm_etl_geocode_cache_total` | Counter | `method`, `result` | Cache hits/misses |

### API (`storm_api_*`)

**Package**: prometheus/client_golang

#### HTTP Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `storm_api_http_requests_total` | Counter | `method`, `path`, `status` | Total HTTP requests |
| `storm_api_http_request_duration_seconds` | Histogram | `method`, `path` | Request duration (1ms--5s buckets) |

#### Kafka Consumer Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `storm_api_kafka_messages_consumed_total` | Counter | `topic` | Messages consumed |
| `storm_api_kafka_consumer_errors_total` | Counter | `topic`, `error_type` | Consumer errors |
| `storm_api_kafka_consumer_running` | Gauge | `topic` | 1 when consumer is active |

#### Database Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `storm_api_db_query_duration_seconds` | Histogram | `operation` | Query duration |
| `storm_api_db_pool_connections` | Gauge | `state` | Connection pool stats |

## Prometheus Scrape Configuration

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

## Key Monitoring Queries

### System Health

```promql
# All services running
up{job=~"storm-.*"}

# Pipeline throughput (messages/second)
rate(storm_etl_messages_consumed_total[1m])

# API request rate
rate(storm_api_http_requests_total[1m])

# Data lag (via GraphQL dataLagMinutes field, or approximate via metrics)
time() - storm_api_kafka_messages_consumed_total
```

### Latency

```promql
# ETL per-message processing (p99)
histogram_quantile(0.99, rate(storm_etl_processing_duration_seconds_bucket[5m]))

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

# Geocoding cache hit rate (ETL)
rate(storm_etl_geocode_cache_total{result="hit"}[5m])
  / rate(storm_etl_geocode_cache_total[5m])
```

## Structured Logging

### Log Levels

All services support the same log levels:

| Level | Collector (Pino) | ETL (slog) | API (slog) |
|-------|-----------------|------------|------------|
| `debug` | Detailed diagnostics | Detailed diagnostics | Detailed diagnostics |
| `info` | General events (default) | General events (default) | General events (default) |
| `warn` | Retries, skipped messages | Backoff, degraded geocoding | Skipped messages |
| `error` | Failed operations | Failed operations | Failed operations |

### Log Format

| Environment | Collector | ETL / API |
|-------------|-----------|-----------|
| Development | Pretty-printed (pino-pretty) | `LOG_FORMAT=text` |
| Production | JSON | `LOG_FORMAT=json` |

### What Gets Logged

| Event | Service | Level | Key Fields |
|-------|---------|-------|------------|
| Job start/complete | Collector | info | `duration`, `status`, `report_types` |
| CSV fetch | Collector | info | `url`, `type`, `statusCode`, `rowCount` |
| HTTP retry | Collector | warn | `url`, `attempt`, `statusCode` |
| Message consumed | ETL | debug | `topic`, `partition`, `offset` |
| Transform error | ETL | warn | `event_id`, `error` |
| Geocoding failure | ETL | warn | `event_id`, `method`, `error` |
| Pipeline backoff | ETL | warn | `delay`, `consecutive_failures` |
| HTTP request | API | info | `method`, `path`, `status`, `duration` |
| Kafka message consumed | API | debug | `topic`, `partition`, `offset` |
| DB insert | API | debug | `event_id`, `duration` |
| Poison pill skipped | ETL / API | warn | `topic`, `partition`, `offset`, `error` |
| Shutdown | All | info | `reason` |

## Data Lag Monitoring

The API exposes `dataLagMinutes` via the GraphQL `stormReports` query. This represents the time since the most recent `processed_at` timestamp in PostgreSQL. During storm season, a lag exceeding 2 hours may indicate a pipeline problem.

See [[Architecture]] for the alerting improvement roadmap.
