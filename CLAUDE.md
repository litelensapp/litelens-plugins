# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

Official plugin repository for Litelens (a Kubernetes desktop app built with Wails). Each plugin is
a self-contained subdirectory pairing a Go gRPC subprocess (backend) with a dynamically-loaded
TypeScript/React ES module (frontend). Currently the repo holds one plugin: `plugins/helm/`.

This is a pnpm workspace (`pnpm-workspace.yaml` includes `plugins/helm/frontend`) at the JS layer, and a
single Go module (`github.com/litelensapp/litelens-plugins`) at the Go layer — `plugins/helm/` has its own
nested `go.mod` (`.../litelens-plugins/plugins/helm`) since it depends on heavy packages (`helm.sh/helm/v3`,
`k8s.io/client-go`) that shouldn't pollute the root module. `staticcheck` is declared as a `tool`
directive in that `go.mod` (Go 1.26+ tool dependency) and invoked via `go tool staticcheck ./...`,
not installed separately.

## Commands

Build/test/lint commands (pnpm scripts for the helm plugin, single-test invocations, the
`deploy-plugin-helm-local.mjs` mirror script, and the `@litelens/design-system` workspace-dependency
caveat) are catalogued in `.claude/memory/development_workflow.md` rather than duplicated here.

## Architecture

The business/data-plane call path is **plugin frontend → plugin backend directly over localhost
HTTP**, bypassing the host for every business call (`listCharts`, `installChart`, etc.). The host
still owns process lifecycle (spawn, handshake parsing, crash detection) and cluster-context
propagation, since those are inherent to running plugins as separate OS processes — see
`litelens/.claude/plans/plugin-architecture-inversion.md` (host repo) for the full design rationale.
A gRPC connection still exists, but only as a thin control channel in the _opposite_ direction from
the old model: the plugin dials the host and subscribes to generic pub/sub topics (`cluster.context`,
`namespaces.active`), rather than the host driving every plugin call.

The Go backend (`plugins/helm/internal/`) follows hexagonal architecture:
`internal/applications/` (business logic + `port` interfaces, framework-agnostic) is driven by
`internal/adapters/presentations/rest/` (inbound HTTP) and drives
`internal/adapters/infrastructures/{app,kube,restconfig}/` (outbound gRPC pub/sub client, k8s cluster
provider, Helm REST-config shim).

Backend and frontend (`plugins/helm/frontend/src/`) directory-level detail — handshake/gRPC-watch
wiring in `main.go`, the `applications/lock.LockedService` locking strategy, HTTP route/error
conventions in `internal/adapters/presentations/rest/`, the `pb`/`packages/core` dependency, the ESM
bundle/`external` setup, the `wailsBridge.ts` fetch-retry contract, `@litelens/core`
registration-based plugin contract, the CSS build pipeline, and the local install-mirroring script —
is catalogued in `.claude/memory/file_structure.md` and `.claude/memory/architecture_call_path.md`
rather than duplicated here.

## Conventions

- Go backend and TS frontend payload shapes must be kept in sync by hand across three places: the
  route + handler in `internal/adapters/presentations/rest/handlers.go`, the corresponding export in
  `frontend/src/api/wailsBridge.ts`, and the type in `frontend/src/api/resources.ts`.
- Frontend tests live under `__tests__/` alongside the components they cover, using vitest +
  `@testing-library/react`.
- Go tests are colocated as `*_test.go` in the same package.
