## ADDED Requirements

### Requirement: Optional portal route registration
The HTTP server SHALL register portal shell, asset, JSON API, and event routes only when the dashboard is enabled and SHALL mount them beneath the configured dashboard path.

#### Scenario: Dashboard is disabled
- **WHEN** the server starts with dashboard enablement set to false
- **THEN** no dashboard shell, asset, JSON API, or event route is available

#### Scenario: Custom dashboard path is configured
- **WHEN** the dashboard path is `/monitor`
- **THEN** shell, deep links, assets, JSON API, and events work beneath `/monitor` without hard-coded `/dashboard` references

### Requirement: Cache-safe embedded assets
The HTTP server SHALL embed production portal assets and apply cache headers based on asset mutability.

#### Scenario: Hashed asset is requested
- **WHEN** a client requests an existing content-hashed JavaScript, CSS, font, or image asset
- **THEN** the server returns the correct content type with immutable long-lived caching

#### Scenario: Portal shell is requested
- **WHEN** a client requests the portal shell or valid client route
- **THEN** the server returns the current shell with no-cache semantics and non-secret runtime configuration

### Requirement: Secure portal responses
The HTTP server SHALL apply dashboard authentication and security headers consistently to the portal shell, JSON API, and SSE endpoint without exposing credentials.

#### Scenario: Protected portal request lacks credentials
- **WHEN** dashboard authentication is enabled and a shell, JSON API, or SSE request lacks valid Basic credentials
- **THEN** the server returns HTTP 401 with a Basic authentication challenge

#### Scenario: Portal response is successful
- **WHEN** an authenticated portal response is returned
- **THEN** it includes the configured content security, frame, content-type, and referrer protections

### Requirement: Reliable SSE transport
The HTTP server SHALL explicitly flush SSE events and heartbeats, enforce connection limits, release disconnected clients, and provide deterministic overflow recovery.

#### Scenario: Event is written
- **WHEN** the server sends an SSE event or heartbeat
- **THEN** the complete frame is flushed to the connected client without waiting for the response buffer to fill

#### Scenario: Client limit is reached
- **WHEN** a new SSE connection would exceed the configured maximum
- **THEN** the server rejects it with HTTP 503 and does not leak a client registration

#### Scenario: Slow client overflows
- **WHEN** a client cannot consume its buffered events
- **THEN** the server sends a reset indication when possible or closes the stream so the client reconnects and revalidates authoritative data

### Requirement: Reproducible single-binary build
The build system SHALL generate and validate frontend contracts and assets before compiling the static Go binary and production container.

#### Scenario: Production build starts from a clean checkout
- **WHEN** the production build runs without previously generated frontend output
- **THEN** it installs locked dependencies, checks generated API types, tests and builds the portal, embeds the output, and produces the static binary

#### Scenario: Production container starts
- **WHEN** the final container image runs
- **THEN** it remains a non-root scratch image serving the embedded portal without Node or external asset files at runtime