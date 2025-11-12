# Dashboard API Specification

**Change ID**: `add-embedded-dashboard`
**Spec**: Dashboard Web Interface
**Status**: Draft

## ADDED Requirements

### Dashboard Configuration

### Requirement: Dashboard configuration support
The system SHALL support dashboard configuration through the existing configuration system.

**Details:**
- Dashboard configuration MUST be part of the main configuration file
- Dashboard MUST be disabled by default to maintain backward compatibility
- Configuration MUST include path, title, refresh interval, and authentication settings

#### Scenario: Dashboard disabled by default
```yaml
# Default configuration
dashboard:
  enabled: false
```
**Expected**: Dashboard routes are not registered, no dashboard functionality available

#### Scenario: Dashboard enabled with custom configuration
```yaml
dashboard:
  enabled: true
  path: "/monitor"
  title: "Job Monitor"
  refresh_interval: 10
  auth_required: true
  max_jobs_per_page: 25
```
**Expected**: Dashboard available at `/monitor` with specified configuration

### Web Interface Routes

### Requirement: Dashboard HTTP endpoints
The system SHALL provide web-based interface endpoints for job monitoring and management.

**Details:**
- All dashboard endpoints MUST be prefixed with configurable path (default: `/dashboard`)
- Endpoints MUST support both HTML (browser) and JSON (API) responses based on Accept header
- Authentication MUST be applied consistently across all dashboard endpoints

#### Scenario: Dashboard home page access
```http
GET /dashboard
Accept: text/html
Authorization: Bearer admin-api-key
```
**Expected**: HTML page with job status overview, recent jobs, and summary statistics

#### Scenario: Job list JSON API
```http
GET /dashboard/api/jobs?status=active&limit=50
Accept: application/json
Authorization: Bearer admin-api-key
```
**Expected**: JSON response with paginated job list matching filter criteria

#### Scenario: Job creation via web form
```http
POST /dashboard/jobs
Content-Type: application/x-www-form-urlencoded
Authorization: Bearer admin-api-key

name=backup-job&host=server1&automatic_failure_threshold=3600&labels=env:prod,type:backup
```
**Expected**: Job created, redirect to job detail page or return JSON response

### Real-time Updates

### Requirement: Live status updates
The system SHALL provide real-time job status updates to connected dashboard clients.

**Details:**
- Real-time updates MUST be implemented using Server-Sent Events (SSE)
- Updates MUST include only changed job data to minimize bandwidth
- Fallback polling mechanism MUST be available for browsers without SSE support

#### Scenario: SSE connection establishment
```http
GET /dashboard/events
Accept: text/event-stream
Authorization: Bearer admin-api-key
```
**Expected**: SSE stream established, periodic status updates sent to client

#### Scenario: Job status change notification
```
event: job-status-update
data: {"job_id": 123, "status": "failure", "last_reported_at": "2025-11-02T10:30:00Z"}
```
**Expected**: Connected clients receive job status update via SSE

### Authentication Integration

### Requirement: Dashboard authentication
The system SHALL integrate dashboard authentication with existing API authentication.

**Details:**
- Dashboard MUST reuse existing admin API key authentication mechanism
- Authentication requirement MUST be configurable (default: enabled)
- Unauthenticated access MUST return appropriate error responses

#### Scenario: Dashboard access with valid admin API key
```http
GET /dashboard
Authorization: Bearer valid-admin-api-key
```
**Expected**: Dashboard content rendered and returned

#### Scenario: Dashboard access without authentication (auth required)
```http
GET /dashboard
```
**Expected**: HTTP 401 Unauthorized response with authentication error

#### Scenario: Dashboard access without authentication (auth disabled)
```yaml
dashboard:
  enabled: true
  auth_required: false
```
```http
GET /dashboard
```
**Expected**: Dashboard content rendered and returned without authentication

### Job Management Interface

### Requirement: Web-based job CRUD operations
The system SHALL provide complete job management functionality through the web interface.

**Details:**
- All existing job CRUD operations MUST be available via web interface
- Form validation MUST match API validation requirements
- Success/error feedback MUST be provided to users

#### Scenario: Job creation form display
```http
GET /dashboard/jobs/new
```
**Expected**: HTML form with fields for job name, host, threshold, labels, and status

#### Scenario: Job editing with existing data
```http
GET /dashboard/jobs/123/edit
```
**Expected**: HTML form pre-populated with existing job data

#### Scenario: Invalid job data submission
```http
POST /dashboard/jobs
Content-Type: application/x-www-form-urlencoded

name=&host=server1&automatic_failure_threshold=-1
```
**Expected**: Form redisplayed with validation errors, no job created

### Responsive Design

### Requirement: Mobile-friendly interface
The dashboard SHALL provide responsive design that works on various screen sizes.

**Details:**
- Interface MUST be usable on desktop, tablet, and mobile devices
- Layout MUST adapt to different screen sizes without horizontal scrolling
- Touch-friendly controls MUST be provided for mobile devices

#### Scenario: Mobile browser access
```http
GET /dashboard
User-Agent: Mozilla/5.0 (iPhone; CPU iPhone OS 14_0 like Mac OS X)
```
**Expected**: Mobile-optimized layout with touch-friendly controls

#### Scenario: Tablet browser access
```http
GET /dashboard
User-Agent: Mozilla/5.0 (iPad; CPU OS 14_0 like Mac OS X)
```
**Expected**: Tablet-optimized layout with appropriate spacing and controls

### HTMX Integration

### Requirement: Dynamic content updates with HTMX
The dashboard SHALL use HTMX for dynamic content updates without full page reloads.

**Details:**
- HTMX library MUST be embedded in the binary (~14KB)
- Form submissions MUST use HTMX for inline validation and feedback
- Job status updates MUST use HTMX partial template rendering
- Search functionality MUST provide real-time results via HTMX

#### Scenario: HTMX job search
```http
GET /dashboard/jobs/search?q=host:server1
X-Requested-With: XMLHttpRequest
HX-Request: true
```
**Expected**: Partial HTML with filtered job list, no full page reload

#### Scenario: HTMX form submission
```http
POST /dashboard/jobs
Content-Type: application/x-www-form-urlencoded
HX-Request: true

name=test-job&host=server1&automatic_failure_threshold=3600
```
**Expected**: Form validation feedback via partial template, success state update

### Theme System

### Requirement: Dark/light theme support
The dashboard SHALL provide dark and light theme options with user preference persistence.

**Details:**
- Theme toggle MUST be available in dashboard header
- User preference MUST persist across browser sessions
- System theme detection MUST be supported
- All UI components MUST support both themes

#### Scenario: Theme toggle activation
```http
GET /dashboard/theme/toggle
HX-Request: true
```
**Expected**: Theme switched, preference saved, UI updated without page reload

#### Scenario: Theme preference persistence
```javascript
// localStorage check
localStorage.getItem('dashboard-theme') === 'dark'
```
**Expected**: User's theme preference restored on page load

### Lazy Loading and Search

### Requirement: Job list lazy loading
The dashboard SHALL implement lazy loading for job lists to handle large datasets efficiently.

**Details:**
- Initial load MUST show first 25 jobs
- Infinite scroll MUST load additional jobs progressively
- Loading states MUST be shown during data fetch
- Performance MUST remain acceptable with 1000+ jobs

#### Scenario: Initial job list load
```http
GET /dashboard/jobs
```
**Expected**: First 25 jobs displayed with lazy loading trigger at bottom

#### Scenario: Lazy load more jobs
```http
GET /dashboard/jobs/load-more?offset=25&limit=25
HX-Request: true
```
**Expected**: Next 25 jobs appended to existing list via HTMX

### Requirement: Multi-criteria job search
The dashboard SHALL provide comprehensive search functionality across job attributes.

**Details:**
- Search MUST support filtering by host, name, status, and tags
- Search syntax MUST support "key:value" format (e.g., "host:server1")
- Search results MUST update in real-time as user types
- Search MUST be case-insensitive and support partial matches

#### Scenario: Multi-criteria search
```http
GET /dashboard/jobs/search?q=host:server1 status:active tag:prod
HX-Request: true
```
**Expected**: Jobs matching all criteria returned via HTMX partial update

#### Scenario: Real-time search typing
```javascript
// User types "host:ser" in search box
```
**Expected**: Search results update automatically, showing hosts matching "ser"

## MODIFIED Requirements

### HTTP Server Configuration

### Requirement: HTTP server route registration (MODIFIED)
The existing HTTP server SHALL support optional dashboard route registration.

**Details:**
- Dashboard routes MUST only be registered when dashboard is enabled
- Dashboard routes MUST NOT conflict with existing API routes
- Route registration MUST happen during server initialization

#### Scenario: Server startup with dashboard enabled
```yaml
dashboard:
  enabled: true
  path: "/dashboard"
```
**Expected**: Dashboard routes registered at `/dashboard/*`, server starts successfully

#### Scenario: Server startup with dashboard disabled
```yaml
dashboard:
  enabled: false
```
**Expected**: No dashboard routes registered, existing functionality unchanged

### Configuration System

### Requirement: Configuration schema extension (MODIFIED)
The existing configuration system SHALL support dashboard-specific configuration options.

**Details:**
- Dashboard configuration MUST be optional with sensible defaults
- Configuration validation MUST include dashboard-specific validation
- Environment variable support MUST include dashboard options

#### Scenario: Configuration validation with dashboard options
```yaml
dashboard:
  enabled: true
  path: "/api"  # Conflicts with existing API path
```
**Expected**: Configuration validation error, server fails to start

## Security Considerations

### Input Validation

### Requirement: Dashboard input security
All dashboard user inputs SHALL be validated and sanitized to prevent security vulnerabilities.

**Details:**
- HTML template rendering MUST use automatic escaping
- Form inputs MUST be validated against expected schemas
- URL parameters MUST be sanitized before processing

#### Scenario: XSS attempt via job name
```http
POST /dashboard/jobs
Content-Type: application/x-www-form-urlencoded

name=<script>alert('xss')</script>&host=server1&automatic_failure_threshold=3600
```
**Expected**: Job name escaped in HTML output, no script execution

### CSRF Protection

### Requirement: CSRF attack prevention
The dashboard SHALL implement protection against Cross-Site Request Forgery attacks.

**Details:**
- State-changing operations MUST validate request origin
- CSRF tokens SHOULD be implemented for form submissions
- SameSite cookie attributes MUST be used where applicable

#### Scenario: Cross-origin job deletion attempt
```http
DELETE /dashboard/jobs/123
Origin: https://malicious-site.com
Authorization: Bearer admin-api-key
```
**Expected**: Request rejected due to invalid origin

### Visual Status Indicators

### Requirement: Job deadline status visualization
The dashboard SHALL provide visual indicators for job deadline status based on automatic failure thresholds.

**Details:**
- Visual indicators MUST use the same logic as the Prometheus metrics system
- Status calculation MUST be based on `AutomaticFailureThreshold` per job
- Indicators MUST be color-coded for quick visual identification
- Status MUST persist through real-time updates and page refreshes

#### Status Categories

**On Time (Success)**: Job reported within deadline
- **Color**: Green (#198754)
- **Condition**: `timeSinceLastReport <= AutomaticFailureThreshold`
- **Visual**: Green background with left border, green status icon

**Approaching Deadline (Warning)**: Job approaching failure threshold
- **Color**: Yellow (#ffc107)
- **Condition**: `timeSinceLastReport > (AutomaticFailureThreshold * 0.8)`
- **Visual**: Yellow background with left border, yellow status icon

**Deadline Missed (Danger)**: Job past automatic failure threshold
- **Color**: Red (#dc3545)
- **Condition**: `timeSinceLastReport > AutomaticFailureThreshold`
- **Visual**: Red background with left border, red status icon

**Inactive**: Job in maintenance or paused status
- **Color**: Gray (#6c757d)
- **Condition**: `job.Status == "maintenance" || job.Status == "paused"`
- **Visual**: Gray background with left border, gray status icon

#### Scenario: Job deadline status display
```go
// Job with 1-hour threshold, last reported 45 minutes ago
job := &Job{
    AutomaticFailureThreshold: 3600, // 1 hour
    LastReportedAt: time.Now().Add(-45 * time.Minute),
    Status: "active"
}
```
**Expected**: Display with green "On Time" indicator

#### Scenario: Job approaching deadline
```go
// Job with 1-hour threshold, last reported 50 minutes ago (83% of threshold)
job := &Job{
    AutomaticFailureThreshold: 3600, // 1 hour
    LastReportedAt: time.Now().Add(-50 * time.Minute),
    Status: "active"
}
```
**Expected**: Display with yellow "Deadline Approaching" indicator

#### Scenario: Job missed deadline
```go
// Job with 1-hour threshold, last reported 75 minutes ago
job := &Job{
    AutomaticFailureThreshold: 3600, // 1 hour
    LastReportedAt: time.Now().Add(-75 * time.Minute),
    Status: "active"
}
```
**Expected**: Display with red "Deadline Missed" indicator

#### Scenario: Job in maintenance mode
```go
// Job in maintenance status regardless of last report time
job := &Job{
    AutomaticFailureThreshold: 3600,
    LastReportedAt: time.Now().Add(-2 * time.Hour), // Would be failed
    Status: "maintenance"
}
```
**Expected**: Display with gray "Maintenance" indicator, not red

### Real-time Status Updates

### Requirement: Persistent visual indicators during updates
Visual status indicators SHALL remain consistent during background updates and refreshes.

**Details:**
- Background updates MUST use HTML partial templates with status calculations
- JavaScript polling MUST maintain visual indicator consistency
- SSE updates MUST preserve deadline status information
- Template functions MUST be used for all status calculations

#### Scenario: Background refresh maintaining status
```javascript
// Before: Manual page load shows red "Deadline Missed" indicator
// Background refresh occurs after 5 seconds
// After: Status indicator remains red "Deadline Missed"
```
**Expected**: Visual indicators persist through background updates

## Performance Requirements

### Response Time

### Requirement: Dashboard response performance
Dashboard endpoints SHALL meet performance requirements for web interface usability.

**Details:**
- Dashboard pages MUST load within 2 seconds under normal conditions
- API endpoints MUST respond within 500ms under normal conditions
- Static assets MUST be served with appropriate caching headers

#### Scenario: Dashboard home page load time
```http
GET /dashboard
```
**Expected**: Complete page load (including all assets) within 2 seconds

### Resource Usage

### Requirement: Dashboard resource efficiency
The dashboard SHALL have minimal impact on system resources when enabled.

**Details:**
- Binary size increase MUST be less than 500KB
- Memory usage increase MUST be less than 20MB under normal load
- Dashboard MUST NOT impact core functionality performance

#### Scenario: Binary size validation
```bash
# Before dashboard implementation
ls -la cronmetrics
# After dashboard implementation
ls -la cronmetrics
```
**Expected**: Binary size increase less than 500KB

## Compatibility Requirements

### Backward Compatibility

### Requirement: Existing functionality preservation
The dashboard implementation SHALL NOT break existing functionality.

**Details:**
- All existing API endpoints MUST remain unchanged
- CLI functionality MUST remain unchanged
- Metrics endpoint performance MUST remain unchanged
- Default configuration MUST maintain existing behavior

#### Scenario: Existing API endpoint functionality
```http
GET /api/job
Authorization: Bearer admin-api-key
```
**Expected**: Same response format and behavior as before dashboard implementation

#### Scenario: CLI command functionality
```bash
cronmetrics job list
```
**Expected**: Same output format and behavior as before dashboard implementation
