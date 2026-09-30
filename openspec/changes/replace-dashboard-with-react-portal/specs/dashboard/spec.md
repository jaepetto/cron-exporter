## ADDED Requirements

### Requirement: React dashboard portal
The system SHALL provide an embedded React and TypeScript portal for monitoring and managing cron jobs at the configured dashboard path.

#### Scenario: Dashboard portal is opened
- **WHEN** an authenticated user requests the configured dashboard path
- **THEN** the server returns the portal shell and the browser loads its embedded content-hashed assets

#### Scenario: Dashboard deep link is reloaded
- **WHEN** an authenticated user reloads a valid job list, detail, create, or edit route beneath the configured dashboard path
- **THEN** the server returns the portal shell and client routing restores the requested view

### Requirement: Dashboard job workflows
The portal SHALL provide list, search, filter, pagination, detail, create, edit, maintenance toggle, and delete workflows using the same validation and persistence behavior as the public API.

#### Scenario: User searches a large job collection
- **WHEN** the user supplies search or status criteria for a collection containing at least 1000 jobs
- **THEN** the portal requests a bounded result page and displays matching jobs with usable pagination controls

#### Scenario: User submits invalid job data
- **WHEN** the user submits a create or edit form containing invalid fields
- **THEN** no mutation occurs and the portal displays structured field or form errors returned by the server

#### Scenario: User deletes a job
- **WHEN** the user confirms deletion of an existing job
- **THEN** the job is deleted, the list is revalidated, and the portal navigates to a valid route

### Requirement: Protected dashboard JSON API
The system SHALL expose same-origin JSON endpoints beneath the configured dashboard path for portal queries and mutations, protected by the configured dashboard authentication policy.

#### Scenario: Authenticated API request
- **WHEN** a browser with valid dashboard Basic credentials requests a dashboard JSON endpoint
- **THEN** the endpoint processes the request without exposing the admin API key to frontend code or storage

#### Scenario: Unauthenticated API request
- **WHEN** dashboard authentication is required and a request lacks valid Basic credentials
- **THEN** the endpoint returns HTTP 401 without job data

### Requirement: Live dashboard invalidation
The system SHALL notify connected portal clients of job changes using Server-Sent Events and SHALL provide polling as a recovery path.

#### Scenario: Job changes while connected
- **WHEN** a job is created, updated, toggled, deleted, or reports a changed status
- **THEN** connected clients receive an invalidation event and refetch authoritative affected data

#### Scenario: Client misses events
- **WHEN** an event buffer overflows or an SSE connection disconnects
- **THEN** the client reconnects or polls and performs full relevant cache revalidation

#### Scenario: SSE is disabled
- **WHEN** SSE is disabled and polling fallback is enabled
- **THEN** the portal refreshes authoritative data at the configured polling interval

### Requirement: Responsive and accessible portal
The portal SHALL remain usable on supported desktop, tablet, and mobile viewports with keyboard-accessible controls and light, dark, and system themes.

#### Scenario: Portal is used on a mobile viewport
- **WHEN** the viewport cannot display the desktop job table without overflow
- **THEN** the portal presents an adapted layout with readable content and touch-usable controls without horizontal page scrolling

#### Scenario: Portal is operated by keyboard
- **WHEN** a user navigates and activates interactive controls without a pointing device
- **THEN** focus order, visible focus, labels, dialogs, forms, and actions remain operable

### Requirement: Dashboard frontend test coverage
The dashboard SHALL have automated contract, component, and browser tests covering supported workflows and failure states.

#### Scenario: Portal change enters CI
- **WHEN** portal frontend or backend code changes
- **THEN** Go tests, type checking, frontend unit tests, production build, and configured Playwright browser tests run before acceptance
