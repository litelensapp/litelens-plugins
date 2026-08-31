---
name: development-workflow
description: "Build, test, and lint commands for litelens-plugins (run from repo root) — helm frontend + backend, plus local install mirroring"
metadata:
  node_type: memory
  type: reference
  originSessionId: 7d5e9c83-be5e-4cf5-9fd6-36ce1356d5fc
  modified: 2026-08-27T13:08:10.944Z
---

All commands below run from the repo root unless noted.

Install:
- `pnpm install` — install JS workspace deps.

Frontend (helm plugin, `plugins/helm/frontend`):
- `pnpm build:helm:fe` — build the bundle (tailwind + tsup).
- `pnpm test:helm:fe` — vitest run.
- `pnpm test:helm:fe:coverage` — vitest --coverage.
- `pnpm lint:fe` — eslint `plugins/helm/frontend/src`.
- Single test file: `cd plugins/helm/frontend && pnpm vitest run src/components/release/__tests__/HelmReleaseStatusBadge.test.tsx`.

Backend (helm plugin, `plugins/helm`):
- `pnpm test:helm:be` — `go test -race -v ./...` inside `plugins/helm/`.
- `pnpm test:helm:be:coverage` — adds `-cover -coverprofile=coverage.out`.
- `pnpm lint:be` — `go vet` + `staticcheck` (invoked as `go tool staticcheck ./...`, not a separate install).
- Single Go test: `cd plugins/helm && go test -race -run TestHelmAge ./internal/applications/helm/...`
  (package moved here under the hexagonal-architecture refactor — see [[file-structure]]).
- Build the plugin binary: `cd plugins/helm && GOOS=linux GOARCH=amd64 VERSION=1.2.3 ./scripts/build.sh`
  (same script used by local dev and CI).

Local install mirroring:
- `cd plugins/helm && node scripts/deploy-plugin-helm-local.mjs` (run `build:helm:fe` first) — mirrors
  a full plugin install (frontend dist + binary + tar.gz + metadata) under `<repo-root>/.output/helm/`;
  never touches the real `~/.litelens/plugins/helm`; intentionally produces no `helm.lock` (that's
  host-runtime-only state). See [[file-structure]] for what these paths map to.

`@litelens/core` and `@litelens/design-system` (npm) and `github.com/litelensapp/litelens/packages/core`
(Go) are consumed as ordinary published/tagged dependencies from the `litelens` host repo — no local
`file:`/`replace` linking is needed or expected in this repo anymore.
