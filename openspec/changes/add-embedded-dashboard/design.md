# Design Document: Embedded Dashboard

**Change ID**: `add-embedded-dashboard`
**Status**: Draft
**Created**: 2025-11-02

## Architecture Overview

The embedded dashboard will be implemented as a new package (`pkg/dashboard`) that integrates with the existing HTTP server infrastructure. It will provide a web-based interface for job monitoring and management without requiring external dependencies.

### System Context

```
┌─────────────────────────────────────────────────────────────┐
│                    Cron Exporter Server                    │
├─────────────────────────────────────────────────────────────┤
│  Existing Components                                        │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐        │
│  │   API       │  │   Metrics   │  │    CLI      │        │
│  │ Endpoints   │  │  Collector  │  │  Commands   │        │
│  └─────────────┘  └─────────────┘  └─────────────┘        │
├─────────────────────────────────────────────────────────────┤
│  New Component                                              │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │                  Dashboard                              │ │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐    │ │
│  │  │   HTTP      │  │  Templates  │  │   Assets    │    │ │
│  │  │  Handlers   │  │   Engine    │  │  (CSS/JS)   │    │ │
│  │  └─────────────┘  └─────────────┘  └─────────────┘    │ │
│  └─────────────────────────────────────────────────────────┘ │
├─────────────────────────────────────────────────────────────┤
│  Shared Infrastructure                                      │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐        │
│  │ HTTP Server │  │    Auth     │  │  Data Layer │        │
│  │    (mux)    │  │ Middleware  │  │ (JobStore)  │        │
│  └─────────────┘  └─────────────┘  └─────────────┘        │
└─────────────────────────────────────────────────────────────┘
```

## Component Design

### Dashboard Package Structure

```
pkg/dashboard/
├── handler.go          # HTTP request handlers
├── service.go          # Business logic and data transformation
├── templates.go        # HTML template management
├── assets.go           # Embedded CSS/JS assets
├── sse.go             # Server-Sent Events implementation
└── templates/
    ├── base.html       # Base layout template
    ├── dashboard.html  # Main dashboard page
    ├── jobs.html       # Job list page
    ├── job_form.html   # Job create/edit form
    └── job_detail.html # Job detail page
```

### Configuration Schema

```go
type DashboardConfig struct {
    Enabled         bool   `mapstructure:"enabled"`           // Default: false
    Path            string `mapstructure:"path"`              // Default: "/dashboard"
    Title           string `mapstructure:"title"`             // Default: "Cron Monitor"
    RefreshInterval int    `mapstructure:"refresh_interval"`  // Default: 5 seconds
    AuthRequired    bool   `mapstructure:"auth_required"`     // Default: true
    MaxJobsPerPage  int    `mapstructure:"max_jobs_per_page"` // Default: 50
}
```

### HTTP Routes

The dashboard will add the following routes to the existing HTTP server:

```
GET  /dashboard                    # Main dashboard (job overview)
GET  /dashboard/jobs               # Job list page
GET  /dashboard/jobs/new           # Job creation form
POST /dashboard/jobs               # Create new job
GET  /dashboard/jobs/{id}          # Job detail page
GET  /dashboard/jobs/{id}/edit     # Job edit form
PUT  /dashboard/jobs/{id}          # Update job
DELETE /dashboard/jobs/{id}        # Delete job
GET  /dashboard/jobs/{id}/history  # Job execution history
GET  /dashboard/api/jobs           # JSON API for job data
GET  /dashboard/api/stats          # JSON API for dashboard stats
GET  /dashboard/events             # Server-Sent Events stream
GET  /dashboard/assets/*           # Static assets (CSS/JS)
```

## Data Models

### Dashboard View Models

```go
// DashboardSummary represents the main dashboard overview data
type DashboardSummary struct {
    TotalJobs     int                    `json:"total_jobs"`
    ActiveJobs    int                    `json:"active_jobs"`
    FailedJobs    int                    `json:"failed_jobs"`
    MaintenanceJobs int                  `json:"maintenance_jobs"`
    RecentJobs    []JobSummary           `json:"recent_jobs"`
    Stats         map[string]interface{} `json:"stats"`
    LastUpdated   time.Time              `json:"last_updated"`
}

// JobSummary represents condensed job information for lists
type JobSummary struct {
    ID                int               `json:"id"`
    Name              string            `json:"name"`
    Host              string            `json:"host"`
    Status            string            `json:"status"`
    LastStatus        string            `json:"last_status"`
    LastReportedAt    time.Time         `json:"last_reported_at"`
    IsOverdue         bool              `json:"is_overdue"`
    NextExpectedAt    *time.Time        `json:"next_expected_at,omitempty"`
    Labels            map[string]string `json:"labels"`
}

// JobDetail extends JobSummary with additional detail information
type JobDetail struct {
    JobSummary
    AutomaticFailureThreshold int              `json:"automatic_failure_threshold"`
    CreatedAt                 time.Time        `json:"created_at"`
    UpdatedAt                 time.Time        `json:"updated_at"`
    RecentResults            []JobResultSummary `json:"recent_results"`
    StatusHistory            []StatusChange     `json:"status_history"`
}

// JobResultSummary represents condensed job result information
type JobResultSummary struct {
    Status    string    `json:"status"`
    Duration  int       `json:"duration"`
    Timestamp time.Time `json:"timestamp"`
    Output    string    `json:"output,omitempty"` // Truncated for display
}

// StatusChange represents job status transition events
type StatusChange struct {
    PreviousStatus string    `json:"previous_status"`
    NewStatus      string    `json:"new_status"`
    Reason         string    `json:"reason"`
    Timestamp      time.Time `json:"timestamp"`
}
```

## Implementation Details

### Template Engine

Use Go's built-in `html/template` package for server-side rendering:

```go
type TemplateManager struct {
    templates *template.Template
    config    *config.DashboardConfig
}

func (tm *TemplateManager) Render(w http.ResponseWriter, name string, data interface{}) error {
    return tm.templates.ExecuteTemplate(w, name, data)
}
```

**Template Features:**
- XSS protection via automatic HTML escaping
- Reusable components and partials
- Layout inheritance with base templates
- Custom template functions for formatting

### Asset Management

Embed static assets using Go 1.16+ `embed` package:

```go
//go:embed templates/*.html
var templateFiles embed.FS

//go:embed assets/styles.css assets/dashboard.js
var staticAssets embed.FS

func ServeAsset(w http.ResponseWriter, r *http.Request) {
    // Serve embedded assets with proper content types and caching headers
}
```

### Real-time Updates

Implement Server-Sent Events for live status updates:

```go
type SSEBroker struct {
    clients    map[chan []byte]bool
    newClients chan chan []byte
    defuncts   chan chan []byte
    messages   chan []byte
}

func (broker *SSEBroker) HandleSSE(w http.ResponseWriter, r *http.Request) {
    // Set SSE headers and establish connection
    // Register client for job status updates
}
```

**Update Strategy:**
- Broadcast job status changes to all connected clients
- Include only changed data to minimize bandwidth
- Fallback to periodic polling for browsers without SSE support

### Authentication Integration

Reuse existing authentication middleware:

```go
func (s *Server) withDashboardAuth(handler http.HandlerFunc) http.HandlerFunc {
    if !s.config.Dashboard.AuthRequired {
        return handler
    }
    return s.withAuth(handler) // Reuse existing auth middleware
}
```

### Error Handling

Implement consistent error handling across dashboard:

```go
type DashboardError struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
    Details string `json:"details,omitempty"`
}

func (h *Handler) handleError(w http.ResponseWriter, r *http.Request, err error) {
    // Log error with context
    // Return appropriate HTTP status
    // Render error template or JSON response based on Accept header
}
```

## Security Considerations

### Input Validation

- **Form Data**: Validate all form inputs against expected schemas
- **URL Parameters**: Sanitize and validate all URL parameters
- **File Uploads**: Not supported to avoid security risks

### XSS Protection

- **Template Escaping**: Use Go's automatic HTML escaping
- **Content Security Policy**: Add CSP headers to prevent XSS
- **Input Sanitization**: Sanitize user input before storage

### CSRF Protection

- **Same-Origin Policy**: Validate Origin and Referer headers
- **CSRF Tokens**: Implement CSRF tokens for state-changing operations
- **SameSite Cookies**: Use SameSite cookie attributes

### Authentication

- **API Key Validation**: Reuse existing admin API key validation
- **Session Management**: Use stateless authentication (API keys)
- **Rate Limiting**: Apply rate limiting to dashboard endpoints

## Performance Considerations

### Database Optimization

- **Query Efficiency**: Optimize queries for dashboard data retrieval
- **Connection Pooling**: Reuse existing database connection pooling
- **Indexing**: Ensure proper indexes for dashboard queries

### Caching Strategy

```go
type DashboardCache struct {
    summaryCache  *time.Timer
    jobListCache  map[string]interface{}
    cacheTTL      time.Duration
}
```

- **Summary Data**: Cache dashboard summary for 30 seconds
- **Job Lists**: Cache filtered job lists for 10 seconds
- **Static Assets**: Use HTTP caching headers for CSS/JS

### Memory Management

- **Template Caching**: Parse templates once at startup
- **Connection Limits**: Limit concurrent SSE connections
- **Data Pagination**: Paginate large datasets to control memory usage

## Testing Strategy

### Unit Tests

- **Handler Tests**: Test all HTTP handlers with various inputs
- **Service Tests**: Test business logic and data transformation
- **Template Tests**: Test template rendering with edge cases

### Integration Tests

- **End-to-End Workflows**: Test complete user workflows
- **Authentication Tests**: Test dashboard authentication integration
- **Performance Tests**: Test dashboard performance under load

### Browser Compatibility

- **Modern Browsers**: Full functionality in Chrome, Firefox, Safari, Edge
- **Graceful Degradation**: Basic functionality without JavaScript
- **Mobile Support**: Responsive design for mobile devices

## Deployment Considerations

### Configuration

- **Default Values**: Provide sensible defaults for all configuration
- **Environment Variables**: Support configuration via environment variables
- **Validation**: Validate dashboard configuration at startup

### Resource Requirements

- **Memory**: Additional ~10-20MB for templates and caching
- **CPU**: Minimal CPU overhead for template rendering
- **Storage**: Additional ~500KB for embedded assets

### Monitoring

- **Metrics**: Add dashboard-specific metrics to /metrics endpoint
- **Logging**: Log dashboard access and errors with structured logging
- **Health Checks**: Include dashboard health in existing health checks

## Migration Path

### Backward Compatibility

- **API Compatibility**: No changes to existing API endpoints
- **Configuration**: Dashboard disabled by default
- **CLI Compatibility**: No changes to existing CLI commands

### Rollout Strategy

1. **Phase 1**: Deploy with dashboard disabled by default
2. **Phase 2**: Enable dashboard in staging environments
3. **Phase 3**: Document and promote dashboard feature
4. **Phase 4**: Gather feedback and iterate based on usage

## Future Enhancements

### Advanced Features

- **Custom Themes**: Support for custom CSS themes
- **User Management**: Multi-user support with role-based access
- **API Explorer**: Interactive API documentation within dashboard
- **Mobile App**: Native mobile application using dashboard APIs

### Integration Options

- **Webhook Integration**: Trigger webhooks on job status changes
- **External Authentication**: Support for OAuth2, LDAP, etc.
- **Multi-tenant Support**: Support for multiple isolated job namespaces
