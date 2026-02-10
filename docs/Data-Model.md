# Data Model

The complete data model for storm reports as they move through the storm data pipeline. Data starts as NOAA CSV rows, passes through two Kafka topics, and lands in PostgreSQL where it is served via GraphQL.

## Event Types

| Type | Source CSV | Magnitude Meaning | Default Unit |
|------|-----------|-------------------|--------------|
| `hail` | `*_rpts_hail.csv` | Hail stone diameter | `in` (inches) |
| `wind` | `*_rpts_wind.csv` | Wind speed | `mph` |
| `tornado` | `*_rpts_torn.csv` | F/EF scale rating | `f_scale` |

## Severity Classification

Severity is derived from event type and magnitude during ETL enrichment. A magnitude of `0` produces no severity.

### Hail (inches)

| Magnitude | Severity |
|-----------|----------|
| < 0.75 | minor |
| 0.75 -- 1.49 | moderate |
| 1.50 -- 2.49 | severe |
| >= 2.50 | extreme |

### Wind (mph)

| Magnitude | Severity |
|-----------|----------|
| < 50 | minor |
| 50 -- 73 | moderate |
| 74 -- 95 | severe |
| >= 96 | extreme |

### Tornado (F/EF scale)

| Magnitude | Severity |
|-----------|----------|
| 0 -- 1 | minor |
| 2 | moderate |
| 3 -- 4 | severe |
| >= 5 | extreme |

## Raw Message Shape (Collector Output)

Published to `raw-weather-reports`. One message per CSV row. Keys match CSV column headers. The collector adds a `Type` field to identify the event type.

### Hail

```json
{
  "Type": "hail",
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

### Tornado

```json
{
  "Type": "tornado",
  "Time": "1223",
  "F_Scale": "UNK",
  "Location": "2 N Mcalester",
  "County": "Pittsburg",
  "State": "OK",
  "Lat": "34.96",
  "Lon": "-95.77",
  "Comments": "This tornado moved across the northwest side of McAlester. (TSA)"
}
```

### Wind

```json
{
  "Type": "wind",
  "Time": "1245",
  "Speed": "UNK",
  "Location": "Mcalester",
  "County": "Pittsburg",
  "State": "OK",
  "Lat": "34.94",
  "Lon": "-95.77",
  "Comments": "Large trees and power lines down. (TSA)"
}
```

Note: The magnitude field name varies by type (`Size` for hail, `F_Scale` for tornado, `Speed` for wind). The ETL normalizes these into a unified `measurement.magnitude` field.

## Enriched Message Shape (ETL Output)

Published to `transformed-weather-data`. Normalized, enriched, and ready for persistence.

```json
{
  "id": "a3f8b2c1e7d9...",
  "type": "hail",
  "geo": {
    "lat": 31.02,
    "lon": -98.44
  },
  "measurement": {
    "magnitude": 1.25,
    "unit": "in",
    "severity": "moderate"
  },
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
  "source_office": "SJT",
  "time_bucket": "2026-01-01T15:00:00Z",
  "processed_at": "2026-01-01T22:00:00Z",
  "geocoding": {
    "formatted_address": "",
    "place_name": "",
    "confidence": 0,
    "source": ""
  }
}
```

When geocoding is enabled in the ETL, `geocoding.formatted_address`, `geocoding.place_name`, `geocoding.confidence`, and `geocoding.source` are populated with Mapbox results. When disabled, the `geocoding` object is omitted (all fields are zero-valued with `omitempty`).

**Why nested objects?** Cohesive domain concepts are nested as structs:

- **`geo`**: Coordinates are always used as a pair.
- **`location`**: Six tightly coupled fields -- the ETL parses `raw` into `name`/`distance`/`direction`.
- **`measurement`**: Magnitude, unit, and severity form a semantic chain -- unit depends on event type, severity is derived from magnitude + unit.
- **`geocoding`**: All four fields are set together by the geocoding enrichment step and describe the geocoding process result.

Nesting improves enrichment code readability and maps directly to GraphQL types (`Geo`, `Location`, `Measurement`, `Geocoding`) that gqlgen auto-resolves without field resolvers. The API deserializes nested objects automatically via `json.Unmarshal` and flattens to prefixed DB columns for relational storage and indexing.

### Kafka Headers

| Header | Value | Description |
|--------|-------|-------------|
| `type` | Event type string | `hail`, `wind`, or `tornado` |
| `processed_at` | RFC 3339 timestamp | When enrichment occurred |

### Optional Fields

| Field | When Absent |
|-------|-------------|
| `measurement.severity` | Magnitude is 0 or unmeasured |
| `location.distance` | Report is at the named location (no offset) |
| `location.direction` | Report is at the named location (no offset) |
| `geocoding` | Geocoding disabled or all fields are zero-valued |

## Database Schema

The API flattens nested JSON into PostgreSQL columns:

```sql
CREATE TABLE storm_reports (
    id                          TEXT PRIMARY KEY,
    type                        TEXT NOT NULL,
    geo_lat                     DOUBLE PRECISION NOT NULL,
    geo_lon                     DOUBLE PRECISION NOT NULL,
    measurement_magnitude       DOUBLE PRECISION NOT NULL,
    measurement_unit            TEXT NOT NULL,
    begin_time                  TIMESTAMPTZ NOT NULL,
    end_time                    TIMESTAMPTZ NOT NULL,
    source                      TEXT NOT NULL,
    location_raw                TEXT NOT NULL,
    location_name               TEXT NOT NULL,
    location_distance           DOUBLE PRECISION,
    location_direction          TEXT,
    location_state              TEXT NOT NULL,
    location_county             TEXT NOT NULL,
    comments                    TEXT NOT NULL,
    measurement_severity        TEXT,
    source_office               TEXT NOT NULL,
    time_bucket                 TIMESTAMPTZ NOT NULL,
    processed_at                TIMESTAMPTZ NOT NULL,
    geocoding_formatted_address TEXT NOT NULL DEFAULT '',
    geocoding_place_name        TEXT NOT NULL DEFAULT '',
    geocoding_confidence        DOUBLE PRECISION NOT NULL DEFAULT 0,
    geocoding_source            TEXT NOT NULL DEFAULT '',
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### JSON to Column Mapping

| JSON Path | Database Column |
|-----------|----------------|
| `id` | `id` |
| `type` | `type` |
| `geo.lat` | `geo_lat` |
| `geo.lon` | `geo_lon` |
| `measurement.magnitude` | `measurement_magnitude` |
| `measurement.unit` | `measurement_unit` |
| `measurement.severity` | `measurement_severity` |
| `location.raw` | `location_raw` |
| `location.name` | `location_name` |
| `location.distance` | `location_distance` |
| `location.direction` | `location_direction` |
| `location.state` | `location_state` |
| `location.county` | `location_county` |
| `geocoding.formatted_address` | `geocoding_formatted_address` |
| `geocoding.place_name` | `geocoding_place_name` |
| `geocoding.confidence` | `geocoding_confidence` |
| `geocoding.source` | `geocoding_source` |
| All other fields | Same name (snake_case) |

### Indexes

| Index | Columns | Purpose |
|-------|---------|---------|
| `idx_begin_time` | `begin_time` | Date range queries, ORDER BY |
| `idx_type` | `type` | Filter by event type |
| `idx_state` | `location_state` | Filter by state |
| `idx_severity` | `measurement_severity` | Filter by severity level |
| `idx_type_state_time` | `type, location_state, begin_time` | Composite for common "type + state + time" filter |
| `idx_geo` | `geo_lat, geo_lon` | Bounding box pre-filter for radius queries |

### ID Generation

Event IDs are deterministic SHA-256 hashes of `type|state|lat|lon|time`. This enables:

- **Idempotent inserts**: `ON CONFLICT (id) DO NOTHING` naturally deduplicates
- **No distributed coordination**: Any service can compute the same ID for the same event
- **Replay safety**: Reprocessing the same data produces identical IDs

See [[Architecture]] for the tradeoff analysis on collision risk.
