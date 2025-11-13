# Change Proposal: Add Embedded Dashboard

**Change ID**: `add-embedded-dashboard`
**Type**: New Capability
**Status**: Draft
**Created**: 2025-11-02

## Problem Statement

Currently, the cron-exporter provides excellent Prometheus metrics integration for monitoring cron jobs, but users who don't use the Prometheus/Grafana stack have limited visibility into their job statuses and no convenient way to manage jobs without using the CLI or direct API calls.

**Current Limitations:**
- No visual interface for users without Prometheus/Grafana
- Job management requires CLI expertise or API knowledge
- Status monitoring limited to metrics endpoint or logs
- No real-time job status visualization
- Barrier to adoption for teams preferring self-contained solutions

## Proposed Solution

Add an optional embedded web dashboard built using the **Gin web framework** that provides:

1. **Live Status Dashboard**: Simple, clean view of all jobs with current status and last run times
2. **Job Management Interface**: Full CRUD operations for jobs via web forms
3. **Job History View**: Recent job execution results and failure details
4. **Maintenance Operations**: Easy job pause/resume toggle functionality
5. **Simple Interface**: Clean, responsive design that works without external dependencies

### Gin Framework Choice

The dashboard will be built using **Gin** (<https://github.com/gin-gonic/gin>) for the following reasons:

- **Most Popular Go Web Framework**: Widely adopted with excellent community support
- **Lightweight & Fast**: Minimal overhead, perfect for simple admin interfaces
- **Built-in Template Engine**: HTML template rendering with layout support
- **Middleware Support**: Easy authentication integration with existing API key system
- **Simple & Focused**: No unnecessary complexity, just what's needed for job management

## Benefits

### Primary Benefits
- **Lower Barrier to Entry**: Users can immediately see job status without external tools
- **Self-Contained Solution**: Complete monitoring solution without dependencies
- **Operational Convenience**: Quick job management without CLI commands
- **Better Troubleshooting**: Visual interface for diagnosing job issues

### Secondary Benefits
- **Complementary to Prometheus**: Dashboard doesn't replace metrics, enhances accessibility
- **Development/Testing**: Easier debugging during development
- **Demos/Proof-of-Concepts**: Better for showcasing capabilities
- **Small Team Friendly**: Reduces toolchain complexity for smaller deployments

## Design Principles

1. **Optional by Default**: Dashboard must be completely optional, disabled by default
2. **Lightweight Framework**: Use Gin web framework for minimal overhead and complexity
3. **Simple & Clean**: Focus on essential functionality without unnecessary features
4. **Security First**: NO authentication bypasses or dev-mode exceptions - identical auth in dev/prod
5. **Mobile Friendly**: Responsive design using simple CSS (Bootstrap for styling)
6. **Self-Contained**: Embed all static assets (CSS, JS) in the binary

### Security Requirements

**Mandatory Security Principle**: The dashboard implementation must maintain identical authentication behavior between development and production environments. This means:

- **NO development-only authentication bypasses**
- **NO configuration flags to disable authentication**
- **NO special dev-mode authentication shortcuts**

Development testing will use generated API keys in configuration files, ensuring developers experience the exact same authentication flow as end users.

## High-Level Architecture

### Frontend

- **Technology**: HTML templates with Bootstrap CSS for clean, responsive design
- **Interactivity**: HTMX for dynamic form submissions and live updates (optional JavaScript fallback)
- **Styling**: Bootstrap 5 embedded in binary using Go 1.16+ embed directive
- **Updates**: Server-sent events for real-time job status updates
- **Asset Structure**: Organized in `pkg/dashboard/assets/` with templates and static files embedded via `//go:embed`

### Backend Integration

- **Web Framework**: Gin router mounted as sub-router at `/dashboard/*` path within existing HTTP server using `http.StripPrefix`
- **Authentication**: HTTP Basic Auth with API key as password (username can be anything), validated against existing admin API key system
- **Session Management**: Stateless authentication - no session storage required, browser handles Basic Auth caching
- **Data Source**: Direct access to existing `JobStore` and `JobResultStore` with existing schema
- **Database**: Uses existing tables with performance indexes added for dashboard queries (no schema changes)
- **Templates**: Go html/template with Gin's template rendering
- **Integration**: Single HTTP server maintains self-contained principle with isolated dashboard routes

### Error Handling & User Experience

**Multi-layered User Feedback System:**

- **Toast Notifications**: Non-blocking feedback for CRUD operations (success/error)
  - Bootstrap toast components with auto-dismiss functionality
  - HTMX integration for seamless operation feedback without page reload
- **Inline Validation**: Contextual form error display
  - Bootstrap validation classes for immediate visual feedback
  - Field-level error messages with clear guidance for resolution
- **Modal Error Dialogs**: Critical system errors (authentication failures, server errors)
  - Bootstrap modal components for blocking errors requiring user attention
  - Clear error messages with suggested corrective actions
- **Graceful Degradation**: Full functionality maintained when JavaScript is disabled
  - Server-side validation with redirect-based feedback mechanisms
  - Progressive enhancement approach ensures universal accessibility

### Configuration

Dashboard configuration extends the existing config struct with backward compatibility:

```go
// Extends existing Config struct in pkg/config/config.go
type DashboardConfig struct {
    Enabled         bool   `yaml:"enabled" default:"false"`
    Path            string `yaml:"path" default:"/dashboard"`
    Title           string `yaml:"title" default:"Cron Monitor"`
    RefreshInterval int    `yaml:"refresh_interval" default:"5"`
    PageSize        int    `yaml:"page_size" default:"25"`
    AuthRequired    bool   `yaml:"auth_required" default:"true"`
}
```

```yaml
# Example configuration (integrates with existing config file)
dashboard:
  enabled: false          # Disabled by default
  path: "/dashboard"      # Dashboard URL path prefix
  title: "Cron Monitor"   # Page title
  refresh_interval: 5     # Auto-refresh interval in seconds
  page_size: 25           # Default number of jobs per page
  auth_required: true     # Require admin API key
```

### Route Structure

```
# HTML Pages (server-rendered)
GET  /dashboard/                     -> redirect to /dashboard/jobs
GET  /dashboard/jobs                -> job list page (main dashboard)
GET  /dashboard/jobs/new            -> create job form
GET  /dashboard/jobs/{id}           -> job detail/edit form
POST /dashboard/jobs                -> create job (form submission)
PUT  /dashboard/jobs/{id}           -> update job (form submission)
DELETE /dashboard/jobs/{id}         -> delete job (form submission)

# HTMX/AJAX Endpoints (JSON responses)
GET  /dashboard/api/jobs            -> job list JSON (for live updates)
GET  /dashboard/api/jobs/{id}/status -> job status JSON (for SSE)
POST /dashboard/api/jobs/{id}/toggle -> pause/resume job
GET  /dashboard/events              -> Server-Sent Events endpoint

# Static Assets
GET  /dashboard/static/css/*        -> embedded CSS files
GET  /dashboard/static/js/*         -> embedded JS files
```

## Implementation Approach

### Phase 1: Core Dashboard (MVP)

- Gin router integration as sub-router within existing HTTP server
- Job list view with status indicators and basic filtering
- Job CRUD forms (create, edit, delete) with validation that works without JavaScript
- Basic HTMX integration for inline form validation and feedback
- HTTP Basic Auth middleware using existing API key system
- **Mise tasks**: `mise run dev`, `mise run build`, `mise run test` (NO direct go/npm commands)
- **Playwright tests**: Basic CRUD operations and authentication flow via `mise run dashboard-test`

### Phase 2: Enhanced Features

- Real-time job status updates using Server-Sent Events
- Advanced HTMX features for real-time search/filtering and dynamic updates
- Job execution history view with pagination
- Maintenance mode toggle and bulk operations
- **Enhanced Playwright tests**: Real-time updates, search/filtering, responsive design (all via mise tasks)### Phase 3: Advanced Features (Future)

- Simple charts for job execution trends (using Chart.js)
- Advanced search and filtering capabilities
- Export functionality (CSV/JSON)
- Dark/light theme toggle
- **Additional Playwright tests**: Charts, advanced features, edge cases (via mise task orchestration)

## Development Workflow Requirements

### Mise Task Management

All dashboard development must follow the established mise task pattern with NO direct tool usage:

```toml
# .mise.toml additions for dashboard
[tasks.dashboard-dev]
description = "Start development server with dashboard enabled and API key auth"
run = "go run ./cmd/cronmetrics serve --config dev-config-dashboard.yaml"

[tasks.dashboard-test]
description = "Run dashboard-specific Playwright tests with automatic server lifecycle"
run = [
    "mise run build",  # Always rebuild before testing
    "./scripts/run-playwright-tests.sh"  # Script handles server start/stop
]

[tasks.test]
description = "Run complete test suite including dashboard Playwright tests"
run = [
    "mise run build",  # Rebuild on any backend changes
    "go test ./...",
    "mise run dashboard-test"  # This will handle its own server lifecycle
]

[tasks.build]
description = "Build binary with embedded dashboard assets"
run = "go build -o bin/cronmetrics ./cmd/cronmetrics"

[tasks.lint]
description = "Run linting on dashboard Go code"
run = "golangci-lint run ./..."

[tasks.dashboard-install]
description = "Install Playwright and dashboard dependencies"
run = "npx playwright install chromium"

[tasks.watch-dev]
description = "Watch for changes and rebuild automatically during development"
run = "air -c .air.toml"  # Auto-rebuild on file changes

# Example dev-config-dashboard.yaml
dashboard:
  enabled: true
  path: "/dashboard"
  auth_required: true  # NO bypass - always required
api:
  admin_key: "test-admin-key-12345"  # Generated test API key

# Example test-config-dashboard.yaml (for Playwright tests)
database:
  driver: sqlite3
  connection_string: ":memory:"  # In-memory database for test isolation
dashboard:
  enabled: true
  path: "/dashboard"
  auth_required: true
api:
  admin_key: "test-admin-key-12345"
```

### Playwright Test Lifecycle Script

```bash
#!/bin/bash
# scripts/run-playwright-tests.sh
set -e

CONFIG_FILE="test-config-dashboard.yaml"
PID_FILE="/tmp/cronmetrics-test.pid"

# Cleanup function
cleanup() {
    if [ -f "$PID_FILE" ]; then
        PID=$(cat "$PID_FILE")
        echo "Stopping test server (PID: $PID)"
        kill "$PID" 2>/dev/null || true
        rm -f "$PID_FILE"
    fi
}

# Set trap to cleanup on exit
trap cleanup EXIT

# Start server in background
echo "Starting test server..."
./bin/cronmetrics serve --config "$CONFIG_FILE" &
echo $! > "$PID_FILE"

# Wait for server to be ready
sleep 3

# Run Playwright tests headless
echo "Running Playwright tests..."
npx playwright test --config playwright-dashboard.config.js --reporter=html --reporter=line

echo "Tests completed successfully"
```

### Playwright Configuration

```javascript
// playwright-dashboard.config.js
const { devices } = require('@playwright/test');

module.exports = {
  testDir: './test/e2e-dashboard',
  timeout: 30 * 1000,
  expect: {
    timeout: 5000
  },
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: [
    ['html', { outputFolder: 'test-results/playwright-report', open: 'never' }],
    ['line']
  ],
  use: {
    baseURL: 'http://localhost:8080',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    headless: true,  // Always headless - no browser UI
  },
  projects: [
    {
      name: 'chromium',
      use: {
        ...devices['Desktop Chrome'],
      },
    },
  ],
  webServer: undefined,  // Server managed by shell script, not Playwright
};
```

### Tool Version Management & Build Automation

**Mandatory Requirements**: All development commands must go through mise tasks with automatic rebuilding:

- **NO direct `go` commands** - Use `mise run build`, `mise run test`, etc.
- **NO direct `npm`/`npx` commands** - All Node.js tools via mise tasks
- **NO direct `golangci-lint`** - Use `mise run lint`
- **Automatic rebuilds** - Backend changes trigger rebuild before tests
- **Server lifecycle management** - Tests automatically start/stop server
- **Headless execution** - Tests run without browser UI or user interaction
- **Consistent environments** - All developers use exact same tool versions
- **Documentation through tasks** - `.mise.toml` serves as executable documentation

### Playwright Testing Strategy

- **Chrome-only testing**: No need for cross-browser compatibility
- **Complete workflow coverage**: Every user interaction path tested
- **Integration with existing CI**: Playwright tests run as part of `mise run test`
- **Test data management**: Isolated test database for consistent test runs
- **Tool version consistency**: Playwright installed via `mise run dashboard-install`

## Risk Assessment

### Technical Risks

- **Framework Dependency**: Reliance on external Gin framework
  - *Mitigation*: Gin is the most popular Go web framework with excellent stability
- **Binary Size Increase**: Additional templates, CSS, and JS assets
  - *Mitigation*: Minimal asset footprint, optimize embedded resources (~200KB total)
- **Security Surface**: New web interface increases attack surface
  - *Mitigation*: Reuse existing authentication, follow secure coding practices

### Product Risks
- **Scope Creep**: Dashboard could become overly complex
  - *Mitigation*: Strict adherence to MVP scope, clear requirements
- **User Confusion**: Two interfaces might confuse users
  - *Mitigation*: Clear documentation about when to use each interface
- **Performance Impact**: Dashboard queries could affect API performance
  - *Mitigation*: Efficient queries, optional caching, load testing

## Acceptance Criteria

### Core Dashboard Functionality

**AC-1: Dashboard Accessibility**

- [ ] Dashboard is accessible at configurable URL path (default: `/dashboard`)
- [ ] Dashboard can be enabled/disabled via configuration
- [ ] Dashboard is disabled by default for backward compatibility
- [ ] Configuration validation prevents path conflicts with existing routes

**AC-2: Job Management Interface**
- [ ] Complete job CRUD operations available via web interface
- [ ] Job creation form with validation for all required fields
- [ ] Job editing preserves existing data and allows modifications
- [ ] Job deletion requires confirmation and provides clear feedback
- [ ] Job status display shows current state, last run time, and failure reasons

**AC-3: Authentication Integration**
- [ ] Dashboard reuses existing admin API key authentication without modification
- [ ] Unauthenticated access returns appropriate HTTP 401 responses
- [ ] NO authentication bypass or dev-mode exceptions implemented
- [ ] All dashboard operations respect existing authorization model consistently
- [ ] Development and production use identical authentication flow

### Gin Framework Integration & Features

**AC-4: Gin Framework Integration**

- [ ] Gin router successfully integrated as sub-router within existing HTTP server
- [ ] HTML templates rendered using Gin's template engine
- [ ] Form validation and submission handled with proper error messages
- [ ] Bootstrap CSS provides clean, responsive interface
- [ ] Authentication middleware integrates seamlessly with existing API key system

**AC-5: Dynamic Interface Features**

- [ ] Job list supports pagination and basic sorting
- [ ] Form submissions provide immediate feedback and validation
- [ ] HTMX enables dynamic updates without full page reloads
- [ ] Search results filter job list in real-time
- [ ] Server-sent events provide real-time job status updates

### Responsive Design & Styling

**AC-6: Bootstrap Integration**

- [ ] Bootstrap 5 CSS framework embedded in binary for consistent styling
- [ ] Clean, professional design with good typography and spacing
- [ ] Responsive grid system works across all device sizes
- [ ] Form components use Bootstrap styling for consistency
- [ ] Status indicators use Bootstrap badge/alert components

**AC-7: Mobile & Accessibility**

- [ ] Mobile-optimized layout with touch-friendly controls
- [ ] Responsive navigation that works on small screens
- [ ] Accessible color contrast and keyboard navigation
- [ ] Loading states and user feedback clearly visible

### Simple Search and Filtering

**AC-8: Basic Search & Filter**

- [ ] Search input filters jobs by name, host, or status
- [ ] Filter dropdown for job status (active, maintenance, paused)
- [ ] Search results update in real-time as user types
- [ ] Clear search/filter functionality
- [ ] Search state preserved during navigation

**AC-9: Job List Features**

- [ ] Sortable columns for name, host, status, last run time
- [ ] Pagination with configurable page sizes (25, 50, 100)
- [ ] Job status indicators with clear visual distinctions
- [ ] Quick action buttons for pause/resume/delete
- [ ] Responsive table design that works on mobile

### Performance and Data Loading

**AC-10: Simple Performance Requirements**

- [ ] Job list loads quickly with up to 1000+ jobs
- [ ] Pagination prevents overwhelming the browser
- [ ] HTMX updates only necessary page sections
- [ ] Minimal JavaScript footprint for fast loading
- [ ] Graceful degradation when JavaScript disabled

**AC-11: Responsive Design**
- [ ] Mobile-optimized layout with touch-friendly controls
- [ ] Tablet layout with appropriate spacing and navigation
- [ ] Desktop layout maximizes screen real estate efficiently
- [ ] All interactive elements meet minimum touch target sizes (44px)
- [ ] Text remains readable at all screen sizes

### Performance Requirements

**AC-12: Binary Size and Resource Usage**

- [ ] Total binary size increase remains under 300KB (Gin + Bootstrap + templates)
- [ ] Static assets (CSS, JS, templates) efficiently embedded in binary
- [ ] Minimal JavaScript footprint (HTMX ~14KB, Chart.js optional)

**AC-13: Response Times**
- [ ] Dashboard home page loads within 2 seconds
- [ ] Job list API endpoints respond within 500ms
- [ ] Search results appear within 300ms of user input
- [ ] Theme switches complete within 100ms

### Backward Compatibility

**AC-14: Existing Functionality Preservation**
- [ ] All existing API endpoints remain unchanged
- [ ] CLI functionality operates identically
- [ ] Metrics endpoint performance unaffected
- [ ] Configuration file format maintains backward compatibility

**AC-15: Optional Feature Impact**
- [ ] Dashboard disabled has zero impact on core functionality
- [ ] Server startup time not significantly increased
- [ ] Memory usage unchanged when dashboard disabled

### Security and Data Integrity

**AC-16: Input Validation and XSS Protection**
- [ ] All form inputs validated against expected schemas
- [ ] HTML template rendering uses automatic escaping
- [ ] URL parameters sanitized before processing
- [ ] Content Security Policy headers prevent XSS attacks

**AC-17: CSRF Protection**
- [ ] State-changing operations validate request origin
- [ ] Cross-origin requests appropriately rejected
- [ ] SameSite cookie attributes implemented where applicable

**AC-18: No Authentication Bypasses**
- [ ] NO development-only authentication bypasses implemented
- [ ] NO configuration options to disable authentication for development
- [ ] Development and production environments use identical authentication
- [ ] Generated API keys in config files used for development testing

### User Experience

**AC-19: Usability Goals**
- [ ] New users understand job status within 30 seconds of accessing dashboard
- [ ] Job creation/editing possible without consulting documentation
- [ ] Status information updates automatically without user intervention
- [ ] Error messages provide clear, actionable guidance

**AC-20: Accessibility**
- [ ] Keyboard navigation works for all interactive elements
- [ ] Screen reader compatibility maintained
- [ ] Color contrast ratios meet WCAG 2.1 AA standards
- [ ] Focus indicators visible and consistent

### Testing and Quality Assurance

**AC-21: Test Coverage**
- [ ] Unit test coverage remains at 100%
- [ ] Integration tests cover all dashboard endpoints
- [ ] End-to-end tests validate complete user workflows
- [ ] Playwright tests cover all dashboard functionality (Chrome only)

**AC-22: Error Handling**
- [ ] Network failures gracefully handled with user feedback
- [ ] Invalid configurations provide clear error messages
- [ ] Database connection issues don't crash dashboard
- [ ] Partial failures allow continued operation where possible

### Developer Experience Requirements

**AC-23: Mise Task Management**
- [ ] `mise run test` - Runs complete test suite including Playwright tests
- [ ] `mise run build` - Builds binary with embedded dashboard assets
- [ ] `mise run dev` - Starts development server with dashboard enabled
- [ ] `mise run dashboard-test` - Runs dashboard-specific Playwright tests
- [ ] All dashboard-related tasks documented in `.mise.toml`
- [ ] NO direct `go` or `npm` commands in development workflow
- [ ] ALL tool execution goes through mise tasks for version consistency
- [ ] Backend changes trigger automatic rebuild through mise tasks
- [ ] Playwright tests automatically start/stop server during test execution
- [ ] Tests run headless without requiring developer interaction

**AC-24: Playwright Testing Requirements**
- [ ] Playwright tests cover all CRUD operations (create, read, update, delete jobs)
- [ ] Tests validate job status display and real-time updates
- [ ] Form validation and error handling thoroughly tested
- [ ] Authentication flow tested (login/logout with API key)
- [ ] Search and filtering functionality verified
- [ ] Responsive design tested on different viewport sizes
- [ ] Tests run against Chrome browser only (as specified)
- [ ] Playwright configuration integrated with existing test infrastructure
- [ ] Tests automatically start server in background before execution
- [ ] Server automatically killed after test completion
- [ ] Tests run headless without starting web server UI or waiting for user input
- [ ] Test results available for review without developer interaction

## Design Decisions

Based on requirements analysis, the following design decisions have been made:

1. **Web Framework**: **Gin** - Most popular Go web framework, lightweight and well-documented
2. **UI Framework**: **Bootstrap 5** - Clean, responsive design with minimal custom CSS
3. **Interactivity**: **HTMX** - Dynamic updates without complex JavaScript
4. **Authentication**: **Gin Middleware** - Simple integration with existing API key system
5. **Templates**: **Go html/template** - Standard library templating with Gin integration

## Dependencies

### Internal Dependencies
- Existing HTTP server infrastructure
- Current authentication system
- Job and JobResult data models
- Configuration system

### External Dependencies

- **Gin Framework** (<https://github.com/gin-gonic/gin>) - Lightweight Go web framework
- **Bootstrap 5** - CSS framework for responsive design (embedded in binary)
- **HTMX** (~14KB) - JavaScript library for dynamic interactions (embedded in binary)
- **Chart.js** (optional) - Simple charts for job execution trends

### Development Dependencies

- **Playwright** - End-to-end testing framework for dashboard functionality
- **Mise Task Runner** - Task management and documentation via `.mise.toml`

## Timeline Estimate

- **Research & Design**: 2-3 days
- **Phase 1 Implementation**: 5-7 days
- **Playwright Test Setup & Implementation**: 3-4 days
- **Integration Testing & Documentation**: 2-3 days
- **Phase 2 Enhancement**: 3-5 days

**Total**: ~3-4 weeks for full implementation (including comprehensive Playwright testing)

## Alternatives Considered

1. **External Dashboard**: Separate application/container
   - *Rejected*: Increases deployment complexity, goes against self-contained principle

2. **Grafana Plugin**: Custom plugin for existing Grafana instances
   - *Rejected*: Only helps users already using Grafana

3. **CLI Dashboard**: Terminal-based dashboard (like htop)
   - *Rejected*: Limited functionality, not web-accessible

4. **API-Only Approach**: Provide better API documentation and examples
   - *Rejected*: Still requires technical expertise to implement

## Conclusion

The embedded dashboard addresses a real gap in accessibility for users who need visual job monitoring without the complexity of the Prometheus/Grafana stack. The proposed solution maintains the project's principles of simplicity and self-containment while providing significant value to users.

The phased approach allows for iterative development and validation of user needs, while the optional nature ensures existing users are not impacted.
