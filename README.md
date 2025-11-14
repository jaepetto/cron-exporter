# Cron Metrics Collector & Exporter
#
## Database Access: sqlx Migration

All database access now uses [github.com/jmoiron/sqlx](https://github.com/jmoiron/sqlx) for safer, more maintainable queries and parameter binding. Direct use of `database/sql` is deprecated in this codebase.

### Why sqlx?
- Prevents SQL injection by enforcing parameterized queries
- Simplifies scanning/querying into structs
- Enables more readable and maintainable code

### Example Usage
```go
import (
  "github.com/jmoiron/sqlx"
  _ "github.com/mattn/go-sqlite3"
)

db, err := sqlx.Open("sqlite3", "file.db")
// Use db.Queryx, db.Get, db.Select, etc.
```

See `pkg/model/database.go` for implementation patterns.

[![CI/CD Pipeline](https://github.com/jaepetto/cron-exporter/actions/workflows/ci.yml/badge.svg)](https://github.com/jaepetto/cron-exporter/actions/workflows/ci.yml)
[![Release](https://github.com/jaepetto/cron-exporter/actions/workflows/release.yml/badge.svg)](https://github.com/jaepetto/cron-exporter/actions/workflows/release.yml)
[![Docker Build](https://github.com/jaepetto/cron-exporter/actions/workflows/docker.yml/badge.svg)](https://github.com/jaepetto/cron-exporter/actions/workflows/docker.yml)
[![codecov](https://codecov.io/gh/jaepetto/cron-exporter/branch/main/graph/badge.svg)](https://codecov.io/gh/jaepetto/cron-exporter)

A Go-based API and web server to centralize cron job results and export their statuses as Prometheus-compatible metrics.

## Features

- **Central REST API** for job result submissions and full CRUD management of jobs
- **Per-job API key authentication** - each job has its own unique API key for secure isolation
- **Prometheus /metrics endpoint** displaying per-job status, totals, and label-rich job metrics
- **Per-job automatic failure threshold** (auto-marks jobs as failed if silence exceeds threshold)
- **Arbitrary user-defined labels** per job for flexible Prometheus queries and UI filtering
- **Maintenance mode** - jobs can be paused to suppress alerting/downtime without removal
- **Web Dashboard** (optional) - real-time job monitoring with visual deadline status indicators
- **Admin CLI** for all job management operations with automatic API key generation
- **SQLite backend** with automatic migrations
- **Structured logging** with configurable levels and formats

## Quick Start

### Prerequisites

- Go 1.21+ (managed via mise)
- SQLite3

### Installation

1. Clone the repository:
```bash
git clone https://github.com/jaepetto/cron-exporter
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

### Docker Deployment

The application is available as a multi-architecture container image at `ghcr.io/jaepetto/cron-exporter`.

#### Quick Start with Docker

```bash
# Run standalone container
docker run -d \
  --name cronmetrics \
  -p 8080:8080 \
  -p 9090:9090 \
  -v cronmetrics_data:/data \
  ghcr.io/jaepetto/cron-exporter:main
```

#### Production Stack with Docker Compose

```bash
# Clone the repository for docker-compose.yml
git clone https://github.com/jaepetto/cron-exporter
cd cron-exporter

# Start the full monitoring stack
docker-compose up -d

# Access services:
# - cronmetrics API: http://localhost:8080
# - Prometheus: http://localhost:9091
# - Grafana: http://localhost:3000 (admin/admin)
```

The Docker Compose setup includes:
- **cronmetrics**: Main application with persistent data storage
- **Prometheus**: Metrics collection and alerting
- **Grafana**: Dashboards and visualization (optional)

#### Container Configuration

Set environment variables for container configuration:

```bash
docker run -d \
  -e CRONMETRICS_LOG_LEVEL=info \
  -e CRONMETRICS_DATABASE_PATH=/data/cronmetrics.db \
  -e CRONMETRICS_SERVER_HOST=0.0.0.0 \
  -e CRONMETRICS_SERVER_PORT=8080 \
  -e CRONMETRICS_METRICS_PORT=9090 \
  ghcr.io/jaepetto/cron-exporter:main
```

#### Available Tags

- `main`: Latest development build from main branch
- `v1.x.x`: Specific release versions
- `sha-<commit>`: Specific commit builds

## Web Dashboard (Optional)

The application includes a web dashboard for visual job monitoring and management.

### Enable Dashboard

Add to your configuration file:

```yaml
dashboard:
  enabled: true
  path: "/dashboard"          # Dashboard URL path
  auth_required: true         # Require admin API key
  title: "Cron Metrics"      # Dashboard title
```

### Access Dashboard

Visit `http://localhost:8080/dashboard` to access:

- **Job overview** with real-time status monitoring
- **Visual deadline indicators** showing job health at a glance:
  - 🟢 **Green**: Job reported within deadline (on time)
  - 🟡 **Yellow**: Job approaching deadline (80% of threshold)
  - 🔴 **Red**: Job missed deadline (past AutomaticFailureThreshold)
  - ⚫ **Gray**: Job in maintenance or paused status
- **Search and filtering** by job name, host, or labels
- **Job management** - create, edit, toggle maintenance mode
- **Real-time updates** via Server-Sent Events or polling fallback

#### Dashboard Authentication

The dashboard uses HTTP Basic Authentication when `auth_required: true` is set:

- **Username**: `admin` (or any value)
- **Password**: Your admin API key (e.g., `test-admin-key-12345`)

Example browser access: When prompted, enter:

- Username: `admin`
- Password: `test-admin-key-12345`

Or using curl:

```bash
curl -u admin:test-admin-key-12345 http://localhost:8080/dashboard/
```

### Dashboard Features

- **Responsive design** that works on desktop and mobile
- **Real-time job status** updates without page refresh
- **Visual deadline tracking** based on per-job thresholds
- **Label-based filtering** and search capabilities
- **Maintenance mode controls** for suppressing alerts
- **Pagination** for large job lists
- **Authentication** with admin API keys

## Usage

### Managing Jobs

#### Add a new job
```bash
# Create job with auto-generated API key
./bin/cronmetrics job add \
  --name backup \
  --host db1 \
  --threshold 3600 \
  --label env=prod \
  --label team=infra \
  --status active

# Output:
# Job ID 1 ('backup@db1') created successfully
# API Key: cm_abcd1234567890abcdef1234567890abcdef1234567890abcd
#
# NOTE: Save this API key for your cron jobs to submit results.
# You can retrieve it later using: cronmetrics job show 1

# Or specify a custom API key
./bin/cronmetrics job add \
  --name backup \
  --host db1 \
  --threshold 3600 \
  --api-key cm_custom-secure-key-for-backup-job \
  --label env=prod \
  --label team=infra \
  --status active
```

#### List jobs
```bash
# List all jobs
./bin/cronmetrics job list

# Show API keys (masked for security)
./bin/cronmetrics job list --show-api-keys

# Filter by labels
./bin/cronmetrics job list --label env=prod

# JSON output
./bin/cronmetrics job list --json

# Show detailed job information (includes full API key)
./bin/cronmetrics job show 1
```

#### Update a job
```bash
# Set maintenance mode
./bin/cronmetrics job update 1 --maintenance

# Update threshold
./bin/cronmetrics job update 1 --threshold 7200

# Update labels
./bin/cronmetrics job update 1 --label env=staging --label team=devops

# Update name and host
./bin/cronmetrics job update 1 --name backup-v2 --host db2
```

#### Delete a job
```bash
./bin/cronmetrics job delete 1
```

### Submitting Job Results

From your cron jobs, submit results via HTTP POST using the job's unique API key:

```bash
curl -X POST http://localhost:8080/api/job-result \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-job-specific-api-key" \
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

| Method | Endpoint | Description | Authentication |
|--------|----------|-------------|----------------|
| POST | `/api/job-result` | Submit job execution results | Per-job API key |
| GET | `/api/job` | List all jobs (with optional label filters) | Admin API key |
| POST | `/api/job` | Create a new job | Admin API key |
| GET | `/api/job/{id}` | Get specific job details | Admin API key |
| PUT | `/api/job/{id}` | Update job configuration | Admin API key |
| DELETE | `/api/job/{id}` | Delete a job | Admin API key |
| GET | `/metrics` | Prometheus metrics | None |
| GET | `/health` | Health check | None |
| GET | `/swagger/` | Interactive Swagger UI documentation | None |
| GET | `/api/openapi.yaml` | OpenAPI 3.0.3 specification | None |

### API Documentation

The complete API documentation is available through the interactive Swagger UI:

- **Swagger UI**: Visit `http://localhost:8080/swagger/` when the server is running
- **OpenAPI 3.0.3 Spec**: Available at `http://localhost:8080/api/openapi.yaml`

The Swagger UI provides:
- Interactive API exploration and testing
- Complete request/response schemas
- Authentication examples for both admin and per-job API keys
- Comprehensive endpoint documentation with examples

#### Quick API Testing with Swagger UI

1. Start the server: `./bin/cronmetrics serve --dev`
2. Open your browser to: `http://localhost:8080/swagger/`
3. Explore the API endpoints and try them out interactively
4. Use the "Authorize" button to test with API keys

## Configuration

Environment variables (prefixed with `CRONMETRICS_`):

```bash
CRONMETRICS_SERVER_PORT=8080
CRONMETRICS_DATABASE_PATH=/var/lib/cronmetrics/cronmetrics.db
CRONMETRICS_LOGGING_LEVEL=info
CRONMETRICS_SECURITY_ADMIN_API_KEYS=your-admin-key-here
```

Or use a YAML configuration file (see `cronmetrics config example`).

## Authentication & Security

The system uses a two-tier authentication model for enhanced security:

### Admin API Keys
- **Purpose**: Administrative operations (job management, configuration)
- **Configuration**: Set via `CRONMETRICS_SECURITY_ADMIN_API_KEYS` environment variable
- **Usage**: Required for creating, updating, and deleting jobs
- **Access**: Full CRUD access to all job management endpoints

### Per-Job API Keys
- **Purpose**: Job result submissions (isolated per job)
- **Generation**: Automatically generated when creating jobs (or specify custom key)
- **Usage**: Each job uses its own unique API key to submit results
- **Security**: Jobs can only submit results for themselves, preventing cross-job interference
- **Format**: `cm_` prefix followed by base32-encoded random data (e.g., `cm_abc123...`)

### Authentication Headers
- **Admin operations**: Use `Authorization: Bearer <admin-api-key>` header
- **Job result submissions**: Use `X-API-Key: <job-specific-api-key>` header

### Security Benefits
- **Isolation**: Compromising one job's API key doesn't affect other jobs
- **Least Privilege**: Jobs only have permission to update their own status
- **Easy Rotation**: Each job can have its API key rotated independently
- **Audit Trail**: Clear separation between administrative and operational actions

### Example Workflow

```bash
# 1. Admin creates a new job
export ADMIN_KEY="admin-key-12345"
curl -X POST http://localhost:8080/api/job \
  -H "Authorization: Bearer $ADMIN_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "job_name": "daily-backup",
    "host": "db-server",
    "automatic_failure_threshold": 3600,
    "labels": {"env": "prod", "team": "platform"}
  }'

# Response includes the generated API key:
# {"job_name": "daily-backup", "host": "db-server", "api_key": "cm_abc123...", ...}

# 2. Use the job's API key in your cron script
export JOB_KEY="cm_abc123456789abcdef123456789abcdef123456789abcdef12"
curl -X POST http://localhost:8080/api/job-result \
  -H "X-API-Key: $JOB_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "job_name": "daily-backup",
    "host": "db-server",
    "status": "success",
    "duration": 120
  }'
```

## Architecture

```
┌─────────────────┐    ┌──────────────┐    ┌─────────────┐
│   Cron Jobs     │───▶│  HTTP API    │───▶│  SQLite DB  │
│ (Per-Job Keys)  │    │ (Auth Layer) │    │ (Jobs+Keys) │
└─────────────────┘    └──────────────┘    └─────────────┘
                              │
                              ▼
┌─────────────────┐    ┌──────────────┐
│  Admin CLI      │───▶│ Prometheus   │
│ (Admin Keys)    │    │   Metrics    │
└─────────────────┘    └──────────────┘
```

### Components

- **SQLite Database**: Stores job definitions with per-job API keys and execution results
- **Authentication Layer**: Validates admin keys for management, per-job keys for submissions
- **REST API**: Handles job CRUD operations (admin) and result submissions (per-job)
- **Metrics Collector**: Generates Prometheus metrics with automatic failure detection
- **CLI Interface**: Provides administrative commands with automatic API key generation

### Security Flow

1. **Admin Operations**: CLI/API uses admin keys for job management
2. **Job Creation**: System generates unique API key for each new job
3. **Result Submission**: Each cron job uses its own API key for status updates
4. **Isolation**: Jobs cannot access or modify other jobs' data

## Development

### Testing

This project includes a comprehensive test suite with 100% passing tests:

#### Test Types

- **Unit Tests**: Test individual components in isolation (`pkg/util`)
- **Integration Tests**: Test component interactions with real dependencies (database, HTTP server)
- **End-to-End Tests**: Test complete user workflows from start to finish

#### Running Tests

```bash
# Run all tests
mise run test

# Run only integration tests
mise run test-integration

# Run specific test package
go test ./test/integration/... -v

# Run with coverage
go test ./... -cover
```

#### Test Coverage

- ✅ **100%** of tests passing
- ✅ Complete API endpoint coverage
- ✅ Full CLI command coverage
- ✅ Authentication system validation
- ✅ Prometheus metrics format verification
- ✅ End-to-end workflow scenarios

### Building

#### Single Platform Build

```bash
# Build binary for current platform
mise run build

# Build for development
go build -o bin/cronmetrics ./cmd/cronmetrics
```

#### Cross-Platform Build

Build static, portable binaries for all supported platforms:

```bash
# Build for all platforms (Linux, macOS, Windows, BSD variants)
mise run build-all

# Build release archives with version information
mise run build-release
```

**Supported Platforms:**

- Linux: amd64, arm64, 386
- macOS: amd64 (Intel), arm64 (Apple Silicon)
- Windows: amd64, 386
- FreeBSD, OpenBSD, NetBSD: amd64

## CI/CD and Automation

This project includes comprehensive automation via GitHub Actions:

### Automated Testing
- **Every Push/PR**: Full test suite, security scanning, and multi-platform builds
- **Coverage Reporting**: Automatic code coverage tracking via Codecov
- **Security Scanning**: Gosec and govulncheck for vulnerability detection

### Automated Releases
- **Tag-triggered**: Create releases by pushing version tags (e.g., `v1.0.0`)
- **Multi-platform Binaries**: Automatic building and publishing of release archives
- **Container Images**: Multi-architecture Docker images published to GitHub Container Registry
- **Changelog Generation**: Automatic release notes from git commits

### Container Registry
- **Development**: `ghcr.io/jaepetto/cron-exporter:main`
- **Releases**: `ghcr.io/jaepetto/cron-exporter:v1.x.x`
- **Architectures**: linux/amd64, linux/arm64

### Dependency Management
- **Automated Updates**: Dependabot creates weekly PRs for Go modules, Actions, and Docker images
- **Security Monitoring**: Automated vulnerability scanning and alerts
- Windows: amd64, 386
- FreeBSD, OpenBSD, NetBSD: amd64

**Features:**

- ✅ **Static binaries** - No external dependencies (CGO disabled)
- ✅ **Stripped binaries** - Optimized size (~22MB per binary)
- ✅ **Cross-compilation** - Build all platforms from any host
- ✅ **Release archives** - Compressed `.tar.gz` (Unix) and `.zip` (Windows) packages
- ✅ **Version embedding** - Automatic version detection from git tags

All binaries are created in the `dist/` directory with clear naming: `cronmetrics-{os}-{arch}[.exe]`

### Development Commands

```bash
# Start development server (no auth, debug logging)
mise run dev

# Run linting
mise run lint

# Clean build artifacts
mise run clean

# View available tasks
mise tasks
```

### Available Mise Tasks

| Task | Description |
|------|-------------|
| `build` | Build binary for current platform |
| `build-all` | Build static binaries for all platforms |
| `build-release` | Build release archives with version info |
| `dev` | Start development server |
| `test` | Run unit tests |
| `test-all` | Run all tests (unit + integration + e2e) |
| `test-integration` | Run integration tests |
| `test-e2e` | Run end-to-end tests |
| `lint` | Run linter and formatter |
| `security` | Run gosec security scanner |
| `security-install` | Install gosec security scanner |
| `clean` | Clean build artifacts and coverage reports |
| `ci` | Run full CI pipeline (includes security scan) |
| `ci-full` | Run CI pipeline with multi-platform builds |

## Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/new-feature`
3. Make your changes and add tests
4. Ensure all tests and security scans pass: `mise run ci`
5. Submit a pull request

See [CONTRIBUTING.md](CONTRIBUTING.md) for detailed development guidelines, including security requirements and coding standards.

## License

MIT License - see LICENSE file for details
