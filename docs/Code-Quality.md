# Code Quality

Quality philosophy, tooling, and enforcement across the storm data pipeline. Each service also has its own Code Quality page with service-specific details:

- [Collector Code Quality](https://github.com/couchcryptid/storm-data-collector/wiki/Code-Quality)
- [ETL Code Quality](https://github.com/couchcryptid/storm-data-etl/wiki/Code-Quality)
- [API Code Quality](https://github.com/couchcryptid/storm-data-api/wiki/Code-Quality)
- [Shared Library Code Quality](https://github.com/couchcryptid/storm-data-shared/wiki/Code-Quality)

## Coding Philosophy

These principles guide decisions across all services. They're ordered by priority -- when principles conflict, the earlier one wins.

### Readability over cleverness

Code is read far more than it is written. Names describe intent. Functions do one thing. If code needs a comment to explain *what* it does, refactor it. Comments explain *why*, not *what*.

### Fail fast, fail loud

Configuration is validated at startup -- invalid values cause an immediate exit with a clear error message. Bad Kafka messages are logged and skipped rather than silently dropped. The race detector runs on every test execution.

### Pure domain logic

Domain packages have no infrastructure imports. Transformation, validation, severity classification, and location parsing are pure functions that take data in and return data out. This makes business logic testable without Kafka, HTTP, databases, or network calls.

### Consumer-defined interfaces

Interfaces are defined where they're used, not where they're implemented. The ETL's `pipeline` package defines `BatchExtractor`, `Transformer`, and `BatchLoader` -- Kafka adapters happen to satisfy them. The shared library defines `ReadinessChecker` -- services implement it without importing the shared module. Go's structural typing makes this natural.

### Constructor injection

All dependencies are passed at construction time. No global state, no service locators. Loggers, metrics, database pools, and Kafka clients are all injected. This makes every component testable in isolation.

### Idempotent by design

Deterministic SHA-256 IDs, `ON CONFLICT DO NOTHING` upserts, stateless transforms. Duplicate messages are safe at every stage. Combined with at-least-once Kafka delivery, this eliminates an entire class of data consistency problems.

### Minimize abstraction

Avoid wrapping things that don't need wrapping. GraphQL resolvers are thin -- they validate input, delegate to the store, and assemble the response. Shared library wrappers are thin -- one function call. Only abstract when there's a concrete benefit: testability, swappability, or deduplication.

### Schema as contract

The GraphQL schema is the API contract. The Kafka message shape is defined by the mock data. The database schema is derived from domain types. Schemas are sources of truth -- code is generated from them, not the other way around.

### Functional style where natural

Transformation functions are pure: same input, same output, no side effects. Data flows through Kafka topics as an immutable pipeline. Enrichment steps are sequential and composable. Mutable state is confined to infrastructure boundaries (database connections, Kafka consumers).

## What the Codebase Reveals

Reading across all five repositories, several patterns emerge that reveal how this system was built and why changes happen the way they do.

### The pipeline is designed to survive partial failure

Every boundary in the system has a fallback: the collector skips 404s and retries 5xx errors, the ETL skips poison pill messages rather than blocking, the API deduplicates via `ON CONFLICT DO NOTHING`. Geocoding is feature-flagged and degrades gracefully. No single component can halt the pipeline. This reflects a priority on availability over strict consistency -- appropriate for a system where stale data is acceptable but missing data is not.

### Tests are the change safety net, not type systems

Go's type system is deliberately not leveraged for domain validation (no sum types, no newtypes for magnitudes). Instead, the test pyramid does the heavy lifting: 14+ linters catch structural issues, unit tests catch logic bugs, integration tests with real Kafka and PostgreSQL catch wiring issues, E2E tests catch cross-service regressions. The race detector runs on every test execution. This means changes can be made confidently because the feedback loop is fast and comprehensive -- not because the compiler prevents bad states.

### Schemas are contracts, code is generated

The GraphQL schema, database migration, and Kafka message shape (defined by mock data) are the sources of truth. Code is generated from or validated against these schemas. This means most changes start with a schema change and ripple outward: update the `.graphqls` file, run `go generate`, update the migration, regenerate mock data. The 8-step "Add a New Field" guide in [[Common Tasks]] codifies this workflow. Changes are motivated by schema evolution, not code refactoring.

### Observability is built in, not bolted on

Every service registers Prometheus metrics at construction time. Health endpoints, readiness probes, and structured logging are standard across all services via the shared library. The Docker Compose stack includes Prometheus and Kafka UI by default. This means operational changes (adding a metric, adjusting a health check) are routine -- the infrastructure for observability already exists and just needs a new counter or histogram.

### Multi-repo forces explicit boundaries

Each service has its own CI, release cycle, and Docker image. Cross-service changes require coordinated PRs. This creates friction by design: it forces changes to be backward-compatible (the API can't assume a new Kafka field exists until the ETL is deployed) and makes the blast radius of any change visible. The system repo's E2E tests are the integration point that validates the full pipeline works as a unit.

### The shared library is a utility belt, not a framework

`storm-data-shared` provides logging, health endpoints, config parsing, and retry logic -- all as standalone functions with primitive arguments. Services wrap these in thin adapters. This means shared library changes are low-risk (no service depends on internal implementation details) and service-specific changes don't pollute the shared code.

## Quality Tooling by Service

| Tool | Collector | ETL | API | Shared |
|------|-----------|-----|-----|--------|
| **Language** | TypeScript | Go | Go | Go |
| **Linter** | ESLint + `@typescript-eslint` | golangci-lint (14 linters) | golangci-lint (15 linters) | golangci-lint (12 linters) |
| **Formatter** | Prettier | gofmt + goimports | gofmt + goimports | gofmt + goimports |
| **Type checking** | `tsc --noEmit` | Go compiler | Go compiler | Go compiler |
| **Test framework** | Vitest | `go test` (-race) | `go test` (-race) | `go test` (-race) |
| **Coverage** | `@vitest/coverage-v8` (lcov) | `go test -coverprofile` | `go test -coverprofile` | `go test -coverprofile` |
| **Pre-commit** | Husky + lint-staged | `.pre-commit-config.yaml` | `.pre-commit-config.yaml` | `.pre-commit-config.yaml` |
| **Secret detection** | -- | gitleaks | gitleaks | gitleaks |
| **Vuln scanning** | -- | -- | -- | govulncheck |
| **SonarCloud** | Yes (CI) | Yes (CI) | Yes (CI) | Yes (CI) |

## Static Analysis

### Go Services (ETL, API, Shared)

All Go projects use `golangci-lint` with a shared configuration baseline:

| Category | Linters | Purpose |
|----------|---------|---------|
| Correctness | `errcheck`, `govet`, `staticcheck`, `errorlint` | Unchecked errors, suspicious constructs, error wrapping |
| Security | `gosec`, `noctx` | Security-sensitive patterns, missing HTTP request contexts |
| Style | `gocritic` (diagnostic/style/performance), `revive` (exported) | Idiomatic Go patterns, naming conventions |
| Complexity | `gocyclo` (threshold: 15) | Overly complex functions |
| Hygiene | `misspell`, `unparam`, `errname`, `unconvert`, `prealloc` | Typos, unused params, naming, unnecessary conversions |
| Exhaustiveness | `exhaustive` | Unhandled enum values in switch statements |
| Test quality | `testifylint` | Testify assertion best practices |

The API adds `sqlclosecheck` for database resource safety and excludes gqlgen-generated files from analysis. The ETL adds `bodyclose` for HTTP response body handling (Mapbox API). The shared library omits `bodyclose`, `noctx`, and `sqlclosecheck` (no HTTP clients or databases).

### Collector (TypeScript)

ESLint flat config with `@typescript-eslint` for type-aware linting and Prettier for formatting. TypeScript strict mode (`--noEmit`) runs as a separate CI step.

## Security

Multiple layers prevent security issues from reaching production:

| Layer | What It Catches | When |
|-------|----------------|------|
| `gosec` / type-aware linting | SQL injection, command injection, weak crypto, hardcoded credentials | Lint time |
| `gitleaks` | Secrets in source code (API keys, tokens, passwords) | Pre-commit + CI |
| `detect-private-key` | Private key files accidentally committed | Pre-commit |
| `check-added-large-files` | Accidentally committed binaries or data files | Pre-commit |
| SonarCloud security hotspots | Framework-specific security patterns | CI |

The API also enforces runtime query protection: a complexity budget (600), depth limit (7), and concurrency limit (2) protect against expensive or abusive GraphQL queries.

## Pre-commit Hooks

All projects enforce checks before code enters the repository. This is the first line of defense -- issues are caught locally before a push.

**Go projects** (`.pre-commit-config.yaml`):

- File hygiene: trailing whitespace, end-of-file newline, YAML/JSON validation, merge conflict markers
- Formatting: `gofmt`, `goimports`
- Linting: `golangci-lint` (5-minute timeout)
- Security: `gitleaks`, `detect-private-key`, `check-added-large-files`
- Documentation: `yamllint`, `markdownlint`

**Collector** (Husky + lint-staged):

- `prettier --write` on staged TypeScript files
- `eslint --fix` on staged TypeScript files
- Full unit test suite (`npm run test:unit`)

## CI Quality Gates

Every push and pull request to `main` must pass these checks before merge. This is the second line of defense -- nothing merges without passing CI.

| Check | Collector | ETL | API | Shared |
|-------|-----------|-----|-----|--------|
| Unit tests | `npm run test:unit` | `make test-unit` | `make test-unit` | `make test` |
| Linting | `npm run lint` + `tsc --noEmit` | `make lint` | `make lint` | `make lint` |
| Build | `npm run build` | `make build` | `make build` | -- |
| SonarCloud scan | Yes | Yes | Yes | Yes |

CI runs on GitHub Actions. A separate `release.yml` workflow handles versioning and Docker image publishing after CI passes on `main`.

## SonarQube Cloud

All five projects run [SonarCloud](https://sonarcloud.io/organizations/couchcryptid/projects) analysis as a CI job on every push and pull request.

| Project | Dashboard | Quality Gate |
|---------|-----------|-------------|
| Collector | [SonarCloud](https://sonarcloud.io/summary/overall?id=couchcryptid_storm-data-collector) | Sonar way |
| ETL | [SonarCloud](https://sonarcloud.io/summary/overall?id=couchcryptid_storm-data-etl) | Sonar way |
| API | [SonarCloud](https://sonarcloud.io/summary/overall?id=couchcryptid_storm-data-api) | Sonar way |
| Shared | [SonarCloud](https://sonarcloud.io/summary/overall?id=couchcryptid_storm-data-shared) | Sonar way |
| System | [SonarCloud](https://sonarcloud.io/summary/overall?id=couchcryptid_storm-data-system) | Sonar way |

### SonarCloud Configuration

Each project has a `sonar-project.properties` file with project-specific settings:

- **Go projects**: Report coverage via `sonar.go.coverage.reportPaths=coverage.out`. Idiomatic Go test naming (`TestX_Y_Z` with underscores) is allowed on test files via a rule suppression for `go:S100`.
- **API**: Excludes gqlgen-generated files (`generated.go`, `models_gen.go`) from analysis since they are not hand-written code.
- **Collector**: Reports TypeScript coverage via `sonar.javascript.lcov.reportPaths=coverage/lcov.info`.
- **System**: Scans both `e2e/` and `mock-server/` Go modules. No coverage reporting (E2E tests require the full Docker Compose stack).

### Quality Gate: Sonar Way

All projects use the default "Sonar way" quality gate, which enforces on **new code** (code changed since the last analysis):

| Condition | Threshold |
|-----------|-----------|
| New code coverage | >= 80% |
| New duplicated lines | <= 3% |
| New reliability rating | A |
| New security rating | A |
| New maintainability rating | A |
| New security hotspots reviewed | 100% |

The quality gate focuses on new code to allow incremental improvement without blocking existing work.

## Testing Strategy

Tests are organized in four tiers. See [[Testing]] for the complete strategy, test fixtures, and CI integration.

| Tier | Scope | Docker Required | Duration |
|------|-------|----------------|----------|
| Unit | Isolated functions with mocked dependencies | No | Seconds |
| Integration | Service + real infrastructure (testcontainers) | Yes | 1-2 minutes |
| E2E | Full pipeline via Docker Compose | Yes | 2-5 minutes |
| UAT | Dashboard UI via Playwright | Yes | 1-2 minutes |

All Go tests run with `-race -count=1` (race detector, no caching). The race detector has caught real concurrency bugs and is non-negotiable.
