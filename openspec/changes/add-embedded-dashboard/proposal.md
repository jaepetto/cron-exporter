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

Add an optional embedded web dashboard that provides:

1. **Live Status Dashboard**: Real-time view of all jobs with status, last run times, and failure reasons
2. **Job Management Interface**: Full CRUD operations for jobs via web UI
3. **Historical View**: Recent job execution history and trends
4. **Maintenance Operations**: Easy job pause/resume functionality
5. **Configuration Management**: Web-based configuration of thresholds and labels

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
2. **Minimal External Dependencies**: Use Go standard library, existing dependencies, and embedded HTMX
3. **Minimal Resource Impact**: Lightweight implementation that doesn't affect core functionality
4. **Security Conscious**: Same authentication model as API endpoints
5. **Mobile Friendly**: Responsive design that works on various screen sizes
6. **API Compatible**: All dashboard operations use existing API endpoints

## High-Level Architecture

### Frontend
- **Technology**: Server-side rendered HTML with vanilla JavaScript
- **Styling**: Embedded CSS (no external frameworks)
- **Updates**: Server-Sent Events (SSE) for real-time status updates
- **Forms**: Standard HTML forms with JavaScript enhancement

### Backend Integration
- **Routing**: New `/dashboard/*` route prefix in existing HTTP server
- **Authentication**: Reuse existing admin API key authentication
- **Data Source**: Use same `JobStore` and `JobResultStore` as API
- **Templates**: Go `html/template` for server-side rendering

### Configuration
```yaml
dashboard:
  enabled: false          # Disabled by default
  path: "/dashboard"      # URL path prefix
  title: "Cron Monitor"   # Page title
  refresh_interval: 5     # Auto-refresh interval in seconds
  auth_required: true     # Require admin API key
```

## Implementation Approach

### Phase 1: Core Dashboard (MVP)
- Job status overview page
- Basic job management (view, create, edit, delete)
- Simple HTML templates with embedded CSS
- Admin authentication integration

### Phase 2: Enhanced Features
- Real-time updates with Server-Sent Events
- Job execution history view
- Responsive design improvements
- Maintenance mode toggle UI

### Phase 3: Advanced Features (Future)
- Job status charts/graphs
- Alert configuration UI
- Bulk operations
- Export functionality

## Risk Assessment

### Technical Risks
- **Bundle Size**: Additional HTML/CSS/JS increases binary size
  - *Mitigation*: Embed minimal, optimized assets
- **Security Surface**: New web interface increases attack surface
  - *Mitigation*: Reuse existing auth, follow secure coding practices
- **Maintenance Overhead**: Frontend code requires ongoing maintenance
  - *Mitigation*: Keep UI simple, use standard web technologies

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
- [ ] Dashboard reuses existing admin API key authentication
- [ ] Unauthenticated access returns appropriate HTTP 401 responses
- [ ] Authentication can be disabled via configuration for development
- [ ] All dashboard operations respect existing authorization model

### HTMX Integration & Interactivity

**AC-4: Dynamic Content Updates**
- [ ] HTMX library (~14KB) embedded in binary
- [ ] Form submissions use HTMX for inline validation and feedback
- [ ] Job list updates without full page reloads
- [ ] Search results appear in real-time as user types
- [ ] Status toggles provide immediate visual feedback

**AC-5: Real-time Features**
- [ ] Job status updates appear automatically via Server-Sent Events
- [ ] Status changes broadcast to all connected dashboard clients
- [ ] Connection failures gracefully fallback to periodic polling
- [ ] Real-time updates work with HTMX partial template rendering

### Theme System

**AC-6: Dark/Light Theme Support**
- [ ] Theme toggle button available in dashboard header
- [ ] Themes switch immediately without page reload
- [ ] User theme preference persists across browser sessions
- [ ] System theme detection and automatic theme selection
- [ ] All UI components (forms, tables, status indicators) support both themes

**AC-7: Theme Implementation**
- [ ] CSS custom properties used for theme variables
- [ ] Smooth transitions between theme switches
- [ ] Theme-aware status colors and indicators
- [ ] Accessibility maintained in both themes (contrast ratios)

### Search and Filtering

**AC-8: Multi-Criteria Search**
- [ ] Search supports filtering by host, name, status, and tags
- [ ] Search syntax supports "key:value" format (e.g., `host:server1`)
- [ ] Multiple criteria can be combined (e.g., `host:server1 status:active`)
- [ ] Search is case-insensitive and supports partial matches
- [ ] Search results update in real-time via HTMX

**AC-9: Search User Experience**
- [ ] Search input provides autocomplete suggestions
- [ ] Invalid search syntax shows helpful error messages
- [ ] Search history can be cleared or managed
- [ ] Search state persists during navigation within dashboard

### Lazy Loading and Performance

**AC-10: Job List Performance**
- [ ] Initial page load shows first 25 jobs
- [ ] Infinite scroll loads additional jobs progressively
- [ ] Loading states displayed during data fetching
- [ ] Performance remains acceptable with 1000+ jobs
- [ ] Lazy loading works without JavaScript (graceful degradation)

**AC-11: Responsive Design**
- [ ] Mobile-optimized layout with touch-friendly controls
- [ ] Tablet layout with appropriate spacing and navigation
- [ ] Desktop layout maximizes screen real estate efficiently
- [ ] All interactive elements meet minimum touch target sizes (44px)
- [ ] Text remains readable at all screen sizes

### Performance Requirements

**AC-12: Binary Size and Resource Usage**
- [ ] Total binary size increase remains under 500KB
- [ ] Dashboard adds less than 20MB memory usage under normal load
- [ ] Static assets served with appropriate caching headers
- [ ] CSS and JavaScript minified and optimized

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

### User Experience

**AC-18: Usability Goals**
- [ ] New users understand job status within 30 seconds of accessing dashboard
- [ ] Job creation/editing possible without consulting documentation
- [ ] Status information updates automatically without user intervention
- [ ] Error messages provide clear, actionable guidance

**AC-19: Accessibility**
- [ ] Keyboard navigation works for all interactive elements
- [ ] Screen reader compatibility maintained
- [ ] Color contrast ratios meet WCAG 2.1 AA standards
- [ ] Focus indicators visible and consistent

### Testing and Quality Assurance

**AC-20: Test Coverage**
- [ ] Unit test coverage remains at 100%
- [ ] Integration tests cover all dashboard endpoints
- [ ] End-to-end tests validate complete user workflows
- [ ] Cross-browser compatibility verified (Chrome, Firefox, Safari, Edge)

**AC-21: Error Handling**
- [ ] Network failures gracefully handled with user feedback
- [ ] Invalid configurations provide clear error messages
- [ ] Database connection issues don't crash dashboard
- [ ] Partial failures allow continued operation where possible

## Design Decisions

Based on requirements analysis, the following design decisions have been made:

1. **UI Framework**: **HTMX** - Provides dynamic interactivity with minimal JavaScript complexity
2. **Theming**: **Dark/Light Theme Toggle** - Support both themes with user preference persistence
3. **Pagination**: **Lazy Loading** - Progressive loading for large job lists to improve performance
4. **Filtering**: **Multi-Criteria Search** - Jobs searchable by host, name, status, and tags
5. **Integration**: **Standalone Dashboard** - No external integrations to maintain simplicity

## Dependencies

### Internal Dependencies
- Existing HTTP server infrastructure
- Current authentication system
- Job and JobResult data models
- Configuration system

### External Dependencies

- **HTMX** (~14KB) - Embedded JavaScript library for dynamic interactions
- No runtime external dependencies (HTMX embedded in binary)

## Timeline Estimate

- **Research & Design**: 2-3 days
- **Phase 1 Implementation**: 5-7 days
- **Testing & Documentation**: 2-3 days
- **Phase 2 Enhancement**: 3-5 days

**Total**: ~2-3 weeks for full implementation

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
