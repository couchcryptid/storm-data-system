# Storm Data System

System-level orchestration, end-to-end testing, and documentation for the storm data pipeline. This wiki covers the full system architecture -- how the three services work together, how data flows from NOAA to GraphQL, and how to run and test the complete stack.

## System at a Glance

```
NOAA CSVs ──> Collector ──> Kafka (raw) ──> ETL ──> Kafka (enriched) ──> API ──> PostgreSQL
                                                                          │
                                                                     GraphQL /query
```

Three services, two Kafka topics, one database. Data flows left to right.

## Pages

- [[Architecture]] -- System design, data flow, and tradeoffs
- [[Development]] -- Running the stack, E2E tests, and conventions

## Repositories

| Repository | Language | Description |
|------------|----------|-------------|
| [storm-data-collector](https://github.com/couchcryptid/storm-data-collector) | TypeScript | Fetches NOAA CSVs, publishes raw JSON to Kafka |
| [storm-data-etl](https://github.com/couchcryptid/storm-data-etl) | Go | Enriches raw events (severity, location, time bucketing) |
| [storm-data-api](https://github.com/couchcryptid/storm-data-api) | Go | Persists to PostgreSQL, serves GraphQL API |
| [storm-data-shared](https://github.com/couchcryptid/storm-data-shared) | Go | Shared library: config helpers, observability, retry |
| [storm-data-system](https://github.com/couchcryptid/storm-data-system) | Go | Unified stack, E2E tests, system documentation |

Each service has its own wiki with implementation-level detail. This wiki covers the system as a whole.
