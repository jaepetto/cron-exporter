# Project Context

## Purpose
A production-ready Go-based API and web server for centralizing cron job results and exporting them as Prometheus-compatible metrics. The system provides comprehensive job lifecycle management (CRUD operations), sophisticated monitoring with automatic failure detection, maintenance mode for alert suppression, and two-tier authentication for enhanced security isolation.

**Key Goals:**

- Centralize cron job status reporting across multiple hosts and environments
- Provide rich Prometheus metrics with per-job labels and thresholds
- Enable maintenance/operational control without data loss (pause/resume jobs)
- Secure isolation between jobs with per-job API keys
- Simple curl-based integration for cron job scripts
- Administrative CLI for complete job management

## Tech Stack

- **Language**: Go 1.23+ (managed via mise tool)
- **Database**: SQLite3 with automatic schema migrations
- **HTTP Server**: Standard Go net/http with custom multiplexer
- **Metrics**: Prometheus client library for /metrics endpoint
- **CLI Framework**: Cobra for command-line interface
- **Configuration**: Viper for configuration management with YAML/env support
- **Logging**: Logrus for structured logging with configurable levels
- **Testing**: Testify framework with 100% test coverage requirement
- **Documentation**: OpenAPI 3.0.3 specification with Swagger UI
- **Build/Deploy**: Docker containerization, static binary builds, GitHub Actions CI/CD

## Project Conventions

### Code Style

- **Go Standards**: Strict adherence to `gofmt`, `go vet`, and `golangci-lint` rules
- **Package Structure**: Clear separation with `/cmd`, `/pkg`, `/internal`, `/test` layout
- **Naming**: Descriptive names, avoiding abbreviations, following Go naming conventions
- **Error Handling**: Explicit error handling with structured logging context
- **Documentation**: Godoc comments required for all exported functions and types
- **API Design**: RESTful endpoints with consistent JSON responses and proper HTTP status codes

### Architecture Patterns

- **Dual Interface**: REST API + Cobra CLI both managing the same SQLite job store
- **Layered Architecture**:
  - `/cmd` - Entry points and main functions
  - `/pkg` - Public APIs and domain models (api, config, metrics, model)
  - `/internal` - Private CLI implementation and test utilities
- **Dependency Injection**: Services passed as parameters, avoiding global state
- **Database Layer**: Repository pattern with `JobStore` and `JobResultStore`
- **Metrics Collection**: Custom Prometheus collector with dynamic label support
- **Two-Tier Authentication**: Admin API keys for CRUD, per-job keys for result submission

### Testing Strategy

- **100% Test Coverage Requirement**: Absolutely mandatory, no exceptions
- **Test Types**: Unit tests, integration tests, and end-to-end tests
- **Test Structure**: Mirror source structure in `/test` directory
- **Test Categories**:
  - Unit: Individual function/method testing
  - Integration: API endpoints, database operations, CLI commands
  - E2E: Complete workflows from job creation to metrics export
- **Mocking**: Minimal mocking, prefer real database operations with test isolation
- **CI Validation**: All tests must pass before merge, run on multiple Go versions

### Git Workflow

- **Branch Strategy**: Feature branches from main, no direct main commits
- **Commit Messages**: Conventional commits format preferred
- **PR Requirements**:
  - All tests passing (validated by CI)
  - Code review required
  - Documentation updates included
  - Changelog updates for user-facing changes
- **Release Process**: Semantic versioning with automated GitHub releases

## Domain Context

### Cron Job Monitoring Domain

- **Job Identity**: Jobs identified by unique (name, host) combination
- **Execution Results**: Success/failure status with optional duration and output
- **Automatic Failure Detection**: Configurable per-job thresholds for silence-based failures
- **Maintenance Operations**: Jobs can be paused without deletion to suppress alerting
- **Label System**: Arbitrary JSON labels per job for flexible Prometheus queries

### Prometheus Integration

- **Metrics Format**: Standard Prometheus exposition format at `/metrics`
- **Job Status Values**: 1=success, 0=failure, -1=maintenance/paused
- **Dynamic Labels**: Job-specific labels automatically added to metrics
- **Alert Suppression**: Maintenance mode jobs excluded from alerting logic

### Security Model

- **Admin API Keys**: Full CRUD access to job definitions, manually configured
- **Per-Job API Keys**: Automatically generated, job-specific, write-only access
- **Isolation**: Jobs can only submit results for themselves
- **Development Mode**: Authentication bypass for local development

## Important Constraints

### Technical Constraints

- **Database**: SQLite only (no external database dependencies)
- **Go Version**: Must support Go 1.23+ for language features and performance
- **Static Binary**: Must compile to single static binary for easy deployment
- **Memory Usage**: Efficient memory usage for embedded/small server deployments
- **Thread Safety**: All operations must be thread-safe for concurrent API access

### Operational Constraints

- **Zero Downtime**: Schema migrations must not require downtime
- **Backward Compatibility**: API changes must maintain backward compatibility
- **Configuration**: All configuration via files, environment variables, or CLI flags
- **Logging**: Structured logging required for production operations
- **Metrics Performance**: /metrics endpoint must respond quickly under load

### Business Constraints

- **Data Retention**: Configurable retention periods for historical data
- **Security Isolation**: Jobs must not access other jobs' data
- **Audit Trail**: All job management operations should be logged
- **Maintenance Windows**: Support for planned maintenance without false alerts

## External Dependencies

### Core Runtime Dependencies

- **github.com/mattn/go-sqlite3** v1.14.32 - SQLite3 database driver with CGO requirements
- **github.com/prometheus/client_golang** v1.23.2 - Prometheus metrics collection and exposition
- **github.com/sirupsen/logrus** v1.9.3 - Structured logging with configurable levels and formats
- **github.com/spf13/cobra** v1.10.1 - CLI framework for command-line interface
- **github.com/spf13/viper** v1.21.0 - Configuration management (YAML, env vars, flags)

### Development Dependencies

- **github.com/stretchr/testify** v1.11.1 - Testing framework with assertions and mocking
- **github.com/swaggo/http-swagger/v2** v2.0.2 - Swagger UI integration for API documentation
- **golang.org/x/tools** - Static analysis and code generation tools
- **golangci-lint** - Comprehensive linting and static analysis
- **gosec** - Security vulnerability scanner

### Build and Deployment Dependencies

- **Docker** - Container runtime for deployment and CI/CD
- **GitHub Actions** - CI/CD pipeline automation
- **mise** - Development tool version management (replaces asdf)
- **Go 1.23+** - Runtime and build toolchain

### System Dependencies

- **SQLite3** - Database engine (linked statically in build)
- **CGO** - Required for SQLite3 driver compilation
- **Git** - Version control and dependency management

### Optional Dependencies

- **Prometheus** - Metrics collection and alerting (external system)
- **Grafana** - Metrics visualization and dashboards (external system)
- **Reverse Proxy** - TLS termination and load balancing (nginx, traefik, etc.)
[Document key external services, APIs, or systems]
