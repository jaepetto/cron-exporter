# Contributing to Cron Metrics Collector & Exporter

Welcome! We're excited that you're interested in contributing to the cron-exporter project. This document provides guidelines and instructions for contributing.

## Quick Start for New Contributors

### Prerequisites

- Go 1.23+ (managed via mise)
- Git
- macOS, Linux, or Windows with WSL

### Setup Development Environment

1. **Clone and setup:**

   ```bash
   git clone https://github.com/jaepetto/cron-exporter
   cd cron-exporter
   mise install  # Installs Go version from .tool-versions
   go mod tidy
   ```

2. **Verify setup works:**

   ```bash
   # Build the application
   mise run build

   # Run all tests (REQUIRED - must pass 100%)
   mise run test-all

   # Start development server
   mise run dev
   ```

3. **Test cross-platform builds:**

   ```bash
   # Test building for all platforms
   mise run build-all

   # Clean up artifacts
   mise run clean
   ```

## Development Workflow

### 🚨 MANDATORY TESTING REQUIREMENTS

**BEFORE making any code changes:**

```bash
mise run test-all  # ALL tests must pass - NO EXCEPTIONS
```

**AFTER making any code changes:**

```bash
mise run test-all  # ALL tests must still pass - 100% required
```

**This project maintains 100% passing test coverage. NO code changes are accepted without all tests passing.**

### Standard Development Process

1. **Create a feature branch:**
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes and test frequently:**
   ```bash
   # Run tests while developing
   mise run test
   mise run test-integration
   mise run build
   ```

3. **Before committing, run full validation:**
   ```bash
   mise run ci  # Runs: lint + security + test-all + build
   ```

4. **Commit and push:**
   ```bash
   git add .
   git commit -m "feat: your descriptive commit message"
   git push origin feature/your-feature-name
   ```

5. **Create Pull Request:**
   - GitHub Actions will automatically run the full CI pipeline
   - All tests must pass before the PR can be merged
   - Security scans, linting, and multi-platform builds are automated
   - Code coverage reports will be generated and checked

### GitHub Actions CI/CD Pipeline

This project uses a comprehensive CI/CD pipeline that runs automatically:

#### On Pull Requests and Pushes to Main:
- **Testing**: Full test suite (unit, integration, e2e)
- **Security**: Gosec security scanning and govulncheck
- **Linting**: Code formatting and style checks
- **Building**: Multi-platform binary builds with artifact storage
- **Coverage**: Code coverage reporting via Codecov
- **Docker**: Container image building and security scanning

#### On Version Tags (Releases):
- **Release Builds**: Creates binaries for all supported platforms
- **GitHub Releases**: Automatic release creation with changelog
- **Container Publishing**: Multi-architecture images to GitHub Container Registry
- **Documentation**: Automatic updates to release documentation

#### Weekly Automated Maintenance:
- **Dependencies**: Dependabot creates PRs for Go module updates
- **Security**: Automated security vulnerability scanning
- **Infrastructure**: GitHub Actions and Docker base image updates

### Available Development Commands

| Command | Description | When to Use |
|---------|-------------|-------------|
| `mise run test` | Unit tests only | During development |
| `mise run test-all` | All tests (unit + integration + e2e) | **Required before commits** |
| `mise run test-integration` | Integration tests | When testing API/DB changes |
| `mise run test-e2e` | End-to-end workflows | When testing full scenarios |
| `mise run build` | Single platform build | Regular development |
| `mise run build-all` | Cross-platform builds | Testing portability |
| `mise run dev` | Development server | Manual testing |
| `mise run lint` | Code formatting/linting | Before commits |
| `mise run security` | Security vulnerability scan | **Required before commits** |
| `mise run security-install` | Install gosec scanner | First time setup |
| `mise run security-report` | Detailed security reports | Investigation/CI |
| `mise run clean` | Clean build artifacts | When needed |
| `mise run ci` | Full CI pipeline (includes security) | **Required before PRs** |
| `mise run ci-full` | CI + multi-platform builds | Release preparation |

### Docker Development Workflow

For contributors working with containers or testing the full stack:

#### Local Development with Docker
```bash
# Build and test container locally
docker build -t cronmetrics-dev .

# Run with full monitoring stack
docker-compose up -d

# Check container health
docker-compose ps
docker-compose logs cronmetrics

# Clean up
docker-compose down -v
```

#### Testing Container Images
```bash
# Test multi-architecture builds (requires buildx)
docker buildx build --platform linux/amd64,linux/arm64 -t test .

# Security scanning locally (if trivy is installed)
trivy image cronmetrics-dev

# Test different configurations
docker run -e CRONMETRICS_LOG_LEVEL=debug cronmetrics-dev
```

#### Container Registry Access
- **Development**: Images are automatically built on every push
- **Releases**: Tagged images are published to `ghcr.io/jaepetto/cron-exporter`
- **Pull Requests**: Container builds are tested but not published

## Code Standards

### Architecture Guidelines

- **Dual-interface design**: REST API + CLI both manage the same SQLite store
- **Job model**: Auto-incrementing IDs, per-job API keys, arbitrary labels (JSON)
- **Prometheus metrics**: Include status labels for maintenance mode suppression
- **SQLite schema**: Use JSON columns for flexible label storage
- **Structured logging**: Use logrus with contextual fields throughout

### Code Quality Requirements

1. **100% Test Coverage**: All new code must have comprehensive tests
2. **Security Scanning**: All code must pass gosec security analysis with zero issues
3. **Documentation**: Update all relevant `.md` files with changes
4. **Error Handling**: Proper error responses with structured logging
5. **Input Validation**: Validate all inputs, use proper authentication
6. **Performance**: Consider database query efficiency and memory usage

### Security Requirements

**MANDATORY**: All code must pass security scanning before submission.

```bash
# Install gosec on first use
mise run security-install

# Run security scan (must show 0 issues)
mise run security

# Generate detailed reports for investigation
mise run security-report
```

**Common security issues to avoid:**
- **G204**: Command injection - validate all arguments passed to `exec.Command()`
- **G304**: File inclusion - validate file paths and prevent directory traversal
- **G302**: File permissions - use restrictive permissions (0600/0644) for sensitive files
- **G104**: Unhandled errors - always handle error returns appropriately

All security fixes must:
- Include proper input validation
- Use `#nosec` comments only when security has been verified
- Maintain existing functionality while improving security
- Include tests that verify both security and functionality

### Testing Requirements

**Every contribution must include:**

- **Unit tests** for new functions/methods
- **Integration tests** for API endpoints or CLI commands
- **End-to-end tests** for new user workflows
- **Documentation updates** in relevant `.md` files

**Test patterns to follow:**

```go
// Use testutil helpers for common setup
func TestNewFeature(t *testing.T) {
    db := testutil.SetupTestDatabase(t)
    defer db.Close()

    server := testutil.SetupTestServer(t, db)
    defer server.Close()

    // Your test code here...
}
```

### Database Access Policy
- All new and existing database code **must** use [sqlx](https://github.com/jmoiron/sqlx). Direct use of `database/sql` is not permitted.
- All PRs must ensure parameterized queries and avoid string interpolation for SQL.
- See `pkg/model/database.go` for canonical usage.

## Project Structure

```text
cmd/cronmetrics/        # Main application entry point
pkg/
  api/                  # REST API server implementation
  metrics/              # Prometheus metrics collector
  config/               # Configuration management
  model/                # Database models and operations
  util/                 # Utility functions (API keys, etc.)
internal/
  cli/                  # Cobra CLI commands
  testutil/             # Test utilities and helpers
test/
  integration/          # Integration tests
  e2e/                  # End-to-end workflow tests
migrations/             # Database schema migrations
docs/                   # API documentation and specs
```

## Common Contribution Types

### Adding New API Endpoints

1. Add endpoint to `pkg/api/server.go`
2. Add model operations to `pkg/model/`
3. Add integration tests in `test/integration/api_test.go`
4. Update `docs/openapi.yaml` with new endpoint
5. Test with: `mise run test-integration`

### Adding New CLI Commands

1. Add command to `internal/cli/`
2. Add CLI tests in `test/integration/cli_test.go`
3. Update help documentation
4. Test with: `mise run test-integration`

### Enhancing Metrics

1. Update `pkg/metrics/collector.go`
2. Add metrics tests in `test/integration/metrics_test.go`
3. Verify Prometheus format compliance
4. Test with: `mise run test-integration`

### Adding New Job Features

1. Update `pkg/model/job.go` and database schema
2. Add migration file in `migrations/`
3. Update API endpoints and CLI commands
4. Add comprehensive tests for all affected components
5. Update documentation and OpenAPI spec

## Release Process

### Building for Release

```bash
# Test all platforms build successfully
mise run build-all

# Create release archives with version info
mise run build-release

# Verify archives
ls -la dist/
```

### Version Management

- Release versions are automatically detected from git tags
- Development builds use `dev-{commit-hash}` format
- Update `CHANGELOG.md` following [Keep a Changelog](https://keepachangelog.com/) format

## Getting Help

### Common Issues

1. **Tests failing?** Run `mise run test-all` to see detailed output
2. **Build issues?** Ensure Go version matches `.tool-versions`
3. **Database errors?** Check database file permissions and migrations are applied (no SQLite3 installation required)
4. **Port conflicts?** Tests use random ports, but check if development server is running

### Development Resources

- **Architecture**: See `docs/specs.md` for complete technical specifications
- **API Documentation**: Visit `/swagger/` endpoint when server is running
- **Test Examples**: Look at existing tests in `test/` directory for patterns
- **Debugging**: Use `mise run dev` for development server with debug logging

### Getting Support

1. Check existing [GitHub Issues](https://github.com/jaepetto/cron-exporter/issues)
2. Review this contributing guide and project documentation
3. Run `mise run test-all` to ensure your environment is working
4. Create a new issue with:
   - Go version (`go version`)
   - Operating system
   - Complete error messages
   - Steps to reproduce

## Pull Request Guidelines

### Before Submitting

- [ ] All tests pass: `mise run test-all`
- [ ] Code is formatted: `mise run lint`
- [ ] Cross-platform build works: `mise run build-all`
- [ ] Documentation is updated (README.md, CHANGELOG.md, etc.)
- [ ] Commit messages follow conventional format

### PR Description Template

```markdown
## Description
Brief description of changes and motivation.

## Type of Change
- [ ] Bug fix (non-breaking change)
- [ ] New feature (non-breaking change)
- [ ] Breaking change (fix or feature that causes existing functionality to change)
- [ ] Documentation update

## Testing
- [ ] All existing tests pass
- [ ] New tests added for new functionality
- [ ] Integration tests updated if needed
- [ ] End-to-end tests cover new workflows

## Checklist
- [ ] Code follows project style guidelines
- [ ] Self-review completed
- [ ] Documentation updated
- [ ] CHANGELOG.md updated
```

---

## Code of Conduct

This project follows the [Contributor Covenant Code of Conduct](https://www.contributor-covenant.org/). Please be respectful and inclusive in all interactions.

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
