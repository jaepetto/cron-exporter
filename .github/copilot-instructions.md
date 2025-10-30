# GitHub Copilot Instructions for Cron Metrics Collector & Exporter (Go Edition)

Welcome to the project! This repo is a Go-based API and web server for aggregating cron job results and exposing Prometheus metrics with full lifecycle (CRUD, alerting, maintenance mode, and custom labels).

## Project Status & Key Architecture

This is a **greenfield project in implementation phase**. Complete specifications exist in `docs/specs.md` but the Go codebase needs to be built from scratch.

**Core Architecture:**
- Dual-interface system: REST API + Cobra CLI both managing the same SQLite job store
- Prometheus metrics endpoint with automatic failure detection based on per-job thresholds
- Jobs support arbitrary labels (JSON) and maintenance status to suppress alerting
- Expected project layout: `/cmd/cronmetrics` (main), `/pkg/{api,metrics,config,model}`, `/internal/cli`

## Essential Implementation Knowledge

**Job Data Model (Critical):**
```go
type Job struct {
    Name string                    // Primary key with host
    Host string                    // Primary key with name
    AutomaticFailureThreshold int  // Seconds since last result
    Labels map[string]string       // Arbitrary user labels
    Status string                  // "active", "maintenance", "paused"
    LastReportedAt time.Time       // For auto-failure logic
}
```

**Metrics Logic:** Jobs in `maintenance`/`paused` status must emit value `-1` or `status="maintenance"` label in Prometheus output to suppress alerting.

**CLI Design Pattern:** Use Cobra subcommands: `cronmetrics serve`, `cronmetrics job {add,list,update,delete}`, `cronmetrics config`

**API Endpoints (Priority Order):**
1. `POST /api/job-result` - Job result submission (highest traffic)
2. `GET /metrics` - Prometheus scraping endpoint
3. CRUD endpoints: `GET /job`, `POST /job`, `PUT /job/{id}`, `DELETE /job/{id}`

## Required Dependencies & Setup

```bash
# Initialize with these exact dependencies
go mod init github.com/your-org/cron-exporter
go get github.com/spf13/cobra
go get github.com/spf13/viper
go get github.com/prometheus/client_golang/prometheus
go get github.com/mattn/go-sqlite3
go get github.com/sirupsen/logrus  # or go.uber.org/zap
```

**mise setup:** Create `.tool-versions` with `golang 1.21.0` (or latest stable)

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

**Project Initialization:**
```bash
# Start here - create the basic project structure
cronmetrics serve --dev    # Development mode with in-memory SQLite
mise run test             # Run all tests
mise run build            # Static binary build
```

**mise Task Requirements:** All console commands must have corresponding `mise` tasks defined in `.mise.toml` or `mise.toml` for easy onboarding:
```toml
[tasks.test]
run = "go test ./..."
description = "Run all tests"

[tasks.build]
run = "go build -o bin/cronmetrics ./cmd/cronmetrics"
description = "Build static binary"

[tasks.dev]
run = "./bin/cronmetrics serve --dev"
description = "Start development server"
```

**Database Migrations:** Use simple SQL files in `migrations/` directory, applied at startup. Keep schema changes backward-compatible.

## Copilot/Chat Prompt Examples

- "Create the main.go with Cobra root command and 'serve' subcommand that starts HTTP server"
- "Implement the Job model struct with SQLite CRUD operations and JSON label marshaling"
- "Write the /api/job-result POST handler with request validation and database persistence"
- "Build the Prometheus metrics collector that queries jobs and applies auto-failure threshold logic"
- "Add Cobra CLI commands for job management: add, list, update, delete with proper flag parsing"
- "Implement Viper configuration loading with YAML file and environment variable overrides"
- "Create SQLite database initialization and migration system in pkg/model"

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
