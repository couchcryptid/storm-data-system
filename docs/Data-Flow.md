# Data Flow

End-to-end journey of a storm report from NOAA CSV to GraphQL response. Each stage transforms the data and passes it downstream via Kafka.

## Pipeline Overview

![System Architecture](architecture.excalidraw.svg)

**Data transformation at each stage:**

| Stage | Input | Output | Key Operation |
|-------|-------|--------|---------------|
| Collection | NOAA CSV files | Raw JSON (per row) | Fetch + Parse (cron schedule) |
| ETL | Raw JSON | Enriched JSON (per event) | Enrich + Normalize (severity, location, time bucket, office) |
| Persistence | Enriched JSON | `storm_reports` table | `INSERT ... ON CONFLICT DO NOTHING` |
| Query | GraphQL request | JSON response | `POST /query` |

## Stage 1: Collection

The **collector** (TypeScript) fetches CSV files from the NOAA Storm Prediction Center on a configurable cron schedule. It processes all configured report types (`torn`, `hail`, `wind`) concurrently via `Promise.allSettled()`.

**Input**: NOAA CSV files in the format `{YYMMDD}_rpts_{type}.csv`

```csv
Time,Size,Location,County,State,Lat,Lon,Comments
1510,125,8 ESE Chappel,San Saba,TX,31.02,-98.44,1.25 inch hail reported at Colorado Bend State Park. (SJT)
```

**Output**: One Kafka message per CSV row, published to `raw-weather-reports`. The collector preserves the CSV column names as-is (capitalized keys) and adds a `type` field:

```json
{
  "Time": "1510",
  "Size": "125",
  "Location": "8 ESE Chappel",
  "County": "San Saba",
  "State": "TX",
  "Lat": "31.02",
  "Lon": "-98.44",
  "Comments": "1.25 inch hail reported at Colorado Bend State Park. (SJT)"
}
```

The Kafka message timestamp is set to the fetch time, which the ETL uses as the date base for parsing HHMM time values.

**Error handling**: HTTP 500-599 errors trigger fixed 5-minute interval retries (max 3 attempts). 404 errors are skipped (CSV not published yet). All rows for a report type are published as a single batch.

## Stage 2: Transformation (ETL)

The **ETL** (Go) consumes from `raw-weather-reports`, applies an 11-step enrichment pipeline, and produces to `transformed-weather-data`. See the [ETL Enrichment wiki](https://github.com/couchcryptid/storm-data-etl-service/wiki/Enrichment) for the complete rule set.

**Enrichment steps** (in order):

1. Parse raw JSON into domain types
2. Normalize event type (`hail`, `wind`, `tornado`)
3. Assign default unit per type (`in`, `mph`, `f_scale`)
4. Normalize magnitude (convert legacy hundredths format for hail: `125` becomes `1.25`)
5. Derive severity (`minor`, `moderate`, `severe`, `extreme`) based on type and magnitude
6. Extract NWS source office code from comments (e.g., `(SJT)` at end of string)
7. Parse location string (`8 ESE Chappel` becomes distance=8, direction=ESE, name=Chappel)
8. Derive time bucket (truncate `begin_time` to the hour in UTC)
9. Set `processed_at` timestamp
10. Geocode via Mapbox (optional, feature-flagged)
11. Serialize to JSON

**Output**: One Kafka message per enriched event, published to `transformed-weather-data`:

```json
{
  "id": "a3f8b2c1e7d9...",
  "type": "hail",
  "geo": { "lat": 31.02, "lon": -98.44 },
  "magnitude": 1.25,
  "unit": "in",
  "begin_time": "2026-01-01T15:10:00Z",
  "end_time": "2026-01-01T15:10:00Z",
  "source": "spc",
  "location": {
    "raw": "8 ESE Chappel",
    "name": "Chappel",
    "distance": 8,
    "direction": "ESE",
    "state": "TX",
    "county": "San Saba"
  },
  "comments": "1.25 inch hail reported at Colorado Bend State Park. (SJT)",
  "severity": "moderate",
  "source_office": "SJT",
  "time_bucket": "2026-01-01T15:00:00Z",
  "processed_at": "2026-01-01T22:00:00Z",
  "formatted_address": "",
  "place_name": "",
  "geo_confidence": 0,
  "geo_source": ""
}
```

When geocoding is enabled, the ETL populates `formatted_address`, `place_name`, `geo_confidence`, and `geo_source` with Mapbox results. When disabled, these fields are empty/zero.

**Kafka headers**: `type` (event type) and `processed_at` (RFC 3339 timestamp).

**ID generation**: Deterministic SHA-256 hash of `type|state|lat|lon|time`. The same raw event always produces the same ID, enabling idempotent processing at every downstream stage.

## Stage 3: Persistence (API Consumer)

The **API** (Go) consumes from `transformed-weather-data` and inserts each event into PostgreSQL. The consumer uses manual offset commits -- offsets are committed only after a successful database insert.

**Deduplication**: `INSERT ... ON CONFLICT (id) DO NOTHING`. Because IDs are deterministic, duplicate messages (from at-least-once delivery) are silently ignored.

**Poison pill protection**: Malformed JSON messages have their offsets committed (skipped) to prevent blocking the consumer. See [[Architecture]] for the tradeoff discussion.

## Stage 4: Query (GraphQL API)

Clients query the API via `POST /query` with GraphQL. A single `stormReports` query returns:

- **Paginated reports** with sorting, filtering by type/state/severity/magnitude/location/radius
- **Aggregations**: counts by type, by state (with county breakdown), and by hour
- **Metadata**: `lastUpdated` timestamp and `dataLagMinutes`

See [[API Reference]] for the complete query interface, and [[Data Model]] for the database schema.

## Kafka Topics

| Topic | Producer | Consumer | Message Format | Retention |
|-------|----------|----------|----------------|-----------|
| `raw-weather-reports` | Collector | ETL | Flat CSV JSON (capitalized keys) | Default (7 days) |
| `transformed-weather-data` | ETL | API | Enriched event JSON (snake_case) | Default (7 days) |

Both topics use a single partition in the default configuration. For horizontal scaling, increase partition counts and deploy multiple consumer instances. See [[Architecture]] for scaling considerations.

## Delivery Guarantees

| Guarantee | Implementation |
|-----------|----------------|
| **At-least-once** | Manual offset commit after successful processing in both ETL and API |
| **Idempotent writes** | Deterministic IDs + `ON CONFLICT DO NOTHING` in PostgreSQL |
| **No data loss** | Offsets not committed on processing failure; messages redelivered on restart |
| **Poison pill protection** | Malformed messages logged and skipped (offset committed) |
