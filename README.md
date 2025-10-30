# Cron Metrics Collector & Exporter

A Go-based API and web server to centralize cron job results and export their statuses as Prometheus-compatible metrics.

## Features

- **Central REST API** for job result submissions and full CRUD management of jobs
- **Prometheus /metrics endpoint** displaying per-job status, totals, and label-rich job metrics
- **Per-job automatic failure threshold** (auto-marks jobs as failed if silence exceeds threshold)
- **Arbitrary user-defined labels** per job for flexible Prometheus queries and UI filtering
- **Maintenance mode** - jobs can be paused to suppress alerting/downtime without removal
- **Admin CLI** for all job management operations
- **SQLite backend** with automatic migrations
- **Structured logging** with configurable levels and formats

## Quick Start

### Prerequisites

- Go 1.21+ (managed via mise)
- SQLite3

### Installation

1. Clone the repository:
```bash
git clone https://github.com/jaep/cron-exporter
cd cron-exporter
```

2. Install dependencies:
```bash
mise install  # Installs Go version from .tool-versions
go mod tidy
```

3. Build the application:
```bash
mise run build
```

### Development Mode

Start the server in development mode (no authentication, debug logging):

```bash
mise run dev
# or
./bin/cronmetrics serve --dev
```

The server will start on `http://localhost:8080` with:
- API endpoints at `/api/*`
- Prometheus metrics at `/metrics`
- Health check at `/health`

### Production Deployment

1. Generate a configuration file:
```bash
./bin/cronmetrics config example > /etc/cronmetrics/config.yaml
```

2. Edit the configuration file to set:
   - API keys for authentication
   - Database path
   - TLS certificates (if using HTTPS)
   - Logging preferences

3. Start the server:
```bash
./bin/cronmetrics serve --config /etc/cronmetrics/config.yaml
```

## Usage

### Managing Jobs

#### Add a new job
```bash
./bin/cronmetrics job add \
  --name backup \
  --host db1 \
  --threshold 3600 \
  --label env=prod \
  --label team=infra \
  --status active
```

#### List jobs
```bash
# List all jobs
./bin/cronmetrics job list

# Filter by labels
./bin/cronmetrics job list --label env=prod

# JSON output
./bin/cronmetrics job list --json
```

#### Update a job
```bash
# Set maintenance mode
./bin/cronmetrics job update --name backup --host db1 --maintenance

# Update threshold
./bin/cronmetrics job update --name backup --host db1 --threshold 7200

# Update labels
./bin/cronmetrics job update --name backup --host db1 --label env=staging --label team=devops
```

#### Delete a job
```bash
./bin/cronmetrics job delete --name backup --host db1
```

### Submitting Job Results

From your cron jobs, submit results via HTTP POST:

```bash
curl -X POST http://localhost:8080/api/job-result \
  -H "Content-Type: application/json" \
  -d '{
    "job_name": "backup",
    "host": "db1",
    "status": "success",
    "duration": 120,
    "labels": {
      "env": "prod",
      "team": "infra"
    },
    "timestamp": "2025-10-30T19:56:00Z"
  }'
```

### Prometheus Metrics

The `/metrics` endpoint provides:

```prometheus
# Job status: 1=success, 0=failure, -1=maintenance/paused
cronjob_status{job_name="backup",host="db1",env="prod",team="infra"} 1

# Jobs in maintenance mode
cronjob_status{job_name="maintenance_job",host="web1",status="maintenance"} -1

# Auto-failed jobs (exceeded threshold)
cronjob_status{job_name="old_job",host="web2"} 0
cronjob_status_reason{job_name="old_job",host="web2",reason="missed_deadline"} 1

# Last execution timestamp
cronjob_last_run_timestamp{job_name="backup",host="db1"} 1698696960

# Total registered jobs
cronjob_total 5
```

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/job-result` | Submit job execution results |
| GET | `/api/job` | List all jobs (with optional label filters) |
| POST | `/api/job` | Create a new job |
| GET | `/api/job/{name}/{host}` | Get specific job details |
| PUT | `/api/job/{name}/{host}` | Update job configuration |
| DELETE | `/api/job/{name}/{host}` | Delete a job |
| GET | `/metrics` | Prometheus metrics |
| GET | `/health` | Health check |

## Configuration

Environment variables (prefixed with `CRONMETRICS_`):

```bash
CRONMETRICS_SERVER_PORT=8080
CRONMETRICS_DATABASE_PATH=/var/lib/cronmetrics/cronmetrics.db
CRONMETRICS_LOGGING_LEVEL=info
CRONMETRICS_SECURITY_API_KEYS=your-api-key-here
CRONMETRICS_SECURITY_ADMIN_API_KEYS=your-admin-key-here
```

Or use a YAML configuration file (see `cronmetrics config example`).

## Architecture

```
┌─────────────────┐    ┌──────────────┐    ┌─────────────┐
│   Cron Jobs     │───▶│  HTTP API    │───▶│  SQLite DB  │
│                 │    │              │    │             │
└─────────────────┘    └──────────────┘    └─────────────┘
                              │
                              ▼
                       ┌──────────────┐
                       │ Prometheus   │
                       │   Metrics    │
                       └──────────────┘
```

- **SQLite Database**: Stores job definitions and execution results
- **REST API**: Handles job CRUD operations and result submissions
- **Metrics Collector**: Generates Prometheus metrics with auto-failure detection
- **CLI Interface**: Provides administrative commands for job management

## Development

### Running Tests
```bash
mise run test
```

### Building
```bash
mise run build
```

### Linting
```bash
mise run lint
```

## Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/new-feature`
3. Make your changes and add tests
4. Ensure all tests pass: `mise run test`
5. Submit a pull request

## License

MIT License - see LICENSE file for details
