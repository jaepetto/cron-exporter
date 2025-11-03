# Implementation Tasks: Add Embedded Dashboard

**Change ID**: `add-embedded-dashboard`
**Status**: Draft

## Phase 1: Core Dashboard Foundation

### Backend Infrastructure

- [ ] **T1.1**: Extend configuration system
  - Add `DashboardConfig` struct to existing `Config` in `pkg/config/config.go`
  - Fields: enabled, path, title, refresh_interval, page_size, auth_required
  - Update default configuration values with backward compatibility
  - Add validation for dashboard configuration (path conflicts, intervals)

- [ ] **T1.2**: Set up Gin framework integration
  - Add Gin dependency to `go.mod`
  - Create `pkg/dashboard/` package structure
  - Create `routes.go` for Gin route definitions
  - Create `handlers.go` for HTTP handlers
  - Create `middleware.go` for authentication middleware

- [ ] **T1.3**: Implement asset embedding system
  - Create `pkg/dashboard/assets/` directory structure
  - Set up Go 1.16+ embed directive for static assets
  - Embed Bootstrap 5 CSS (~200KB) and JavaScript
  - Embed HTMX library (~14KB) for dynamic interactions
  - Create `assets.go` with embedded file serving handlers

- [ ] **T1.4**: Integrate Gin sub-router with existing server
  - Mount Gin router at `/dashboard/*` in existing HTTP server
  - Use `http.StripPrefix` for proper path handling
  - Maintain single HTTP server architecture
  - Add dashboard initialization to server startup

### Authentication & Security

- [ ] **T1.5**: Implement HTTP Basic Auth integration
  - Create authentication middleware for Gin routes
  - Reuse existing admin API key validation system
  - Support API key as password (username can be anything)
  - Add stateless authentication without session storage
  - Handle authentication errors with proper HTTP responses

### Frontend Templates & UI

- [ ] **T1.6**: Create Go HTML templates with Bootstrap 5
  - Base layout template with Bootstrap 5 navigation
  - Dashboard home page template (job overview with status cards)
  - Job list template with Bootstrap tables and pagination
  - Job create/edit forms with Bootstrap form components
  - Job delete confirmation modal using Bootstrap modals
  - Error page templates with Bootstrap alert components

- [ ] **T1.7**: Implement HTMX dynamic interactions
  - Form submissions without page reload using HTMX
  - Live job status updates via HTMX polling
  - Dynamic search with real-time filtering
  - Inline form validation with Bootstrap feedback classes
  - Toast notifications for success/error feedback
  - Progressive enhancement approach for JavaScript-disabled browsers

- [ ] **T1.8**: Add error handling and user feedback
  - Bootstrap toast components for operation feedback
  - Inline validation errors with Bootstrap form validation
  - Modal dialogs for critical errors (authentication, server issues)
  - Graceful degradation with server-side fallbacks

### Data Integration & Business Logic

- [ ] **T1.9**: Create dashboard service layer
  - Integrate with existing `JobStore` and `JobResultStore`
  - Implement job summary calculations (status counts, failure rates)
  - Add job list service with filtering and pagination
  - Create performance indexes for dashboard queries (no schema changes)
  - Add automatic failure detection based on job thresholds

- [ ] **T1.10**: Implement core dashboard routes
  - `GET /dashboard/` - Dashboard home page
  - `GET /dashboard/jobs` - Job list with search/filter
  - `GET /dashboard/jobs/new` - Job creation form
  - `POST /dashboard/jobs` - Job creation handler
  - `GET /dashboard/jobs/:id/edit` - Job edit form
  - `PUT /dashboard/jobs/:id` - Job update handler
  - `DELETE /dashboard/jobs/:id` - Job deletion handler

## Phase 2: Enhanced Features & Polish

### Real-time Updates & Search

- [ ] **T2.1**: Implement job status monitoring
  - Server-sent events for real-time job status updates
  - HTMX polling fallback for SSE-incompatible browsers
  - Auto-refresh configuration per dashboard settings
  - Broadcast status changes to all connected clients

- [ ] **T2.2**: Advanced search and filtering
  - Multi-criteria search (host, name, status, labels)
  - Real-time search with HTMX partial updates
  - Search result highlighting and pagination
  - Filter by job status, maintenance mode, last execution time

- [ ] **T2.3**: Performance optimizations
  - Job list pagination (configurable page size, default 25)
  - Database query optimization with proper indexing
  - Caching frequently accessed dashboard data
  - Lazy loading for large job lists

### Mobile & Accessibility

- [ ] **T2.4**: Responsive design enhancements
  - Mobile-optimized Bootstrap layouts
  - Touch-friendly interaction elements (44px minimum targets)
  - Tablet-specific layout adjustments
  - Improved navigation for small screens

- [ ] **T2.5**: Accessibility compliance
  - WCAG 2.1 AA compliance for all dashboard components
  - Keyboard navigation support for all interactive elements
  - Screen reader compatibility with ARIA labels
  - High contrast color schemes and focus indicators

## Phase 3: Advanced Features (Future)

- [ ] **T3.1**: Add dashboard charts and graphs
  - Job success/failure rate charts
  - Execution duration trends
  - Failure pattern analysis

- [ ] **T3.2**: Implement advanced filtering and search
  - Full-text search across job names and labels
  - Advanced filter by labels, status, host
  - Saved filter presets

- [ ] **T3.3**: Add bulk operations UI
  - Multi-select job operations
  - Bulk status changes
  - Bulk label management

## Phase 3: Testing & Quality Assurance

### Mise Task Integration

- [ ] **T3.1**: Configure mise development tasks
  - `mise run dashboard-dev` - Start development server with dashboard enabled
  - `mise run dashboard-build` - Build binary with embedded dashboard assets
  - `mise run dashboard-test` - Run dashboard-specific Playwright tests
  - Update existing `mise run test` to include dashboard tests
  - Update existing `mise run build` to embed dashboard assets

### Unit Tests (100% Coverage Required)

- [ ] **T-UT1**: Gin handler unit tests
  - Test all dashboard HTTP handlers with mock dependencies
  - Test HTTP Basic Auth middleware integration
  - Test error handling and response formats
  - Test template rendering with various data scenarios

- [ ] **T-UT2**: Dashboard service layer unit tests
  - Test job summary calculations and status aggregation
  - Test search and filtering logic with various criteria
  - Test pagination and data transformation
  - Mock JobStore and JobResultStore dependencies

- [ ] **T-UT3**: Asset embedding and serving tests
  - Test embedded Bootstrap and HTMX asset serving
  - Test template compilation and rendering
  - Test static file serving with proper MIME types
  - Test asset caching headers and performance

### Integration Tests

- [ ] **T-IT1**: Full dashboard integration tests
  - Test complete job CRUD workflows via web interface
  - Test authentication integration with existing API key system
  - Test database integration with existing schema + indexes
  - Test error propagation and user feedback

- [ ] **T-IT2**: HTMX and real-time features integration
  - Test HTMX form submissions and partial updates
  - Test server-sent events for job status updates
  - Test search functionality with real-time filtering
  - Test toast notifications and error handling

### End-to-End Tests (Playwright)

- [ ] **T-E2E1**: Configure Playwright test suite
  - Set up Playwright with Chrome-only, headless configuration
  - Create test database isolation using in-memory SQLite
  - Configure mise tasks for running E2E tests
  - Set up CI/CD integration for automated testing

- [ ] **T-E2E2**: Core dashboard workflow tests
  - Job creation, editing, and deletion workflows
  - Authentication flow with API key validation
  - Search and filtering functionality
  - Mobile responsiveness and touch interactions
  - Error handling and user feedback scenarios

## Documentation Tasks

- [ ] **T-DOC1**: Update API documentation
  - Document new dashboard endpoints
  - Update OpenAPI specification
  - Add dashboard configuration examples

- [ ] **T-DOC2**: Create dashboard user guide
  - Getting started with dashboard
  - Feature overview and screenshots
  - Troubleshooting common issues

- [ ] **T-DOC3**: Update deployment documentation
  - Dashboard configuration options
  - Security considerations
  - Performance tuning guidelines

## Configuration and Deployment

- [ ] **T-CFG1**: Add dashboard configuration validation
  - Validate dashboard path conflicts
  - Validate refresh interval ranges
  - Add configuration migration support

- [ ] **T-CFG2**: Update Docker and deployment configs
  - Update Dockerfile for dashboard assets
  - Update docker-compose with dashboard examples
  - Update Kubernetes manifests if applicable

## Security and Performance

- [ ] **T-SEC1**: Security audit of dashboard
  - Review all user inputs for XSS vulnerabilities
  - Audit authentication and authorization
  - Test CSRF protection mechanisms

- [ ] **T-PERF1**: Performance optimization
  - Optimize database queries for dashboard
  - Add caching for frequently accessed data
  - Test dashboard performance under load

## Migration and Backward Compatibility

- [ ] **T-MIG1**: Ensure backward compatibility
  - Verify existing API endpoints unchanged
  - Test existing CLI functionality
  - Validate metrics endpoint performance

- [ ] **T-MIG2**: Configuration migration support
  - Handle configuration upgrades gracefully
  - Provide default values for new options
  - Document configuration changes

## Quality Gates & Acceptance Validation

### Pre-Implementation Validation

- [ ] **T-QG1**: Technical architecture review
  - Validate Gin integration approach with existing HTTP server
  - Confirm asset embedding strategy meets size constraints
  - Review authentication integration maintains security standards
  - Verify database integration approach (indexes only, no schema changes)

### Implementation Quality Gates

- [ ] **T-QG2**: Development milestone validation
  - Phase 1: Core dashboard functionality complete with 100% test coverage
  - Phase 2: Enhanced features and accessibility compliance validated
  - Phase 3: Full test suite passing including Playwright E2E tests
  - Phase 4: Documentation complete and deployment ready

### Final Acceptance Criteria

- [ ] **T-ACC1**: Core functionality acceptance
  - Dashboard accessible at configurable path, disabled by default
  - Complete job CRUD operations via Bootstrap forms
  - HTTP Basic Auth integration working with existing API keys
  - Real-time updates via HTMX without breaking JavaScript-disabled browsers

- [ ] **T-ACC2**: Performance and compatibility acceptance
  - Binary size increase <500KB with all embedded assets
  - Dashboard page load times <2 seconds
  - Zero performance impact on existing API/metrics when disabled
  - Mobile responsive with 44px+ touch targets

- [ ] **T-ACC3**: Testing and quality acceptance
  - 100% unit test coverage maintained across all new code
  - Integration tests cover all dashboard endpoints and workflows
  - Playwright E2E tests validate complete user journeys
  - All mise tasks functional for development workflow

## Phase 4: Documentation & Deployment

### Documentation Updates

- [ ] **T-DOC1**: Update project documentation
  - Update README.md with dashboard setup instructions
  - Add dashboard configuration examples to docs
  - Update API documentation with new dashboard endpoints
  - Create dashboard user guide with screenshots

- [ ] **T-DOC2**: Update deployment documentation
  - Update Docker configurations for dashboard assets
  - Add dashboard security considerations
  - Document performance impact and tuning options
  - Update configuration migration guide

### Configuration & Deployment

- [ ] **T-CFG1**: Configuration validation and migration
  - Add dashboard configuration validation rules
  - Ensure backward compatibility with existing configs
  - Add configuration conflict detection (path conflicts)
  - Test configuration upgrades and defaults

- [ ] **T-CFG2**: Production deployment preparation
  - Update build processes for asset embedding
  - Configure proper HTTP headers for static assets
  - Add monitoring and logging integration
  - Test dashboard in production-like environments

## Dependencies and Technical Requirements

### Required Dependencies

- **Gin Web Framework** - Lightweight Go HTTP web framework
- **Bootstrap 5** (~200KB) - CSS framework embedded in binary
- **HTMX** (~14KB) - JavaScript library embedded in binary
- **Go 1.16+** - Required for embed directive support

### Internal Architecture Dependencies

- **Existing HTTP Server** - Dashboard mounts as sub-router
- **JobStore & JobResultStore** - Data layer integration (no changes)
- **Admin API Key System** - Authentication reuse (no changes)
- **SQLite Database** - Performance indexes added (no schema changes)

### Development Tools Required

- **Playwright** - End-to-end testing framework (Chrome-only)
- **mise** - Task runner for development workflow
- **Go testing tools** - Unit and integration test coverage

## Implementation Notes & Constraints

### Critical Requirements

- **100% Test Coverage** - All code must maintain existing coverage standards
- **Backward Compatibility** - Dashboard disabled by default, zero impact when disabled
- **Single Binary** - All assets embedded, no external file dependencies
- **Performance** - Minimal impact on existing API and metrics endpoints
- **Security** - Reuse existing authentication without bypasses or modifications

### Technical Constraints

- **No Schema Changes** - Only performance indexes, maintain existing database structure
- **No Breaking Changes** - All existing APIs and CLI functionality unchanged
- **Asset Size Limits** - Total embedded assets <500KB for reasonable binary size
- **Memory Usage** - Dashboard features <20MB additional memory under normal load
- **Response Times** - Dashboard pages <2s load time, API calls <500ms

### Development Workflow

- **mise-first Development** - All operations via mise tasks, no manual steps
- **Chrome-only Testing** - Playwright tests optimized for single browser
- **Headless CI/CD** - All automated testing in headless mode
- **In-memory Test DB** - Isolated test execution with `:memory:` SQLite
