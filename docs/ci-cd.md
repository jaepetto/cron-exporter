# CI/CD Documentation

This document explains the GitHub Actions CI/CD pipeline setup for the cron-exporter project.

## Overview

The project includes comprehensive GitHub Actions workflows for automated testing, building, and deployment:

- **CI Pipeline** (`.github/workflows/ci.yml`) - Runs on every push to main and pull requests
- **Release Pipeline** (`.github/workflows/release.yml`) - Creates releases when tags are pushed
- **Docker Pipeline** (`.github/workflows/docker.yml`) - Builds and publishes container images
- **Dependency Updates** (`.github/dependabot.yml`) - Keeps dependencies up to date

## CI Pipeline (`ci.yml`)

Runs on every push to `main` branch and pull requests. Includes:

### Test Job
- Sets up Go 1.21.9 and mise
- Caches Go modules for faster builds
- Runs linting with `mise run lint`
- Executes unit tests with `mise run test-unit`
- Executes integration tests with `mise run test-integration`
- Executes e2e tests with `mise run test-e2e`
- Generates coverage reports and uploads to Codecov

### Build Job
- Depends on successful test completion
- Builds single binary with `mise run build`
- Builds multi-platform binaries with `mise run build-all`
- Uploads build artifacts with 30-day retention

### Security Job
- Runs Gosec security scanner to detect common security issues
- Executes govulncheck for vulnerability detection in dependencies
- Scans for issues like command injection (G204), file inclusion (G304), improper permissions (G302), and unhandled errors (G104)
- **Zero tolerance policy**: Any security issues cause the build to fail
- Security scan results are uploaded to GitHub Security tab for review

## Release Pipeline (`release.yml`)

Triggered when version tags (e.g., `v1.0.0`) are pushed:

- Runs full test suite before releasing
- Builds release binaries with version information using `mise run build-release`
- Generates changelog from git commits
- Creates GitHub release with:
  - Compressed archives for all supported platforms
  - Auto-generated release notes
  - Proper semantic versioning support

### Supported Platforms
- Linux (amd64, arm64)
- macOS (amd64, arm64)
- Windows (amd64)

## Docker Pipeline (`docker.yml`)

Builds and publishes container images:

- Builds multi-architecture images (linux/amd64, linux/arm64)
- Publishes to GitHub Container Registry (`ghcr.io`)
- Tags images with:
  - `main` for main branch
  - `sha-<commit>` for specific commits
  - Semantic version tags for releases
- Includes security scanning with Trivy
- Results uploaded to GitHub Security tab

## Container Usage

The Docker image is available at `ghcr.io/jaepetto/cron-exporter`:

```bash
# Pull latest image
docker pull ghcr.io/jaepetto/cron-exporter:main

# Run with Docker Compose
docker-compose up -d

# Run standalone
docker run -d \
  -p 8080:8080 \
  -p 9090:9090 \
  -v cronmetrics_data:/data \
  ghcr.io/jaepetto/cron-exporter:main
```

## Dependency Management

Dependabot automatically:
- Updates Go modules weekly
- Updates GitHub Actions weekly
- Updates Docker base images weekly
- Creates pull requests for review

## Setting Up CI/CD for New Repositories

1. **Enable GitHub Actions** in repository settings
2. **Add secrets** if needed (none required for basic setup)
3. **Configure branch protection** for main branch:
   - Require status checks to pass
   - Require CI pipeline to pass before merging
   - Include administrators in restrictions

## Monitoring and Badges

Add these badges to your README.md:

```markdown
[![CI/CD Pipeline](https://github.com/jaepetto/cron-exporter/actions/workflows/ci.yml/badge.svg)](https://github.com/jaepetto/cron-exporter/actions/workflows/ci.yml)
[![Release](https://github.com/jaepetto/cron-exporter/actions/workflows/release.yml/badge.svg)](https://github.com/jaepetto/cron-exporter/actions/workflows/release.yml)
[![Docker Build](https://github.com/jaepetto/cron-exporter/actions/workflows/docker.yml/badge.svg)](https://github.com/jaepetto/cron-exporter/actions/workflows/docker.yml)
```

## Creating Releases

To create a new release:

1. **Update version** in code if needed
2. **Create and push tag**:
   ```bash
   git tag -a v1.0.0 -m "Release v1.0.0"
   git push origin v1.0.0
   ```
3. **GitHub Actions** will automatically:
   - Run full test suite
   - Build release binaries
   - Create GitHub release
   - Build and push Docker images

## Local Development

The CI/CD pipeline mirrors local development tasks:

```bash
# Run what CI runs
mise run ci          # Equivalent to CI pipeline
mise run lint        # Linting
mise run test-all    # All tests
mise run build       # Build binary
mise run build-all   # Multi-platform builds
```

## Troubleshooting

### Failed Tests
- Check test logs in GitHub Actions
- Run locally: `mise run test-all`
- Ensure binary is built: `mise run build`

### Failed Builds
- Verify Go version compatibility
- Check for missing dependencies
- Test locally: `mise run build-all`

### Docker Issues
- Test container locally: `docker build -t test .`
- Check base image vulnerabilities
- Verify multi-arch support

### Security Scan Issues
- Run locally: `mise run security`
- Install gosec: `mise run security-install`
- Generate detailed reports: `mise run security-report`
- Common fixes:
  - G204: Validate command arguments before `exec.Command()`
  - G304: Validate file paths, prevent directory traversal
  - G302: Use restrictive file permissions (0600/0644)
  - G104: Handle all error returns appropriately

### Release Issues
- Ensure proper semantic versioning (v1.0.0, not 1.0.0)
- Check tag push: `git push origin --tags`
- Verify release permissions in repository settings
