# API Reference

The GraphQL API is served by the `storm-data-api` service at `POST /query`. This page documents the public query interface from the perspective of a client consuming the storm data system.

For implementation details, see the [API service wiki](https://github.com/couchcryptid/storm-data-api/wiki/API-Reference).

## Query

### stormReports

The single top-level query. Accepts a filter with required time bounds and returns reports, aggregations, and metadata.

```graphql
query {
  stormReports(filter: {
    timeRange: { from: "2026-01-01T00:00:00Z", to: "2026-01-02T00:00:00Z" }
    eventTypes: [HAIL]
    states: ["TX"]
  }) {
    totalCount
    hasMore
    reports {
      id
      eventType
      measurement { magnitude unit severity }
      geo { lat lon }
      location { name county state }
      comments
      beginTime
      sourceOffice
      geocoding { formattedAddress placeName confidence source }
    }
    aggregations {
      byEventType { eventType count maxMeasurement { magnitude unit } }
      byState { state count counties { county count } }
      byHour { bucket count }
    }
    meta { lastUpdated dataLagMinutes }
  }
}
```

## Result Envelope

### StormReportsResult

The top-level result returned by `stormReports`.

| Field | Type | Description |
|-------|------|-------------|
| `totalCount` | `Int!` | Total matching reports (ignores `limit`/`offset`) |
| `hasMore` | `Boolean!` | Whether more results exist beyond the current page |
| `reports` | `[StormReport!]!` | Matching reports (respects sorting and pagination) |
| `aggregations` | `StormAggregations!` | Aggregated statistics |
| `meta` | `QueryMeta!` | Data freshness metadata |

### StormAggregations

| Field | Type | Description |
|-------|------|-------------|
| `totalCount` | `Int!` | Total matching reports |
| `byEventType` | `[EventTypeGroup!]!` | Report counts grouped by event type |
| `byState` | `[StateGroup!]!` | Report counts grouped by state and county |
| `byHour` | `[TimeGroup!]!` | Report counts grouped by time bucket |

### QueryMeta

| Field | Type | Description |
|-------|------|-------------|
| `lastUpdated` | `DateTime` | Most recent `processedAt` timestamp |
| `dataLagMinutes` | `Int` | Minutes since `lastUpdated` |

## Core Types

### StormReport

| Field | Type | Description |
|-------|------|-------------|
| `id` | `ID!` | Deterministic SHA-256 hash (see [[Data Model]]) |
| `eventType` | `String!` | `hail`, `tornado`, or `wind` |
| `geo` | `Geo!` | Geographic coordinates |
| `measurement` | `Measurement!` | Magnitude, unit, and severity |
| `beginTime` | `DateTime!` | Event start (RFC 3339) |
| `endTime` | `DateTime!` | Event end (RFC 3339) |
| `source` | `String!` | Data source identifier |
| `sourceOffice` | `String!` | NWS office code (e.g., `FWD`, `OAX`, `TSA`) |
| `location` | `Location!` | Location details |
| `comments` | `String!` | Free-text event description |
| `timeBucket` | `DateTime!` | Hourly time bucket for aggregation |
| `processedAt` | `DateTime!` | When the record was processed |
| `geocoding` | `Geocoding!` | Geocoding enrichment results (empty when geocoding disabled) |

### Measurement

| Field | Type | Description |
|-------|------|-------------|
| `magnitude` | `Float!` | Interpretation depends on `unit` (see [[Data Model]]) |
| `unit` | `String!` | `in` (hail), `mph` (wind), `f_scale` (tornado) |
| `severity` | `String` | `minor`, `moderate`, `severe`, `extreme` (nullable) |

### Geo

| Field | Type | Description |
|-------|------|-------------|
| `lat` | `Float!` | Latitude |
| `lon` | `Float!` | Longitude |

### Location

| Field | Type | Description |
|-------|------|-------------|
| `raw` | `String!` | Raw location string (e.g., `8 ESE Chappel`) |
| `name` | `String!` | City/place name |
| `distance` | `Float` | Miles from named location (nullable) |
| `direction` | `String` | Cardinal direction (nullable) |
| `state` | `String!` | Two-letter state code |
| `county` | `String!` | County name |

### Geocoding

| Field | Type | Description |
|-------|------|-------------|
| `formattedAddress` | `String!` | Full address from geocoding (empty if geocoding disabled) |
| `placeName` | `String!` | Short place name from geocoding (empty if geocoding disabled) |
| `confidence` | `Float!` | Geocoding confidence score 0-1 (0 if geocoding disabled) |
| `source` | `String!` | Geocoding method: `forward`, `reverse`, `original`, `failed`, or empty |

## Aggregation Types

### EventTypeGroup

| Field | Type | Description |
|-------|------|-------------|
| `eventType` | `String!` | Event type |
| `count` | `Int!` | Number of reports |
| `maxMeasurement` | `Measurement` | Highest magnitude measurement in group |

### StateGroup

| Field | Type | Description |
|-------|------|-------------|
| `state` | `String!` | Two-letter state code |
| `count` | `Int!` | Number of reports |
| `counties` | `[CountyGroup!]!` | Breakdown by county |

### CountyGroup

| Field | Type | Description |
|-------|------|-------------|
| `county` | `String!` | County name |
| `count` | `Int!` | Number of reports |

### TimeGroup

| Field | Type | Description |
|-------|------|-------------|
| `bucket` | `DateTime!` | Hourly time bucket |
| `count` | `Int!` | Number of reports |

## Enums

### EventType

`HAIL`, `WIND`, `TORNADO`

### Severity

`MINOR`, `MODERATE`, `SEVERE`, `EXTREME`

### SortField

`BEGIN_TIME`, `MAGNITUDE`, `LOCATION_STATE`, `EVENT_TYPE`

### SortOrder

`ASC`, `DESC` (default: `DESC`)

## Filter Options

### StormReportFilter

| Field | Type | Description |
|-------|------|-------------|
| `timeRange` | `TimeRange!` | Time bounds (required) |
| `near` | `GeoRadiusFilter` | Center point and radius for geographic search |
| `states` | `[String!]` | Match any listed state code |
| `counties` | `[String!]` | Match any listed county name |
| `eventTypes` | `[EventType!]` | Global event type filter (enum values) |
| `severity` | `[Severity!]` | Global severity filter (enum values) |
| `minMagnitude` | `Float` | Global minimum magnitude threshold |
| `eventTypeFilters` | `[EventTypeFilter!]` | Per-type overrides (max 3) |
| `sortBy` | `SortField` | Sort field |
| `sortOrder` | `SortOrder` | `ASC` or `DESC` (default: `DESC`) |
| `limit` | `Int` | Max reports (max 20, default 20) |
| `offset` | `Int` | Reports to skip (pagination) |

### TimeRange

| Field | Type | Description |
|-------|------|-------------|
| `from` | `DateTime!` | Events starting at or after this time |
| `to` | `DateTime!` | Events starting before this time (`to` must be after `from`) |

### GeoRadiusFilter

| Field | Type | Description |
|-------|------|-------------|
| `lat` | `Float!` | Center latitude |
| `lon` | `Float!` | Center longitude |
| `radiusMiles` | `Float` | Search radius in miles (default: 20, max: 200) |

### EventTypeFilter

Per-type override. At most 3, no duplicate event types.

| Field | Type | Description |
|-------|------|-------------|
| `eventType` | `EventType!` | Which event type this override applies to |
| `severity` | `[Severity!]` | Override severity filter for this type |
| `minMagnitude` | `Float` | Override minimum magnitude for this type |
| `radiusMiles` | `Float` | Override search radius for this type (max: 200) |

## Example Queries

### Geographic Radius Search

```graphql
query {
  stormReports(filter: {
    timeRange: { from: "2026-01-01T00:00:00Z", to: "2026-01-02T00:00:00Z" }
    near: { lat: 32.75, lon: -97.15, radiusMiles: 20.0 }
  }) {
    totalCount
    reports { id eventType geo { lat lon } location { name state } }
  }
}
```

### Sorted and Paginated

```graphql
query {
  stormReports(filter: {
    timeRange: { from: "2026-01-01T00:00:00Z", to: "2026-01-02T00:00:00Z" }
    eventTypes: [HAIL]
    sortBy: MAGNITUDE
    sortOrder: DESC
    limit: 10
    offset: 0
  }) {
    totalCount
    hasMore
    reports { id measurement { magnitude unit } location { name county state } }
  }
}
```

### Per-Type Overrides

```graphql
query {
  stormReports(filter: {
    timeRange: { from: "2026-01-01T15:00:00Z", to: "2026-01-01T20:00:00Z" }
    near: { lat: 32.75, lon: -97.15 }
    eventTypeFilters: [
      { eventType: HAIL, severity: [SEVERE, MODERATE], minMagnitude: 1.0, radiusMiles: 50.0 }
      { eventType: TORNADO, radiusMiles: 100.0 }
    ]
  }) {
    totalCount
    reports { id eventType measurement { magnitude unit severity } location { name county } comments }
    aggregations { byEventType { eventType count maxMeasurement { magnitude unit } } }
  }
}
```

## Protections

The server enforces limits to prevent pathological queries:

| Protection | Value | Effect |
|------------|-------|--------|
| Depth limit | 7 levels | Rejects deeply nested queries |
| Complexity limit | 600 points | Rejects expensive queries |
| HTTP timeout | 25 seconds | Aborts long-running requests |
| Concurrency limit | 2 concurrent queries | Prevents resource exhaustion |
| Page size | 20 max | Caps report count per request |
| Radius | 200 miles max | Caps geographic search area |
| Event type filters | 3 max | Limits per-type overrides |
