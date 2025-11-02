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
2. **No External Dependencies**: Use only Go standard library and existing dependencies
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

## Success Criteria

### Functional Requirements
- [ ] Dashboard accessible at configurable URL path
- [ ] All job CRUD operations available via web UI
- [ ] Real-time status updates without page refresh
- [ ] Responsive design works on desktop and mobile
- [ ] Same authentication model as existing API

### Non-Functional Requirements
- [ ] Dashboard adds <500KB to binary size
- [ ] Page load time <2 seconds on typical deployments
- [ ] No external JavaScript dependencies
- [ ] Compatible with existing configuration system
- [ ] Zero impact on core functionality when disabled

### User Experience Goals
- [ ] New users can understand job status within 30 seconds
- [ ] Job creation/editing possible without reading documentation
- [ ] Status information updates automatically without user action
- [ ] UI works without JavaScript (graceful degradation)

## Open Questions

1. **UI Framework**: Vanilla JS vs lightweight framework (Alpine.js, htmx)?
2. **Theming**: Support for dark/light themes or custom CSS?
3. **Pagination**: How to handle deployments with hundreds of jobs?
4. **Filtering**: What job filtering/searching capabilities are needed?
5. **Integration**: Should dashboard link to Prometheus/Grafana when available?

## Dependencies

### Internal Dependencies
- Existing HTTP server infrastructure
- Current authentication system
- Job and JobResult data models
- Configuration system

### External Dependencies
- None (design principle: no new external dependencies)

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
