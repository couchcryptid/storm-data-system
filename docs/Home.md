# Storm Data System

System-level orchestration, end-to-end testing, and documentation for the storm data pipeline. This wiki covers the full system architecture -- how the three services work together, how data flows from NOAA to GraphQL, and how to run, test, and deploy the complete stack.

## System at a Glance

```
NOAA (SPC)                                                          Clients
    |                                                                  ^
    v                                                                  |
+------------+     +-----------+     +-----------+     +----------+    |
| Collector  |---->|   Kafka   |---->|    ETL    |---->|   Kafka  |----+
| (TypeScript)|    | (raw-     |     |   (Go)    |     | (trans-  |    |
|            |     |  weather- |     |           |     |  formed- |    |
+------------+     |  reports) |     +-----------+     |  weather |    |
                   +-----------+                       |  -data)  |    |
                                                       +----------+    |
                                                            |          |
                                                            v          |
                                                       +----------+   |
                                                       |   API    |---+
                                                       |   (Go)   |
                                                       | GraphQL  |
                                                       +----+-----+
                                                            |
                                                            v
                                                       PostgreSQL
```

Three services, two Kafka topics, one database. Data flows left to right.

## Pages

- [[Architecture]] -- System design, tradeoffs, improvement roadmap, and GCP cost analysis
- [[Data Flow]] -- End-to-end data journey from NOAA CSV to GraphQL response
- [[Data Model]] -- Message shapes, event types, field mapping, and database schema
- [[API Reference]] -- GraphQL types, queries, filters, and aggregations
- [[Configuration]] -- Consolidated environment variables across all services
- [[Deployment]] -- Running the full stack locally and in CI
- [[Development]] -- Multi-repo workflow, prerequisites, and conventions
- [[Testing]] -- Testing strategy from unit tests to E2E validation
- [[Observability]] -- Health checks, Prometheus metrics, and structured logging

## Repositories

| Repository | Language | Description |
|------------|----------|-------------|
| [storm-data-collector](https://github.com/couchcryptid/storm-data-collector) | TypeScript | Fetches NOAA CSVs, publishes raw JSON to Kafka |
| [storm-data-etl-service](https://github.com/couchcryptid/storm-data-etl-service) | Go | Enriches raw events (severity, location, time bucketing) |
| [storm-data-graphql-api](https://github.com/couchcryptid/storm-data-graphql-api) | Go | Persists to PostgreSQL, serves GraphQL API |
| [storm-data-system](https://github.com/couchcryptid/storm-data-system) | Go | Unified stack, E2E tests, system documentation |

Each service has its own wiki with implementation-level detail. This wiki covers the system as a whole.
