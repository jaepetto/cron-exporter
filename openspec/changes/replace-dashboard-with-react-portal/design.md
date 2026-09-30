## Context

The current dashboard combines Gin HTML templates, form handlers, HTMX fragments, custom JavaScript polling, and SSE broadcasting. It directly accesses `JobStore`, duplicates behavior from the public API, has no package-level tests, and leaves `GET /dashboard/api/jobs/:id/status` unimplemented. The browser uses HTTP Basic authentication while the public API uses bearer authentication, so calling the public API directly from a SPA would require exposing an admin key to JavaScript.

Dagu demonstrates a useful architecture: a typed React frontend, an OpenAPI-derived client, cache revalidation, embedded versioned assets, SSE notifications, and browser tests. Dagu is GPL-3.0-or-later, so only architectural observations may be reused.

## Goals / Non-Goals

**Goals:**
- Deliver tested job monitoring and CRUD workflows at the existing dashboard URL.
- Keep credentials managed by the browser rather than frontend storage.
- Use one Go application layer for public API and dashboard behavior.
- Recover deterministically from missed, slow, or disconnected SSE delivery.
- Produce a reproducible, cache-safe, single-binary build.
- Support desktop and mobile layouts, keyboard operation, and light/dark/system themes.

**Non-Goals:**
- Adding Dagu-specific DAG editing, terminals, workspaces, RBAC, artifacts, wiki, agents, or scheduler features.
- Changing the SQLite schema, CLI contract, Prometheus metric contract, or public job-result submission contract.
- Copying or adapting GPL-licensed Dagu implementation code or visual assets.

## Decisions

### Clean-room frontend

Create an original React 19 and TypeScript frontend under `ui/`. Use Vite rather than duplicating Dagu's Webpack configuration because Vite provides the required development and production asset boundary with less project-specific configuration. Use Tailwind and permissively licensed Radix primitives for accessible controls. Record and audit direct dependency licenses.

### Dashboard backend-for-frontend

Add JSON endpoints below `${dashboard.path}/api`. These routes remain behind the existing dashboard Basic-auth middleware, allowing browser requests to use same-origin credentials without placing admin keys in JavaScript, HTML configuration, local storage, or logs.

Extract shared job operations and validation from HTTP handlers. Both the bearer-authenticated public API and the dashboard API call this layer; the dashboard must not create a second definition of defaults, allowed statuses, uniqueness handling, or error mapping.

### Contract-first client

Document the dashboard API and event envelope in `docs/openapi.yaml`, then generate TypeScript types with `openapi-typescript`. A typed `openapi-fetch` client and SWR hooks provide bounded timeouts, structured error handling, cache revalidation, and polling fallback. Generated output is checked for drift in CI.

### SSE as invalidation

SSE messages identify what changed but are not authoritative state. Each event has an ID, type, affected job ID when applicable, and timestamp. The frontend revalidates relevant SWR keys. The server explicitly flushes frames, emits heartbeat frames, limits clients, and sends a reset event or closes the stream when a slow client overflows so reconnect triggers full revalidation. Configured polling covers unsupported or repeatedly disconnected clients.

### Embedded asset pipeline

The production UI emits content-hashed assets. Go embeds only production output. Hashed files receive immutable caching; the HTML shell and injected non-secret runtime configuration receive `no-cache`. SPA fallback applies only to valid browser routes beneath the dashboard prefix and never intercepts `/api`, `/events`, or `/assets`.

The Docker build uses a Node UI stage followed by the Go builder and existing scratch runtime. Mise tasks orchestrate generation, type checking, frontend tests, UI build, Go tests, and final binary build. A clean checkout cannot compile a release from stale CSS or bundles.

### Phased cutover

First lock down current behavior with regression tests and implement the dashboard API. Then deliver list, detail, mutation, and realtime workflows as independently tested frontend slices. Keep a temporary legacy route or configuration-controlled fallback until parity tests pass. Remove templates, HTMX, old JavaScript, duplicated handlers, and the fallback only after production-build browser verification.

## Risks / Trade-offs

- **Larger build toolchain:** Node becomes a build dependency. Mitigation: mise pins tools, lockfiles pin packages, Docker isolates the build, and runtime remains unchanged.
- **Temporary duplicate frontends:** The phased cutover briefly increases code size. Mitigation: define an explicit removal task and do not extend the legacy frontend.
- **Basic authentication UX:** Browser-native credential prompts remain. Mitigation: preserve current security behavior now; session authentication is a separate security proposal.
- **SSE loss under load:** Events may still be missed. Mitigation: events only invalidate caches, overflow forces reset/reconnect, and polling remains available.
- **GPL contamination:** Copying Dagu code could impose incompatible distribution obligations. Mitigation: clean-room implementation, dependency license audit, and no copied source or assets.

## Migration Plan

1. Establish passing Go test and build baselines.
2. Add regression tests and shared Go job operations.
3. Add and test the dashboard JSON API and SSE contract.
4. Build the React shell and read-only job workflows.
5. Add mutations and realtime behavior with unit and browser tests.
6. Embed production assets and integrate all build paths.
7. Switch the default portal after parity verification while retaining rollback.
8. Remove the legacy implementation and document the migration.

Rollback before step 8 selects the legacy portal without changing stored data or public APIs. After step 8, rollback uses the previous application binary because the database schema and external contracts remain compatible.

## Open Questions

- Resolved: The production JavaScript bundle is 128.49 kB gzip and CSS is 4.94 kB gzip. The Chromium 1000-job test verifies a 25-row bounded page and a job-list query below 2 seconds.
- Resolved: The rollback mechanism is the previous compatible binary. No database migration or public API break was introduced, so a runtime legacy-portal flag would have added temporary production complexity without improving rollback safety.
- Verification note: Browser unit coverage is 40.3% statements and source-attributed Go coverage remains below the repository's written 100% target. The repository did not meet or enforce that target before this change; behavior coverage is supplemented by 13 passing production Playwright workflows across Chromium, Firefox, and WebKit.
