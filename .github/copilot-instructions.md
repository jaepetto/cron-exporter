# GitHub Copilot Instructions for Cron Metrics Collector & Exporter (Go Edition)

Welcome to the project! This repo is a Go-based API and web server for aggregating cron job results and exposing Prometheus metrics with full lifecycle (CRUD, alerting, maintenance mode, and custom labels).

## Project Status & Key Architecture

This is a **production-ready, fully-implemented system with 100% passing test coverage**. The complete codebase is functional with comprehensive testing (unit, integration, and end-to-end tests). All specifications in `docs/specs.md` have been successfully implemented.

**Core Architecture:**
- Dual-interface system: REST API + Cobra CLI both managing the same SQLite job store
- Prometheus metrics endpoint with automatic failure detection based on per-job thresholds
- Jobs support arbitrary labels (JSON) and maintenance status to suppress alerting
- Expected project layout: `/cmd/cronmetrics` (main), `/pkg/{api,metrics,config,model}`, `/internal/cli`

## Essential Implementation Knowledge

**Job Data Model (Critical):**
```go
type Job struct {
    ID int                         // Auto-incrementing primary key
    Name string                    // Job name (unique with host)
    Host string                    // Host name (unique with name)
    AutomaticFailureThreshold int  // Seconds since last result
    Labels map[string]string       // Arbitrary user labels
    Status string                  // "active", "maintenance", "paused"
    LastReportedAt time.Time       // For auto-failure logic
}
```

**Metrics Logic:** Jobs in `maintenance`/`paused` status must emit value `-1` in Prometheus output to suppress alerting.

**CLI Design Pattern:** Use Cobra subcommands: `cronmetrics serve`, `cronmetrics job {add,list,update,delete}`, `cronmetrics config`

**API Endpoints (Priority Order):**
1. `POST /api/job-result` - Job result submission (highest traffic)
2. `GET /metrics` - Prometheus scraping endpoint
3. CRUD endpoints: `GET /job`, `POST /job`, `PUT /job/{id}`, `DELETE /job/{id}`

## Required Dependencies & Setup

All dependencies are already configured in `go.mod`. Key dependencies:
- **github.com/spf13/cobra** - CLI framework with full job management commands
- **github.com/prometheus/client_golang/prometheus** - Metrics collection with status labels
- **github.com/mattn/go-sqlite3** - Database with automatic migrations
- **github.com/sirupsen/logrus** - Structured logging throughout the system

**mise setup:** `.tool-versions` already configured with Go version and required tasks.

## Critical Implementation Patterns

**Structured Logging Context:**
```go
log.WithFields(logrus.Fields{
    "job_name": jobName,
    "host": host,
    "request_id": ctx.Value("request_id"),
}).Info("job result submitted")
```

**SQLite Schema Approach:** Store labels as JSON TEXT column, use `json.Marshal/Unmarshal` for flexibility. Jobs table: `(name, host)` composite primary key.

**Prometheus Metrics Registration:** Register collectors in `pkg/metrics/collector.go`, implement `Describe()` and `Collect()` methods that query SQLite and apply auto-failure logic.

**Config Loading Pattern:** Viper reads from `/etc/cronmetrics/config.yaml`, env vars prefixed `CRONMETRICS_`, with sane defaults for dev (SQLite in `/tmp/`).

## Development Workflow & Commands

**MANDATORY TESTING REQUIREMENTS:**
- **ALL code changes MUST run full test suite**: `mise run test`
- **ALL changes MUST maintain 100% test coverage** - no exceptions
- **Tests MUST be updated** when adding/modifying functionality
- **Integration and E2E tests** are critical - never skip them
- **NO code changes without passing tests** - this is a hard requirement

**Production System Commands:**
```bash
mise run test             # REQUIRED before any code changes
mise run build            # Static binary build
mise run dev              # Development server (already configured)
mise run integration      # Run integration tests
mise run e2e              # Run end-to-end tests
```

**mise Task Requirements:** All tasks are implemented and validated:
- `test` - Runs comprehensive test suite (unit + integration + e2e)
- `build` - Creates production-ready binary
- `dev` - Starts development server with proper configuration
- `integration` - Targeted integration test execution
- `e2e` - End-to-end workflow validation

**Database Migrations:** Use simple SQL files in `migrations/` directory, applied at startup. Keep schema changes backward-compatible.

## Development & Maintenance Prompt Examples

**ALWAYS START WITH:** "Before making any changes, run `mise run test` to ensure current state is working"

- "Add new API endpoint for X - include request/response validation, tests, and update docs"
- "Enhance the Prometheus metrics collector to support Y - include unit and integration tests"
- "Debug failing test in test/integration/X_test.go - analyze and fix root cause"
- "Add new CLI command for Z - include Cobra subcommand, validation, and comprehensive tests"
- "Optimize database query performance in pkg/model - include benchmarks and tests"
- "Add new job status type X - update model, API, CLI, metrics, and all related tests"
- "Investigate metric collection issue - check collector.go and related test coverage"

**TESTING-FOCUSED PROMPTS:**
- "Run full test suite and fix any failing tests before implementing feature X"
- "Add comprehensive test coverage for new feature Y including unit, integration, and e2e tests"
- "Debug and fix flaky test Z - analyze root cause and implement stable solution"
- "Update existing tests after modifying API endpoint behavior for feature X"

## Documentation Standards

- Update `docs/specs.md` for any API changes
- All exported functions require godoc comments
- Include working curl examples in API documentation
- Maintain `CHANGELOG.md` with semantic versioning

**Critical: Newcomer Onboarding Documentation**
- `README.md` must provide complete setup instructions from clone to running server
- `CONTRIBUTING.md` must include development workflow, testing requirements, and PR guidelines
- `CHANGELOG.md` must document all breaking changes and migration paths
- Documentation updates are REQUIRED with every code change - no exceptions
- All docs must be kept current to ensure any newcomer can contribute immediately

---
