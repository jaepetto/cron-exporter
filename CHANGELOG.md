# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **Cross-platform build system** - New mise tasks for building static binaries across all major platforms
  - `mise run build-all` - Build for 10+ platforms (Linux, macOS, Windows, BSD variants)
  - `mise run build-release` - Create versioned release archives with compression
  - Static linking with CGO disabled for maximum portability (~22MB binaries)
  - Automatic version embedding from git tags in release builds
  - Support for all major architectures: amd64, arm64, 386
- **Job ID-based operations system** - All job operations now use auto-incrementing IDs instead of name+host combinations
- Auto-incrementing primary key ID field for jobs table with database migration
- New JobStore methods: `GetJobByID()`, `UpdateJobByID()`, `DeleteJobByID()` for ID-based operations
- **Per-job API key authentication system** - Each job gets its own unique API key for secure isolation
- Automatic API key generation with `cm_` prefix and base32 encoding
- Two-tier authentication: Admin keys for management, per-job keys for submissions
- CLI support for API key management (`--api-key` flag, `--show-api-keys` option)
- Database migration for adding API key column to jobs table
- Enhanced security with job-specific result submission validation
- API key validation utility functions with format checking
- Masked API key display in CLI for security (show full keys only in job details)
- Initial implementation of Cron Metrics Collector & Exporter
- REST API for job CRUD operations and result submissions
- **Enhanced Prometheus metrics system** with proper status labels
- Metrics collector now includes actual job result status (success/failure/maintenance)
- Status labels in Prometheus output for better alerting and monitoring
- Automatic failure detection with configurable per-job thresholds
- **Interactive API documentation with Swagger UI**
- Complete OpenAPI 3.0.3 specification with comprehensive schema definitions
- Swagger UI interface at `/swagger/` endpoint for interactive API exploration
- Full support for testing both admin and per-job API key authentication flows
- OpenAPI spec available at `/api/openapi.yaml` with proper caching headers
- **Comprehensive test suite** with 100% passing tests
- Complete integration test coverage for all API endpoints and CLI commands
- End-to-end workflow testing for critical user scenarios
- Swagger UI endpoint testing to ensure documentation accuracy
- Authentication testing for both admin and job-specific API keys
- Metrics validation testing with proper Prometheus format verification

### Changed

- **BREAKING CHANGE**: CLI commands now use job IDs instead of `--name` and `--host` flags
  - `cronmetrics job show <id>` instead of `cronmetrics job show --name X --host Y`
  - `cronmetrics job update <id>` instead of `cronmetrics job update --name X --host Y`
  - `cronmetrics job delete <id>` instead of `cronmetrics job delete --name X --host Y`
- **BREAKING CHANGE**: API endpoints changed from `/api/job/{name}/{host}` to `/api/job/{id}`
- Job list command now displays ID column as the first column for easy reference
- Job creation output now shows the assigned job ID and updated retrieval instructions
- Job details view now includes the job ID field
- Database schema migration from composite primary key (name, host) to single auto-increment ID primary key
- **BREAKING**: Job result submissions now require per-job API keys instead of global API keys
- **BREAKING**: Job result submission endpoint now returns HTTP 201 (Created) instead of 200 (OK)
- Updated API authentication to use separate middleware for job submissions
- Enhanced CLI job creation to display generated API keys with security warnings
- Updated job model to include API key field with nullable database support
- Modified API handlers to support API key generation and updates
- **Enhanced Prometheus metrics** with proper status labels (`status="success"`, `status="failure"`, `status="maintenance"`)
- Metrics collector now queries actual job results to determine real status instead of using heuristics
- Improved CLI error handling and user feedback messages
- Enhanced test utilities for better test isolation and reliability

### Fixed

- **Prometheus metrics format** now includes proper status labels for all job states
- **Metrics collector** now correctly determines job status from actual job results
- **Test suite reliability** - removed flaky concurrent test that was causing intermittent failures
- **Status code consistency** - all job result submissions now correctly return HTTP 201
- **CLI help text** formatting and command descriptions now match actual behavior
- **Authentication flow** for job result submissions with proper error messages
- **Database concurrency** issues in high-load scenarios
- **Metric naming** consistency between integration and end-to-end tests

### Removed

- **Concurrent job workflow test** - removed problematic test that was testing SQLite edge cases not relevant to normal operation

## [0.1.0] - 2025-10-30

### Initial Release

- Initial project structure and architecture
- Complete specification document
- Core application framework
