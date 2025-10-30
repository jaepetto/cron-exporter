
# Cron Metrics Collector & Exporter (Go Edition)

## Overview

A Go-based API and web server to centralize cron job results and export their statuses as Prometheus-compatible metrics. Focused on simple curl integration, fast and robust deployment, complete lifecycle management (CRUD API), and sophisticated monitoring—including maintenance/alert suppression and structured, maintainable code and docs.

## Key Features

- Central REST API for job result submissions (/api/job-result/) and full CRUD management of jobs
- Prometheus /metrics endpoint displaying per-job status, totals, and label-rich job metrics
- Per-job automatic failure threshold (auto-marks jobs as failed if silence exceeds threshold)
- Arbitrary user-defined labels per job for flexible Prometheus queries and UI filtering
- Maintenance mode: jobs can be paused to suppress alerting/downtime without removal
- Admin CLI for all job management operations
- Strictly versioned configuration and code, reproducible builds, and up-to-date docs

## Functional Requirements

- Job Management:
  - CRUD operations for job definitions via API and CLI:
    - Create: Register new job (name, host, threshold, labels, status)
    - Read: List/filter all jobs
    - Update: Modify thresholds, labels, maintenance flag
    - Delete: Remove job definition
  - Job status/lifecycle flags (active, maintenance, paused, retired), adjustable at runtime via API/CLI
  - Maintenance/paused jobs are excluded from alerting in /metrics, and clearly flagged in all outputs
- Metrics and Monitoring:
  - /metrics endpoint outputs Prometheus-formatted metrics:
    - Per-job status (cronjob_status), last run, last error, auto-failure reason/expiry, all labels
    - Jobs in maintenance/status have value -1 or a dedicated status="maintenance" label
  - Automatic failure if no result within automatic_failure_threshold (per job)
  - Metrics include all user-supplied labels
- Security:
  - All admin/CRUD endpoints authenticated by API keys or tokens
  - All submission endpoints authenticated (per job or global tokens)
  - All endpoints over HTTPS
- Data Retention:
  - Configurable retention above default (e.g., 30 or 90 days)
  - Option to purge logs, redact sensitive “output” fields
- Admin Tooling & Dashboard:
  - Cobra CLI commands for serve, job CRUD, config management
  - Simple API-driven UI/dashboard for browsing/filtering jobs and statuses (optional in MVP)

## Non-Functional Requirements

- Containerized build (Docker), static Linux binary, CI/CD pipeline with mise
- <300 ms /metrics for up to 10,000 jobs
- Pure Go codebase, Viper config (YAML + env overrides), structured logging (Zap or Logrus)
- Documentation always up-to-date as source of project truth

## API/Job Structure

### Job Definition Example

```json
{
  "job_name": "db_import",
  "host": "backup3",
  "automatic_failure_threshold": 3600,
  "labels": {
    "env": "stage",
    "team": "migration"
  },
  "status": "maintenance"
}
```

JobResult Submission Example

```json
{
  "job_name": "sync_db",
  "host": "web1",
  "status": "success",
  "labels": {
    "env": "prod",
    "team": "infra",
    "type": "backup"
  },
  "duration": 27,
  "timestamp": "2025-10-30T19:56:00Z"
}
```

### Prometheus Metrics Example

```text
# Active and passing
cronjob_status{job_name="sync_db",host="web1",env="prod",team="infra"} 1

# Missed threshold (auto-failed)
cronjob_status{job_name="cleanup",host="web2",env="prod"} 0
cronjob_status_reason{job_name="cleanup",host="web2"} "missed deadline"

# Maintenance mode, no alerting
cronjob_status{job_name="db_import",host="backup3",env="stage",status="maintenance"} -1
cronjob_status_reason{job_name="db_import",host="backup3",env="stage"} "maintenance"
```

### OpenAPI (Swagger) API

See `docs/openapi.yaml` for full schema and example requests/responses.

- `/job` [POST, GET] — create/list jobs
- `/job/{id}` [GET, PUT, DELETE] — read/update/delete single job
- `/job-result` [POST] — submit job result
- `/metrics` [GET] — Prometheus metrics

### Tooling & Codebase

- Cobra: CLI structure (cmd/cronmetrics for serve, job, config, etc.)
- Viper: Unified config loader (YAML + env), reloadable at runtime
- Logrus/Zap: Structured logging
- mise: Build/test/env standardization, managed with .tool-versions
- Project Layout:

```text
/cmd/cronmetrics
/pkg/api
/pkg/metrics
/pkg/config
/pkg/model
/pkg/log
/internal/cli
/docs/
.tool-versions
.env
```

### CLI Example

```bash
cronmetrics serve --config /etc/cronmetrics/config.yaml
cronmetrics job add --name backup --host db1 --threshold 600 --label env=prod
cronmetrics job update --name backup --maintenance true
cronmetrics job list --label env=prod
```

### Logging Conventions

- Structured logging (Zap or Logrus), key-value pairs
- Context: Always log job, host, user, request_id, and error/return where relevant
- Log Levels:
  - DEBUG: Function entry/exit with params/returns (when enabled)
  - INFO: System lifecycle, submission/crud events
  - WARNING: Recoverable issues (invalid config, submission rejections)
  - ERROR: Critical errors, panics, data loss risks
- Log in UTC, RFC3339 timestamps, always thread safe

### Documentation & Maintainability

- `/docs/` directory contains API spec (OpenAPI), admin/usage/config guides, changelog
- All PRs and code changes must update docs
- Linting and doc-completeness checks in CI
- README, CONTRIBUTING, and onboarding step-by-step for devs

### Acceptance Criteria

- CRUD lifecycle for jobs (API + CLI), maintenance flag functioning as required
- Prometheus metrics correctly represent all job/label/status cases
- Logging, configuration, and documentation all adhere to established conventions
- MVP release includes end-to-end example, CI pipeline, and up-to-date docs

### Changelog

- v0.1, 2025-10-30: Initial specification and system design
