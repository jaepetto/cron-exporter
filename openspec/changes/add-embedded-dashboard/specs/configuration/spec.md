# Configuration Specification Changes

**Change ID**: `add-embedded-dashboard`
**Spec**: Configuration System
**Status**: Draft

## ADDED Requirements

### Dashboard Configuration Schema

### Requirement: Dashboard configuration structure
The configuration system SHALL support dashboard-specific configuration options.

**Details:**
- Dashboard configuration MUST be part of the main configuration structure
- All dashboard options MUST have sensible default values
- Dashboard MUST be disabled by default for backward compatibility

#### Scenario: Default dashboard configuration
```yaml
# No dashboard configuration specified
```
**Expected**: Dashboard disabled, no dashboard routes registered

#### Scenario: Minimal dashboard configuration
```yaml
dashboard:
  enabled: true
```
**Expected**: Dashboard enabled with default values (path="/dashboard", auth_required=true, etc.)

#### Scenario: Complete dashboard configuration
```yaml
dashboard:
  enabled: true
  path: "/monitor"
  title: "Job Monitor"
  refresh_interval: 10
  auth_required: false
  lazy_load_size: 25
  default_theme: "dark"
  enable_search: true
```
**Expected**: Dashboard configured with all specified values including theming and search

### Configuration Validation

### Requirement: Dashboard configuration validation
The configuration system SHALL validate dashboard configuration options.

**Details:**
- Path conflicts with existing routes MUST be detected
- Refresh interval MUST be within acceptable range (1-300 seconds)
- Max jobs per page MUST be within acceptable range (1-500)

#### Scenario: Dashboard path conflicts with API
```yaml
dashboard:
  enabled: true
  path: "/api"
```
**Expected**: Configuration validation error, server startup fails

#### Scenario: Invalid refresh interval
```yaml
dashboard:
  enabled: true
  refresh_interval: 0
```
**Expected**: Configuration validation error with helpful message

#### Scenario: Invalid max jobs per page
```yaml
dashboard:
  enabled: true
  max_jobs_per_page: 1000
```
**Expected**: Configuration validation error, value clamped or rejected

### Environment Variable Support

### Requirement: Dashboard environment variable configuration
The configuration system SHALL support dashboard configuration via environment variables.

**Details:**
- All dashboard options MUST be configurable via environment variables
- Environment variable names MUST follow existing naming conventions
- Environment variables MUST override file-based configuration

#### Scenario: Enable dashboard via environment variable
```bash
export CRONMETRICS_DASHBOARD_ENABLED=true
export CRONMETRICS_DASHBOARD_PATH="/monitor"
```
**Expected**: Dashboard enabled at /monitor path regardless of config file

#### Scenario: Disable authentication via environment variable
```bash
export CRONMETRICS_DASHBOARD_AUTH_REQUIRED=false
```
**Expected**: Dashboard authentication disabled

## MODIFIED Requirements

### Configuration Loading

### Requirement: Configuration structure extension
The existing configuration loading SHALL support dashboard configuration section.

**Details:**
- Dashboard configuration MUST be loaded with other configuration options
- Configuration merging MUST include dashboard options
- Default values MUST be applied for missing dashboard options

#### Scenario: Configuration file with dashboard section
```yaml
server:
  port: 8080
dashboard:
  enabled: true
  path: "/dashboard"
database:
  path: "/tmp/cronmetrics.db"
```
**Expected**: All configuration sections loaded including dashboard

#### Scenario: Partial dashboard configuration
```yaml
dashboard:
  enabled: true
  title: "Custom Monitor"
```
**Expected**: Dashboard enabled with custom title, other options use defaults

### Configuration Validation Extension

### Requirement: Validation rule extension
The existing configuration validation SHALL include dashboard-specific validation rules.

**Details:**
- Dashboard path validation MUST prevent conflicts with existing routes
- Dashboard options MUST be validated for type and range
- Validation errors MUST provide clear error messages

#### Scenario: Valid complete configuration
```yaml
server:
  port: 8080
dashboard:
  enabled: true
  path: "/dashboard"
  refresh_interval: 5
api:
  path: "/api"
```
**Expected**: Configuration validation passes, server starts successfully

#### Scenario: Dashboard path too similar to existing route
```yaml
dashboard:
  enabled: true
  path: "/api/dashboard"
```
**Expected**: Configuration validation warning about potential confusion
