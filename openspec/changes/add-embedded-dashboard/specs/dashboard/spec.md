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
