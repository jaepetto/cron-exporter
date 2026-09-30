## Why

The embedded dashboard has unresolved correctness and maintainability problems: an unimplemented status endpoint, untested CRUD and authentication paths, fragile SSE delivery, duplicated validation, and a frontend build that can embed stale generated assets. A clean-room React portal can adopt the proven architectural patterns observed in Dagu while preserving cronmetrics' existing operational model and avoiding Dagu's GPL-licensed implementation.

## What Changes

- Replace the Gin template and HTMX frontend with a React and TypeScript single-page application in phased, testable slices.
- Add a protected, same-origin dashboard JSON API that shares validation and persistence behavior with the existing public API.
- Change dashboard SSE messages into reliable invalidation events with explicit flushing, heartbeat, overflow recovery, and polling fallback.
- Generate the frontend API types from the OpenAPI contract.
- Embed content-hashed frontend assets in the Go binary and make clean frontend builds mandatory for local, CI, container, and release builds.
- Add Go contract tests, frontend unit tests, and cross-browser Playwright workflows before removing the legacy dashboard.
- Preserve the configurable dashboard path, Basic browser authentication, public bearer API, SQLite storage, CLI, Prometheus metrics, and single-binary deployment.
- Implement the new portal independently using permissively licensed upstream dependencies; do not copy Dagu source code, components, styles, icons, or assets.

## Impact

- Affected specs: `dashboard`, `configuration`, `http-server`
- Affected code: `pkg/dashboard`, `pkg/api`, `pkg/config`, `pkg/model`, `ui`, `docs/openapi.yaml`, test suites, mise tasks, and `Dockerfile`
- Compatibility: Existing `/api/job*`, CLI, metrics, database schema, and dashboard configuration remain compatible. Existing dashboard URLs remain stable during the phased cutover.
- Dependencies: Adds a Node-built React frontend and permissively licensed UI/data-fetching dependencies. The runtime remains a non-root static Go binary with embedded assets.