# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
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
- Prometheus metrics endpoint with auto-failure detection
- SQLite database backend with automatic migrations
- Cobra CLI for job management and server operations
- Maintenance mode support to suppress alerting
- Configurable authentication with API keys
- Structured logging with logrus
- Development mode for easy testing
- Docker-ready configuration structure
- Comprehensive documentation and examples

### Changed
- **BREAKING**: Job result submissions now require per-job API keys instead of global API keys
- Updated API authentication to use separate middleware for job submissions
- Enhanced CLI job creation to display generated API keys with security warnings
- Updated job model to include API key field with nullable database support
- Modified API handlers to support API key generation and updates

### Features
- Job management: Create, Read, Update, Delete operations
- Arbitrary user-defined labels per job
- Per-job automatic failure thresholds
- Job status tracking (active, maintenance, paused)
- HTTP API with authentication middleware
- Prometheus metrics with proper labeling
- CLI commands for all operations
- YAML configuration with environment variable overrides
- Health check endpoints
- Request logging and monitoring

## [0.1.0] - 2025-10-30

### Added
- Initial project structure and architecture
- Complete specification document
- Core application framework
