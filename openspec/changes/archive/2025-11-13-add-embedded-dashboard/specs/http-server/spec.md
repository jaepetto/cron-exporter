# HTTP Server Specification Changes

**Change ID**: `add-embedded-dashboard`
**Spec**: HTTP Server
**Status**: Draft

## MODIFIED Requirements

### Route Registration

### Requirement: Optional dashboard route registration
The HTTP server SHALL support optional registration of dashboard routes.

**Details:**
- Dashboard routes MUST only be registered when dashboard.enabled=true
- Dashboard routes MUST use configurable path prefix (default: "/dashboard")
- Dashboard routes MUST NOT conflict with existing API routes
- Route registration MUST happen during server initialization

#### Scenario: Dashboard routes registered when enabled
```yaml
dashboard:
  enabled: true
  path: "/dashboard"
```
**Expected**: Dashboard routes registered at `/dashboard/*`, accessible via HTTP

#### Scenario: Dashboard routes not registered when disabled
```yaml
dashboard:
  enabled: false
```
**Expected**: No dashboard routes registered, `/dashboard/*` returns 404

#### Scenario: Dashboard path conflict validation
```yaml
dashboard:
  enabled: true
  path: "/api"
```
**Expected**: Configuration validation error during startup

### Middleware Integration

### Requirement: Dashboard authentication middleware

The HTTP server SHALL apply authentication middleware to dashboard routes.

**Details:**
- Dashboard routes MUST use existing authentication middleware when auth_required=true
- Authentication bypass MUST be available when auth_required=false
- Authentication errors MUST return appropriate HTTP status codes

#### Scenario: Dashboard authentication required
```yaml
dashboard:
  enabled: true
  auth_required: true
```
```http
GET /dashboard
```
**Expected**: HTTP 401 Unauthorized response without valid API key

#### Scenario: Dashboard authentication disabled
```yaml
dashboard:
  enabled: true
  auth_required: false
```
```http
GET /dashboard
```
**Expected**: Dashboard content returned without authentication

### Content Type Negotiation

### Requirement: Response format negotiation
Dashboard endpoints SHALL support both HTML and JSON responses based on Accept header.

**Details:**
- HTML responses MUST be returned for browser requests
- JSON responses MUST be returned for API requests
- Default response format MUST be HTML for dashboard endpoints

#### Scenario: Browser request for dashboard
```http
GET /dashboard
Accept: text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8
```
**Expected**: HTML response with dashboard page

#### Scenario: API request for dashboard data
```http
GET /dashboard/api/jobs
Accept: application/json
```
**Expected**: JSON response with job data

## ADDED Requirements

### Static Asset Serving

### Requirement: Embedded asset serving
The HTTP server SHALL serve embedded dashboard assets.

**Details:**
- CSS and JavaScript assets MUST be embedded in the binary
- Assets MUST be served with appropriate content types
- Caching headers MUST be set for static assets

#### Scenario: CSS asset request
```http
GET /dashboard/assets/styles.css
```
**Expected**: CSS content with Content-Type: text/css and caching headers

#### Scenario: JavaScript asset request
```http
GET /dashboard/assets/dashboard.js
```
**Expected**: JavaScript content with Content-Type: application/javascript and caching headers

### Server-Sent Events Support

### Requirement: SSE endpoint support
The HTTP server SHALL support Server-Sent Events for real-time updates.

**Details:**
- SSE endpoint MUST be available at dashboard path + "/events"
- SSE connections MUST be authenticated when auth_required=true
- Connection limits MUST be enforced to prevent resource exhaustion

#### Scenario: SSE connection establishment
```http
GET /dashboard/events
Accept: text/event-stream
Authorization: Bearer admin-api-key
```
**Expected**: SSE stream established with proper headers

#### Scenario: SSE connection without authentication
```yaml
dashboard:
  auth_required: true
```
```http
GET /dashboard/events
Accept: text/event-stream
```
**Expected**: HTTP 401 Unauthorized response
