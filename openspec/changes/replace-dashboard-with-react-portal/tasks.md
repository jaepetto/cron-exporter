## 1. Baseline and Contracts

- [x] 1.1 Run the mandatory pre-change `mise run test` baseline.
- [x] 1.2 Run `mise run test-all` and `mise run build`; record pre-existing failures.
- [x] 1.3 Add dashboard package and HTTP regression tests for routing, authentication, CRUD, validation, configurable paths, assets, and SSE lifecycle.
- [x] 1.4 Extract shared job validation and application operations used by public and dashboard APIs.
- [x] 1.5 Add protected, paginated dashboard JSON endpoints and structured error responses.
- [x] 1.6 Define and implement flushed SSE invalidation, heartbeat, overflow recovery, reconnect, and polling fallback behavior.
- [x] 1.7 Update the OpenAPI document and add contract tests for generated types.

## 2. Frontend Foundation

- [x] 2.1 Create the clean-room React 19, TypeScript, and Vite application under `ui/` with locked dependencies.
- [x] 2.2 Configure OpenAPI type generation, the typed fetch client, SWR, routing, runtime base-path configuration, and normalized errors.
- [x] 2.3 Build the original accessible component layer, responsive shell, theme support, and complete loading, empty, error, and offline states.
- [x] 2.4 Implement list, search, filtering, pagination, and SSE-driven revalidation.
- [x] 2.5 Implement job detail, create, edit, maintenance toggle, and confirmed delete workflows.
- [x] 2.6 Add Vitest and React Testing Library coverage for routes, hooks, validation, events, fallback polling, and components.
- [x] 2.7 Audit direct frontend dependency licenses and document the clean-room boundary.

## 3. Embedding and Cutover

- [x] 3.1 Embed content-hashed production assets and implement cache-safe shell, asset, API, event, and deep-link routing.
- [x] 3.2 Integrate frontend generation, type checking, linting, tests, and builds into mise and CI tasks.
- [x] 3.3 Add the Node builder stage to the Dockerfile while preserving the scratch, non-root runtime image.
- [x] 3.4 Add Playwright Chromium, Firefox, and WebKit workflows for auth, CRUD, search, realtime recovery, themes, accessibility, and mobile layouts.
- [x] 3.5 Verify rollback through the previous compatible binary, then switch the React portal to the default.
- [x] 3.6 Remove Gin templates, HTMX, legacy assets, duplicated handlers, and the temporary fallback after parity approval.

## 4. Documentation and Release Verification

- [x] 4.1 Update README, CONTRIBUTING, specs, OpenAPI, configuration examples, and CHANGELOG.
- [x] 4.2 Run Go, frontend, coverage, browser, security, clean-build, cross-build, and production-container verification.
- [x] 4.3 Verify desktop, tablet, and mobile visual states; 1000+ job behavior; CSP; console output; credential secrecy; bundle size; and query latency.
- [x] 4.4 Confirm every task is complete and validate the OpenSpec change with `--strict`.