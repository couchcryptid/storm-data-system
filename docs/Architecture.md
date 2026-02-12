# Architecture

System design, tradeoffs, and improvement roadmap for the storm data pipeline.

## System Overview

![System Architecture](architecture.excalidraw.svg)

Three services, two Kafka topics, one database. Data flows left to right through a collector → ETL → API pipeline. Clients query the GraphQL API on the far right.

## Design Tradeoffs

### Multi-repo over monorepo

Each service is an independent Git repository with its own CI, dependencies, and release cycle.

**Why**: Services are written in different languages (TypeScript, Go) with different toolchains. Independent repos allow independent deployments and avoid coupling release schedules. Each repo has its own Docker Compose for local development of that service in isolation.

**Tradeoff**: Cross-service changes (like adding a new field to the Kafka message schema) require coordinated PRs across repos. The `storm-data-system` repo mitigates this by providing a unified E2E test that validates the full pipeline.

**Alternative considered**: A monorepo with shared tooling. Would simplify cross-cutting changes but introduces coupling between TypeScript and Go toolchains.

### Kafka as the integration layer

Services communicate exclusively through Kafka topics. There are no synchronous service-to-service calls.

**Why**: Decouples producers from consumers. The collector can publish without knowing (or waiting for) downstream processing. The ETL can be restarted, scaled, or replaced without affecting the collector. Kafka provides durability, replay capability, and natural backpressure.

**Tradeoff**: Added infrastructure complexity. Kafka requires careful configuration (replication, retention, partitioning) and consumes significant memory (~512MB--1GB). For the current data volume (hundreds of records per day), this is overprovisioned.

**Alternative considered**: Direct HTTP calls between services, or a simpler queue like Redis Streams. Would reduce infrastructure but sacrifice decoupling and replay capability.

### At-least-once delivery

The ETL and API consumers use manual offset commits. Offsets are committed only after successful processing and writing.

**Why**: Prevents data loss. If a consumer crashes mid-processing, the message is redelivered on restart.

**Tradeoff**: Duplicate processing is possible. The API handles this with `INSERT ... ON CONFLICT (id) DO NOTHING` using deterministic IDs (SHA-256 hash of event_type+state+lat+lon+time+magnitude). The ETL's transform is stateless and idempotent, so duplicates produce identical output.

### Deterministic IDs

Event IDs are SHA-256 hashes of `event_type|state|lat|lon|time|magnitude`. The same raw event always produces the same ID.

**Why**: Enables idempotent writes at every stage. The API's upsert (`ON CONFLICT DO NOTHING`) naturally deduplicates. No distributed ID generation needed.

**Tradeoff**: Two genuinely different events with identical event type, state, coordinates, timestamp, and magnitude would collide. In practice, NOAA data has sufficient uniqueness in these fields. Adding the `Comments` field to the hash would reduce collision risk at the cost of sensitivity to comment edits.

### Cron-based collection

The collector fetches NOAA CSVs on a schedule (daily by default) rather than streaming in real time.

**Why**: NOAA publishes daily CSV files, not a real-time stream. A cron schedule matches the data source's cadence. The collector also runs once immediately on startup for faster feedback during development.

**Tradeoff**: Data freshness is limited to the cron interval. For storm data used in analysis (not real-time alerting), daily collection is sufficient.

### Hexagonal architecture in ETL

The ETL service defines `Extractor`, `Transformer`, and `Loader` interfaces. Kafka adapters implement these interfaces, and the pipeline orchestrates them.

**Why**: Domain logic (parsing, enrichment, severity classification) is completely isolated from infrastructure. Tests can substitute in-memory implementations. The Kafka consumer could be swapped for a file reader or HTTP endpoint without touching business logic.

**Tradeoff**: More files and indirection for what is essentially a single pipeline. For a small service, this is arguably over-engineered. The payoff comes when adding new enrichment steps or swapping infrastructure.

### Schema-first GraphQL (gqlgen)

The API defines the GraphQL schema in `.graphqls` files and generates Go code from it.

**Why**: The schema is the contract. Frontend developers can read the schema without understanding Go. Code generation eliminates boilerplate and ensures the implementation matches the schema.

**Tradeoff**: Schema changes require running `go generate`, and the generated code can be verbose. Runtime reflection-based approaches (like graphql-go) are more flexible but lose compile-time safety.

### Poison pill handling

The ETL skips malformed messages (logs a warning, commits the offset, continues). The API does the same.

**Why**: A single bad message should not block the entire pipeline. Logging the error provides visibility for investigation.

**Tradeoff**: Silently skipping messages can hide data quality issues. A dead letter queue (DLQ) would be better -- see improvements below.

For the complete data model and message shapes at each stage, see the ETL [Enrichment](https://github.com/couchcryptid/storm-data-etl/wiki/Enrichment) rules and the API [Architecture](https://github.com/couchcryptid/storm-data-api/wiki/Architecture) database schema.

## Improvements

Ingest volume is low (hundreds of records per day during storm season) and doesn't need to scale. The improvements below focus on query performance, API reliability, and developer experience.

### Near-term

**Schema registry** -- Introduce Avro or Protobuf schemas with a Confluent Schema Registry (or Buf). Currently the Kafka message format is an implicit JSON contract. A schema registry would catch breaking changes at publish time rather than at consumer parse time.

**OpenTelemetry tracing** -- Add distributed tracing across all three services. Each Kafka message would carry a trace context header, enabling end-to-end latency visualization from ingest to query response.

**Alerting on data lag** -- The API already exposes `dataLagMinutes` via GraphQL. Add Prometheus alerting rules for when data lag exceeds a threshold (e.g., 2 hours during storm season).

### Medium-term

**API response caching** -- Add an in-memory or Redis cache in front of PostgreSQL for frequently-queried time ranges and aggregations. Storm data is append-only, so cache invalidation only needs to happen on new ingestion batches.

**PostgreSQL partitioning** -- Partition the `storm_reports` table by `event_time` (monthly or yearly). This would improve query performance for time-range filters and simplify data retention as the dataset grows.

**GraphQL subscriptions** -- Add WebSocket-based subscriptions so clients can receive real-time updates when new storm reports are ingested. gqlgen supports subscriptions natively.

### Long-term

**Read replicas** -- Add PostgreSQL read replicas to scale query throughput. The API's read-heavy workload (GraphQL queries) is a natural fit for read replicas, while the single writer (Kafka consumer) continues targeting the primary.

**ML enrichment** -- Add a machine learning step to the ETL pipeline for damage estimation, storm path prediction, or severity classification beyond the current rule-based approach.

## Related

- [Collector Architecture](https://github.com/couchcryptid/storm-data-collector/wiki/Architecture) -- CSV fetching, retry strategy, and Kafka publishing
- [ETL Architecture](https://github.com/couchcryptid/storm-data-etl/wiki/Architecture) -- hexagonal design, enrichment pipeline, and offset strategy
- [API Architecture](https://github.com/couchcryptid/storm-data-api/wiki/Architecture) -- GraphQL resolvers, store layer, and query protection
- [Shared Architecture](https://github.com/couchcryptid/storm-data-shared/wiki/Architecture) -- package design and interface contracts
- [[Development]] -- running the stack, E2E tests, and conventions
