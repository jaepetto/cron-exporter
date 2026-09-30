# Cronmetrics Portal

The portal is an original React and TypeScript implementation built with Vite and embedded into the cronmetrics Go binary.

## Development

Run commands from the repository root:

```bash
mise run ui-install
mise run ui-check
mise run ui-test
mise run ui-build
mise run ui-e2e
```

`npm run gen:api` generates `src/api/schema.ts` from `../docs/openapi.yaml`. Do not edit the generated file directly. Production output is generated under `pkg/dashboard/web/` and ignored except for `placeholder.txt`; every Go production build regenerates it first.

The browser uses same-origin dashboard APIs and browser-managed HTTP Basic credentials. Do not place admin keys in source, HTML metadata, local storage, session storage, or logs.

## Clean-room Boundary

Dagu was inspected as an architectural reference for typed API access, embedded assets, cache revalidation, SSE notifications, and browser testing. No Dagu source code, components, styles, icons, generated files, or assets were copied. Dagu is GPL-3.0-or-later.

Direct portal dependencies were audited on 2026-09-29. Runtime and development dependencies declare MIT, ISC, Apache-2.0, or OFL-1.1 licenses.