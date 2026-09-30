## ADDED Requirements

### Requirement: Compatible dashboard configuration
The configuration system SHALL retain dashboard enablement, path, title, page size, authentication, SSE, and polling settings while the portal implementation changes.

#### Scenario: Existing dashboard configuration is loaded
- **WHEN** a configuration written for the existing dashboard is loaded
- **THEN** the React portal uses the same path, title, page size, authentication, SSE, and polling values without requiring migration

#### Scenario: Dashboard remains disabled by default
- **WHEN** no dashboard enablement setting is supplied
- **THEN** dashboard routes are not registered

### Requirement: Dashboard configuration validation
The configuration system SHALL reject unsafe dashboard paths and out-of-range pagination, SSE, timeout, heartbeat, and polling values before server startup.

#### Scenario: Dashboard path conflicts with API routes
- **WHEN** the configured dashboard path is `/api`, `/metrics`, `/health`, or another reserved server path
- **THEN** configuration loading fails with a specific validation error

#### Scenario: Dashboard limit is invalid
- **WHEN** a dashboard page size or SSE client limit is outside its documented range
- **THEN** configuration loading fails with a specific validation error

### Requirement: Environment configuration parity
Every supported dashboard configuration option SHALL be configurable through the existing `CRONMETRICS_` environment-variable mapping.

#### Scenario: Environment overrides dashboard path
- **WHEN** `CRONMETRICS_DASHBOARD_PATH` is set to a valid path
- **THEN** it overrides the file and default dashboard path
