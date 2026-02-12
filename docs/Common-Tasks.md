# Common Tasks

Step-by-step guides for the most common development tasks that span multiple services. For single-service tasks (build, test, lint), see each service's Development wiki page.

## Add a New Field End-to-End

Adding a field that flows from NOAA CSV through to the GraphQL API touches all three services plus E2E tests.

### 1. ETL: Add the domain field

- Add the field to `StormEvent` in `internal/domain/event.go`
- Add parsing/enrichment logic in `internal/domain/transform.go`
- Add unit tests for the new field in `internal/domain/transform_test.go`
- Run `make test-unit` to verify

### 2. ETL: Update mock data

- Regenerate mock fixtures with the new field:

```sh
cd storm-data-etl
go run ./cmd/genmock \
  -csv-dir ../storm-data-system/mock-server/data \
  -etl-out data/mock/storm_reports_240426_combined.json \
  -api-out ../storm-data-api/data/mock/storm_reports_240426_transformed.json
```

This updates both the ETL and API mock data files in one step.

### 3. API: Add the database column

- Create a new migration file in `internal/database/migrations/` (e.g., `003_add_field.sql`)
- Add `ALTER TABLE storm_reports ADD COLUMN ...`
- Add the column to the `INSERT` statement in `internal/store/store.go`
- Add the column to the `SELECT` scan in `internal/store/store.go`

### 4. API: Update the GraphQL schema

- Add the field to the appropriate type in `internal/graph/schema.graphqls`
- Regenerate the GraphQL code:

```sh
cd storm-data-api
make generate
```

- Update the resolver if the new field requires custom mapping (most fields map automatically via `gqlgen.yml` model binding)

### 5. API: Update the model

- Add the field to `StormReport` in `internal/model/storm_report.go`
- Run `make test-unit` to verify mock data deserialization still works

### 6. Collector (if the field comes from NOAA CSV)

- The collector preserves CSV column names as-is. If the field already exists in the CSV, no collector changes are needed.
- If it requires a new derived field, add it in `src/csv/csvStream.ts`

### 7. E2E: Update test fixtures and assertions

- If the field comes from CSV, update `mock-server/data/*.csv` fixtures
- Update E2E test assertions in `e2e/e2e_test.go` to verify the new field
- Run `make test-e2e` from the system repo to validate the full pipeline

### 8. Validate

```sh
cd storm-data-etl
go run ./cmd/validate \
  -source-dir ../storm-data-system/mock-server/data \
  -collector-dir . \
  -etl-json data/mock/storm_reports_240426_combined.json \
  -api-json ../storm-data-api/data/mock/storm_reports_240426_transformed.json
```

## Add a New Enrichment Rule

Enrichment rules live in the ETL's domain package and are applied during the transform stage.

### 1. Add the rule

- Add the function in `internal/domain/transform.go`
- Follow the existing pattern: pure function, takes a `*StormEvent`, returns nothing (mutates in place)
- Call the new function from `EnrichStormEvent()` in the correct order

### 2. Test the rule

- Add test cases in `internal/domain/transform_test.go`
- Use table-driven tests with edge cases

### 3. Regenerate mock data

- Run `genmock` (see above) to update fixtures with the new enrichment applied
- Run `validate` to verify consistency

### 4. Update downstream if needed

- If the enrichment produces a new field, follow the "Add a New Field" steps above for the API
- If it modifies an existing field's behavior, update E2E test assertions

## Update Mock Data

The `genmock` tool in the ETL repo is the single source of truth for mock data across both the ETL and API test suites.

```sh
cd storm-data-etl
go run ./cmd/genmock \
  -csv-dir ../storm-data-system/mock-server/data \
  -etl-out data/mock/storm_reports_240426_combined.json \
  -api-out ../storm-data-api/data/mock/storm_reports_240426_transformed.json
```

After regenerating:

1. Run `go run ./cmd/validate ...` to verify consistency between all mock data files
2. Run `make test-unit` in both the ETL and API repos
3. Run `make test-e2e` in the system repo

## Add a New CSV Report Type

To add a new NOAA report type (e.g., flash floods):

1. **Collector**: Add the type to `REPORT_TYPES` configuration, add the magnitude column mapping in `src/csv/csvStream.ts`
2. **ETL**: Add the type to event type normalization in `internal/domain/transform.go`, add unit defaults and severity classification rules
3. **API**: Add the GraphQL enum value in `schema.graphqls`, regenerate with `make generate`
4. **Mock data**: Add a CSV fixture to `mock-server/data/`, update `genmock` definitions
5. **E2E tests**: Update expected counts and add type-specific assertions

## Run the Full Pipeline Locally

```sh
cd storm-data-system
make up          # Build all images from source, start stack
make logs        # Tail all service logs (watch data flow)
make test-e2e    # Run E2E tests against the stack
make down        # Tear down when done
```

The collector runs its job once on startup, so data will flow through within seconds of the stack starting.

## Related

- [ETL Enrichment](https://github.com/couchcryptid/storm-data-etl/wiki/Enrichment) -- enrichment rules and severity classification
- [API Data Model](https://github.com/couchcryptid/storm-data-api/wiki/Data-Model) -- database schema and field mapping
- [[Development]] -- multi-repo workflow and CI
- [[Data Model]] -- message shapes and event types
- [[Testing]] -- running E2E and UAT tests
