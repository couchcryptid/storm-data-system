# Research

Analysis of who the system serves, how well it meets their needs, and where targeted improvements would deliver the most value.

## User Stories

Seven personas represent the system's users. Each story maps to a concrete area of the codebase.

### Priya Ramirez -- Emergency Manager, Oklahoma County OEM

Priya manages disaster preparedness for a tornado-prone county. During severe weather she needs rapid situational awareness to coordinate shelter warnings, deploy response teams, and brief county officials.

| ID | Story | Status | Where |
|---|---|---|---|
| PR-01 | Filter by county and state | Done | GraphQL `StormReportFilter.states`/`.counties`; dashboard dropdowns |
| PR-02 | Tornado severity colors on map | Done | ETL `DeriveSeverity`; dashboard marker styling |
| PR-03 | Hourly activity timeline | Done | `byHour` aggregation; dashboard stacked bar chart |
| PR-04 | Data freshness indicator | Done | `dataLagMinutes` field; dashboard badge (green/yellow/orange/red) |
| PR-05 | Click marker to see details | Done | Dashboard popup renders type, magnitude, location, county, time, comments |
| PR-06 | Geographic radius search | Done | `GeoRadiusFilter` with Haversine + bounding-box pre-filter |

### Dev Kowalski -- Backend Engineer, Pipeline Team

Dev maintains and extends the three-service pipeline. Day-to-day, Dev writes code across the collector, ETL, and API repos, runs integration tests locally, monitors metrics, and reviews PRs.

| ID | Story | Status | Where |
|---|---|---|---|
| DK-01 | `make up` starts full stack | Done | `storm-data-system/Makefile` `up` target |
| DK-02 | E2E tests validate full flow | Done | `e2e/e2e_test.go` (13 tests, 271 records) |
| DK-03 | Prometheus metrics on every service | Done | `/metrics` endpoint per service; counters and histograms |
| DK-04 | Documented procedure for adding a new field | Done | [[Common Tasks]] 8-step guide |
| DK-05 | Deterministic SHA-256 IDs | Done | ETL `transform.go`; API `ON CONFLICT DO NOTHING` |
| DK-06 | Kafka UI for topic inspection | Done | Kafka UI container at `:8082` |
| DK-07 | Per-service Docker Compose | Done | Standalone `compose.yml` in each service repo |

### Sable Chen -- NWS Meteorologist & Data Analyst

Sable analyzes severe weather patterns using the GraphQL API. She writes custom queries to study storm distributions, compare hail sizes versus tornado intensity, and export data for research.

| ID | Story | Status | Where |
|---|---|---|---|
| SC-01 | Filter by event type + time range | Done | `StormReportFilter.eventTypes`, `.timeRange` |
| SC-02 | `byState` aggregation with county breakdown | Done | `aggregations.go` CTE; `StateGroup` with nested `CountyGroup` |
| SC-03 | Sort by magnitude descending | Done | `sortBy: MAGNITUDE`, `sortOrder: DESC` |
| SC-04 | Per-type filter overrides | Done | `eventTypeFilters` with per-type `radiusMiles`/`minMagnitude` |
| SC-05 | Pagination with `hasMore` | Done | `limit`/`offset`; `hasMore` computed from `totalCount` |
| SC-06 | `byHour` time-bucket aggregation | Done | `time_bucket` hourly grouping |
| SC-07 | Geocoding enrichment data | Done | ETL `EnrichWithGeocoding`; GraphQL `Geocoding` type with `formattedAddress`, `placeName`, `confidence`, `source` |

### Marcus Obi -- Frontend Developer, Dashboard Team

Marcus builds the interactive dashboard SPA with Leaflet maps, Chart.js timelines, and the GraphQL query panel. He runs Playwright UAT tests to catch UI regressions.

| ID | Story | Status | Where |
|---|---|---|---|
| MO-01 | Map markers with severity sizing | Done | Leaflet circle markers colored by type, sized by severity |
| MO-02 | Dropdown filters update map + table | Done | Filter dropdowns re-query API and refresh both views |
| MO-03 | Stacked bar timeline | Done | `byHour` data rendered as stacked bars (hail/tornado/wind) |
| MO-04 | GraphQL query panel (edit + run) | Done | Expandable drawer with live query, edit mode, run button with timing |
| MO-05 | Playwright UAT tests | Done | 8 spec files (56 tests): date-picker, health, map, query-panel, stats, table, timeline, toolbar |
| MO-06 | Stats cards (totals + max magnitudes) | Done | Four cards: total reports, hail max size, tornado max EF, wind max mph |

### Tanya Flores -- DevOps / Platform Engineer

Tanya owns CI/CD pipelines, Docker image publishing, and infrastructure configuration. She ensures health checks, resource limits, and startup ordering are correct.

| ID | Story | Status | Where |
|---|---|---|---|
| TF-01 | `/healthz` on every service | Done | Health handlers in all 3 services; Docker Compose health checks |
| TF-02 | Resource limits per service | Done | Compose memory limits (Kafka 1GB, Postgres 512MB, API/ETL 256MB) |
| TF-03 | CI runs tests + lint on every push | Done | GitHub Actions `ci.yml` per service |
| TF-04 | Non-root containers | Done | Collector `appuser` (uid 1001); Go services use distroless nonroot |
| TF-05 | Configuration reference | Done | [[Configuration]] wiki page with every env var |
| TF-06 | E2E tests against published images in CI | Done | `e2e.yml` nightly workflow with published images |
| TF-07 | Embedded DB migrations on startup | Done | `go:embed` migrations in API; golang-migrate runs before traffic |

### Jess Nakamura -- Insurance Risk Modeler

Jess uses the GraphQL API to pull hail and wind data for underwriting models. Her automated pipelines query nightly, aggregate severity by state/county, and feed actuarial risk scores.

| ID | Story | Status | Where |
|---|---|---|---|
| JN-01 | Min magnitude filter | Done | `StormReportFilter.minMagnitude`; WHERE `measurement_magnitude >= $n` |
| JN-02 | `byState`/`byCounty` aggregation | Done | CTE in `aggregations.go` |
| JN-03 | Radius search around lat/lon | Done | `GeoRadiusFilter` with Haversine + bounding-box pre-filter |
| JN-04 | `maxMeasurement` in `byEventType` | Done | `MAX(measurement_magnitude)` per event type group |
| JN-05 | Severity classification | Done | ETL rule-based thresholds: hail (in), wind (mph), tornado (EF scale) |
| JN-06 | Query protections | Done | Depth limit 7, complexity limit 600, page size cap 20 |

### Ravi Chatterjee -- Open-Source Contributor

Ravi is a junior developer exploring the project on GitHub. He starts by cloning, running the demo, and reading the wiki. His contributions range from docs fixes to new enrichment rules.

| ID | Story | Status | Where |
|---|---|---|---|
| RC-01 | Comprehensive wiki | Done | 16 system wiki pages + per-service wikis |
| RC-02 | `make up` demo with mock data | Done | Mock NOAA server (271 records); dashboard, Prometheus, Kafka UI |
| RC-03 | Pre-commit hooks | Done | `.pre-commit-config.yaml`: gofmt, lint, gitleaks, etc. |
| RC-04 | Guide for adding enrichment rules | Done | [[Common Tasks]] "Add a New Enrichment Rule" section |
| RC-05 | Troubleshooting wiki | Done | [[Troubleshooting]] covers Kafka, Postgres, Docker, stale images |
| RC-06 | Cross-service conventions | Done | [[Development]] documents health, config, logging, shutdown patterns |

## Remaining Opportunities

A few user stories surfaced small gaps where existing features could be extended with minimal effort.

The dashboard map popup (PR-05) shows type, magnitude, location, county, time, and comments -- but not the geocoded `formattedAddress` from the ETL's Mapbox enrichment. Adding it would turn cryptic NOAA locations like "8 ESE Chappel" into "Chappel, San Saba County, Texas", which is more useful for emergency managers like Priya who need to act on location data quickly.

Similarly, the [[Common Tasks]] enrichment guide (DK-04, RC-04) walks through the steps abstractly but doesn't reference a concrete implementation. The geocoding enrichment is a complete end-to-end example already in the codebase -- linking to it as a worked example would help contributors like Ravi see what each step looks like in practice.

Other improvements from the [[Architecture]] roadmap (schema registry, OpenTelemetry tracing, data lag alerting, API caching, GraphQL subscriptions) are validated by the story analysis but are larger efforts tracked separately.

### Considered and Deferred

| Idea | Why Not |
|---|---|
| Forward geocoding confidence as a filtering dimension | Mock data is 100% reverse-geocoded (all records have coordinates). Confidence is uniformly ~1.0 and provides no discriminating value. |
| PostgreSQL partitioning now | Current dataset is small (hundreds of records/day). Partitioning adds complexity with no measurable query improvement at this scale. Revisit if data grows 100x. |
| Dead letter queue for poison pills | Current volume is low and poison pill rate is near zero with validated upstream data. Logging + skip is sufficient. Revisit if ingestion volume or source diversity increases. |

## Architecture Traceability

Quick reference mapping key architecture decisions back to the personas they serve.

| Decision | Serves | Why It Matters |
|---|---|---|
| Kafka as integration layer | Dev, Tanya, Ravi | Decouples services, enables replay, supports independent scaling |
| Deterministic SHA-256 IDs | Dev, Jess, Sable | Idempotent writes prevent duplicates from Kafka redelivery |
| GraphQL with filtering, sorting, pagination | Sable, Jess, Priya, Marcus | Single flexible endpoint serves analysts and dashboards |
| Per-type filter overrides (`eventTypeFilters`) | Sable, Jess | Heterogeneous searches (different radii/thresholds per type) in one request |
| Geographic radius search (Haversine + bounding box) | Priya, Jess | Emergency managers search around an EOC; risk modelers around insured properties |
| Severity classification in ETL | Priya, Jess, Marcus | Consistent labels drive map coloring, risk scoring, and triage |
| `byHour`/`byState`/`byEventType` aggregations | Priya, Sable, Marcus, Jess | Powers timelines, state analysis, stats cards, and actuarial models |
| Multi-repo with unified system repo | Dev, Tanya, Ravi | Independent CI/deploy per service; system repo provides E2E validation |
| Docker Compose with health checks + ordering | Dev, Tanya, Ravi | One-command local stack; contributors explore without manual setup |
| Prometheus metrics on every service | Dev, Tanya | Diagnose throughput, latency, errors without log spelunking |
| Non-root containers + distroless images | Tanya | Minimal attack surface in production |
| Embedded DB migrations (`go:embed`) | Tanya, Dev | Self-contained binary; no separate migration tooling |
| Data freshness badge (`dataLagMinutes`) | Priya, Marcus | Emergency managers know if data is stale before acting |
| GraphQL query panel (edit + run) | Sable, Marcus, Ravi | Power users learn the API interactively |
| Mock NOAA server with real data | Dev, Ravi, Marcus | Realistic development without depending on live NOAA |
| Pre-commit hooks | Dev, Ravi, Tanya | Catch issues before CI; especially valuable for new contributors |
| Query protections (depth, complexity, page size) | Jess, Sable | Prevents accidental DoS from automated pipelines |

## Related

- [[Architecture]] -- system design and improvement roadmap
- [[Data Flow]] -- end-to-end data journey
- [[Common Tasks]] -- step-by-step guides for adding features
- [[Configuration]] -- environment variables and feature flags
