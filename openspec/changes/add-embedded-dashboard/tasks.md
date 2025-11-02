# Implementation Tasks: Add Embedded Dashboard

**Change ID**: `add-embedded-dashboard`
**Status**: Draft

## Phase 1: Core Dashboard (MVP)

### Backend Infrastructure

- [ ] **T1.1**: Add dashboard configuration to `Config` struct
  - Add `DashboardConfig` struct with enabled, path, title, refresh_interval, auth_required fields
  - Update default configuration values
  - Add validation for dashboard configuration

- [ ] **T1.2**: Create dashboard package structure
  - Create `pkg/dashboard/` package
  - Create `handler.go` for HTTP handlers
  - Create `templates.go` for HTML template management
  - Create `assets.go` for embedded CSS/JS

- [ ] **T1.3**: Implement dashboard HTTP handlers
  - Dashboard home page handler (job status overview)
  - Job list API endpoint for AJAX calls
  - Job details page handler
  - Job create/edit form handlers
  - Job delete confirmation handler

- [ ] **T1.4**: Integrate dashboard routes into main server
  - Add dashboard route prefix to `pkg/api/server.go`
  - Apply authentication middleware to dashboard routes
  - Add dashboard configuration to server initialization

### Frontend Templates

- [ ] **T1.5**: Create base HTML templates with HTMX integration
  - Base layout template with navigation and theme support
  - Job status overview page template with lazy loading
  - Job list component template with search functionality
  - Job form template (create/edit) with HTMX validation
  - HTMX partial templates for dynamic updates
  - Error page templates

- [ ] **T1.6**: Implement theme-aware embedded CSS
  - CSS custom properties for light/dark themes
  - Responsive grid layout styles
  - Job status indicators (success/failure/maintenance colors)
  - Form styling and validation states
  - Mobile-friendly responsive breakpoints
  - Theme toggle switch styling

- [ ] **T1.7**: Add HTMX and search functionality
  - Embed HTMX library (~14KB) in assets
  - Multi-criteria search (host, name, status, tags)
  - Lazy loading with infinite scroll
  - Theme preference persistence
  - Form validation with inline feedback
  - Status badge real-time updates

### Data Integration

- [ ] **T1.8**: Create dashboard service layer
  - Job summary service (status counts, recent failures)
  - Job list service with filtering capabilities
  - Job detail service with execution history
  - Dashboard-specific data transformation

- [ ] **T1.9**: Implement authentication integration
  - Reuse existing admin API key authentication
  - Add dashboard-specific authentication middleware
  - Handle authentication errors in web UI

## Phase 2: Enhanced Features

### Real-time Updates

- [ ] **T2.1**: Implement Server-Sent Events (SSE)
  - SSE endpoint for job status updates
  - Client-side SSE handling in JavaScript
  - Fallback to polling for browsers without SSE support

- [ ] **T2.2**: Add job execution history view
  - Job execution history page template
  - Pagination for large result sets
  - Filter by date range and status

### Search and Filtering

- [ ] **T2.3**: Implement advanced search functionality
  - Multi-criteria search backend (host, name, status, tags)
  - Search result highlighting and ranking
  - Search history and saved filters
  - Real-time search with HTMX

- [ ] **T2.4**: Add lazy loading implementation
  - Infinite scroll with HTMX
  - Progressive job loading (25 jobs per batch)
  - Loading states and skeleton UI
  - Performance optimization for large datasets

### Theme System

- [ ] **T2.5**: Implement dark/light theme system
  - CSS custom properties for theme variables
  - Theme toggle component with HTMX
  - User preference persistence (localStorage)
  - System theme detection and auto-switching
  - Smooth theme transition animations

### UI/UX Improvements

- [ ] **T2.6**: Enhance responsive design
  - Optimize layout for mobile devices
  - Add touch-friendly interaction elements
  - Improve tablet layout and navigation

- [ ] **T2.7**: Add maintenance mode UI controls
  - Toggle switches for job maintenance mode with HTMX
  - Bulk maintenance operations
  - Maintenance status indicators

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

## Testing Tasks

### Unit Tests

- [ ] **T-UT1**: Dashboard handler unit tests
  - Test all HTTP handlers with various inputs
  - Mock dependencies (JobStore, JobResultStore)
  - Test authentication and authorization

- [ ] **T-UT2**: Dashboard service unit tests
  - Test job summary calculations
  - Test data transformation logic
  - Test error handling scenarios

- [ ] **T-UT3**: Template rendering tests
  - Test template rendering with various data
  - Test XSS protection in templates
  - Test responsive layout generation

### Integration Tests

- [ ] **T-IT1**: Dashboard API integration tests
  - Test complete job CRUD workflows via dashboard
  - Test authentication integration
  - Test error response handling

- [ ] **T-IT2**: Real-time update integration tests
  - Test SSE functionality
  - Test auto-refresh mechanisms
  - Test concurrent user scenarios

### End-to-End Tests

- [ ] **T-E2E1**: Complete dashboard workflow tests
  - Job creation to status monitoring workflow
  - Multi-user concurrent access scenarios
  - Dashboard performance under load

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

## Acceptance Criteria Validation

### Core Functionality Validation

- [ ] **T-ACC1**: Dashboard accessibility validation
  - Verify dashboard accessible at configurable URL path
  - Test dashboard enable/disable functionality
  - Validate configuration prevents path conflicts
  - Confirm backward compatibility (disabled by default)

- [ ] **T-ACC2**: Job management interface validation
  - Test complete CRUD operations via web interface
  - Validate form validation for all required fields
  - Test job editing preserves existing data
  - Verify deletion confirmation and feedback
  - Check job status display accuracy

- [ ] **T-ACC3**: Authentication integration validation
  - Test admin API key authentication reuse
  - Verify HTTP 401 responses for unauthenticated access
  - Test authentication disable configuration
  - Validate authorization for all dashboard operations

### HTMX and Interactivity Validation

- [ ] **T-ACC4**: HTMX integration validation
  - Verify HTMX library embedded in binary (~14KB)
  - Test form submissions with inline validation
  - Validate job list updates without page reloads
  - Test real-time search functionality
  - Verify status toggles provide immediate feedback

- [ ] **T-ACC5**: Real-time features validation
  - Test Server-Sent Events for status updates
  - Verify status changes broadcast to all clients
  - Test fallback to polling on connection failures
  - Validate HTMX partial template rendering

### Theme System Validation

- [ ] **T-ACC6**: Theme functionality validation
  - Test theme toggle in dashboard header
  - Verify immediate theme switching without reload
  - Test user preference persistence across sessions
  - Validate system theme detection
  - Check all UI components support both themes

- [ ] **T-ACC7**: Theme implementation validation
  - Verify CSS custom properties usage
  - Test smooth theme transition animations
  - Validate theme-aware status colors
  - Check accessibility compliance in both themes

### Search and Performance Validation

- [ ] **T-ACC8**: Search functionality validation
  - Test multi-criteria search (host, name, status, tags)
  - Verify "key:value" search syntax support
  - Test multiple criteria combination
  - Validate case-insensitive partial matching
  - Check real-time search via HTMX

- [ ] **T-ACC9**: Search user experience validation
  - Test autocomplete suggestions functionality
  - Verify helpful error messages for invalid syntax
  - Test search history management
  - Validate search state persistence

- [ ] **T-ACC10**: Lazy loading validation
  - Test initial 25 job display
  - Verify infinite scroll progressive loading
  - Check loading states during data fetch
  - Test performance with 1000+ jobs
  - Validate graceful degradation without JavaScript

### Performance and Responsiveness Validation

- [ ] **T-ACC11**: Responsive design validation
  - Test mobile layout with touch-friendly controls
  - Verify tablet layout spacing and navigation
  - Check desktop layout screen utilization
  - Validate minimum touch target sizes (44px)
  - Test text readability at all screen sizes

- [ ] **T-ACC12**: Performance metrics validation
  - Measure binary size increase (<500KB)
  - Test memory usage increase (<20MB normal load)
  - Verify appropriate caching headers for static assets
  - Check CSS and JavaScript minification

- [ ] **T-ACC13**: Response time validation
  - Test dashboard home page load (<2 seconds)
  - Verify job list API response times (<500ms)
  - Check search result response times (<300ms)
  - Test theme switch completion (<100ms)

### Compatibility and Security Validation

- [ ] **T-ACC14**: Backward compatibility validation
  - Verify all existing API endpoints unchanged
  - Test CLI functionality remains identical
  - Check metrics endpoint performance unaffected
  - Validate configuration file backward compatibility

- [ ] **T-ACC15**: Security validation
  - Test form input validation against expected schemas
  - Verify HTML template automatic escaping
  - Check URL parameter sanitization
  - Validate Content Security Policy headers
  - Test CSRF protection mechanisms

### User Experience Validation

- [ ] **T-ACC16**: Usability validation
  - Test 30-second job status comprehension for new users
  - Verify job creation/editing without documentation
  - Check automatic status information updates
  - Test error message clarity and actionability

- [ ] **T-ACC17**: Accessibility validation
  - Test keyboard navigation for all interactive elements
  - Verify screen reader compatibility
  - Check WCAG 2.1 AA color contrast ratios
  - Validate consistent focus indicators

### Quality Assurance Validation

- [ ] **T-ACC18**: Test coverage validation
  - Maintain 100% unit test coverage
  - Verify integration test coverage for all endpoints
  - Check end-to-end test coverage for user workflows
  - Test cross-browser compatibility (Chrome, Firefox, Safari, Edge)

- [ ] **T-ACC19**: Error handling validation
  - Test graceful network failure handling
  - Verify clear error messages for invalid configurations
  - Check dashboard stability during database connection issues
  - Test partial failure recovery mechanisms

- [ ] **T-ACC20**: Optional feature impact validation
  - Verify zero core functionality impact when dashboard disabled
  - Test server startup time remains unchanged
  - Check memory usage unchanged when disabled
  - Validate no performance regression to existing functionality

## Dependencies and Blockers

### Internal Dependencies
- No blocking dependencies on other components
- Dashboard reuses existing HTTP server, authentication, and data layers

### External Dependencies

- HTMX library (~14KB) - embedded in binary for dynamic interactions

### Potential Blockers
- Template complexity might require refactoring for maintainability
- Performance testing might reveal need for caching layer
- Security audit might require additional input validation

## Notes

- All tasks should maintain 100% test coverage requirement
- Dashboard must be disabled by default to maintain backward compatibility
- All new configuration options must have sensible defaults
- Consider graceful degradation for JavaScript-disabled browsers
- Performance impact on core functionality must be minimal
