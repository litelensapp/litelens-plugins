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

Run from repo root unless noted:

```bash
pnpm install                # install JS workspace deps

pnpm build:helm:fe          # build the helm frontend plugin bundle (tailwind + tsup)
pnpm test:helm:fe           # run helm frontend tests (vitest) — requires @litelens/design-system
pnpm test:helm:be           # go test -race -v ./... inside plugins/helm/

pnpm lint:fe                # eslint plugins/helm/frontend/src
pnpm lint:be                # go vet + staticcheck (go tool) inside plugins/helm/

# Single Go test (run inside plugins/helm/)
cd plugins/helm && go test -race -run TestHelmAge ./internal/helm/...

# Single frontend test file (run inside plugins/helm/frontend)
cd plugins/helm/frontend && pnpm vitest run src/components/release/__tests__/HelmReleaseStatusBadge.test.tsx

# Build the Go plugin binary (used by both local dev and CI; see plugins/helm/scripts/build.sh)
cd plugins/helm && GOOS=linux GOARCH=amd64 VERSION=1.2.3 ./scripts/build.sh

# Mirror a full local plugin install (frontend dist + binary + tar.gz + metadata) under plugins/helm/.output/
cd plugins/helm && node scripts/deploy-plugin-helm-local.mjs   # run build:helm:fe first
```

**Frontend depends on `@litelens/design-system`**, which lives in the main litelens app repo, not
here. Without workspace access to it, `test:helm:fe` and `build:helm:fe` won't resolve. Ask the user
how their workspace links it (local `pnpm link`, adjusted `pnpm-workspace.yaml`, etc.) before
assuming it's broken.

## Architecture

The business/data-plane call path is **plugin frontend → plugin backend directly over localhost
HTTP**, bypassing the host for every business call (`listCharts`, `installChart`, etc.). The host
still owns process lifecycle (spawn, handshake parsing, crash detection) and cluster-context
propagation, since those are inherent to running plugins as separate OS processes — see
`litelens/.claude/plans/plugin-architecture-inversion.md` (host repo) for the full design rationale.
A gRPC connection still exists, but only as a thin control channel in the _opposite_ direction from
the old model: the plugin dials the host and watches a stream, rather than the host driving every
plugin call.

### Backend (`plugins/helm/internal/`) — HTTP + gRPC-client plugin subprocess

- `internal/main.go` — process entrypoint. Builds a `DynamicClusterProvider` from `-kubeconfig`,
  starts an `internal/api/rest.HttpServer` on `-listen` (default `127.0.0.1:0`, auto-assigned port),
  then emits **exactly one** JSON handshake line (`{"type":"READY","httpPort":...,"pid":...}`) to
  stdout before serving — see `rest.HttpServer.Serve`. The host app parses this line to discover the
  subprocess's port. Never add other stdout writes before the handshake, and never change its shape
  without updating the host app in lockstep.
  - It also opens a gRPC _client_ connection back to the host (`internal/api/grpc.DialAndSubscribe`,
    addr resolved via `config.GetHostGRPCPort()`) and runs `kube.WatchClusterContext` in a goroutine,
    which reconnects with backoff (`internal/kube`'s `BackoffReconnector`) and applies incoming
    context switches to the `DynamicClusterProvider`. This is the plugin _pulling_ cluster-context
    changes from the host over a stream, not the host pushing an HTTP request to the plugin.
- `internal/helm/` — business logic (`Service`, in `helm.go`/`helm_chart.go`/`helm_release.go`).
  Talks to the Helm SDK (`helm.sh/helm/v3`) and to Kubernetes via the `ClusterProvider` interface.
  `internal/helm/lock.go`'s `LockedService` wraps every business method in `RLock`/`RUnlock` and
  `SetActiveContext` in `Lock`/`Unlock` (`sync.RWMutex`) so a cluster-context swap can't race an
  in-flight business call — verified race-free (`go test -race`) including concurrent business +
  context-switch requests. `main.go` always constructs `helm.NewLockedService(...)`; there is no
  unlocked path in production.
  - `plugins/helm/internal/helm/rest/rest.go` — a `genericclioptions.RESTClientGetter` shim wired to
    a live `rest.Config` instead of re-reading kubeconfig from disk, so Helm SDK actions honor the
    currently active cluster.
- `internal/api/rest/` — the HTTP business API. `server.go`'s `NewHttpServer` binds the listener and
  wires `Handler.RegisterRoutes` (`handlers.go`) onto a `http.ServeMux`, wrapped in `corsMiddleware`
  (the Wails webview origin is cross-origin from this loopback server's perspective, so CORS
  preflight has to be answered). Routes are `POST /api/helm/<camelCaseMethod>`, one per `Service`
  method (e.g. `POST /api/helm/listCharts`). **Any new backend capability needs a new route added to
  `RegisterRoutes` plus a handler method** — there is no reflection-based routing.
  - `response.go` defines the error contract every handler must use: `writeError(w, statusCode, code,
message)` writes `{code, message}` as `ErrorResponse` JSON with a non-2xx status; `writeJSON`
    writes a 200 with the typed result. Codes in use: `PLUGIN_UNAVAILABLE` (503, business-logic
    failure — invalid/stale cluster state etc.), `INVALID_REQUEST` (400, bad JSON body),
    `NOT_FOUND` (404, nil lookup), `INTERNAL_ERROR` (500, e.g. `setClusterContext` failure). The
    frontend's `wailsBridge.ts` throws this exact `{code, message}` shape on non-2xx responses — keep
    new handlers consistent with it.
  - `POST /internal/setClusterContext` — despite the historical name suggesting an inbound push, the
    active control path today is the outbound gRPC watch stream above; this HTTP handler is kept as
    an idempotent, localhost-only, same-mutex-guarded entry point.
- `internal/api/grpc/` — the gRPC _client_ side only (`client.go`'s `DialAndSubscribe`, `emitter.go`).
  There is no gRPC _server_ in the plugin anymore, and no `Invoke`/`dispatch()` method-relay switch —
  that generic RPC has been fully replaced by the HTTP routes above. `HostEventEmitter` uses the same
  connection to push async progress events (`helm:install:complete`, etc., see
  `hooks/async-events/useRegisterHelmEvents.ts`) to the host for the frontend's event bridge to
  consume — this is a different, unrelated use of the gRPC connection from the context-watch stream.
  - The gRPC wire contract (`plugin.proto` + generated `plugin.pb.go` / `plugin_grpc.pb.go`) is no
    longer hand-copied here — code imports `pb` from `github.com/litelensapp/litelens/packages/core/pb`,
    the same generated code the host app consumes. Editing the contract means editing
    `packages/core/pb/plugin.proto` in the `litelens` host repo and bumping this plugin's
    `packages/core` dependency version; there is no local `.proto` copy to keep in sync anymore.
- **`github.com/litelensapp/litelens/packages/core` dependency**: `plugins/helm/go.mod` requires
  `packages/core v0.1.0`. No published git tag exists yet for that module, so the root `go.work`
  (`use (. ./plugins/helm)` plus a path to a sibling `litelens` checkout) substitutes the on-disk
  module for the versioned dependency in the meantime — see
  `.claude/memory/go_work_removal_todo.md` for the exact removal steps once a real
  `packages/core/vX.Y.Z` tag is published. `packages/core` supplies the gRPC `pb` package and
  `kube.LoadingRules`; it does **not** supply Helm-specific types — `internal/dto/helm.go` (this repo)
  holds those and is plugin-owned, not a synced copy of anything in the host repo.

### Frontend (`plugins/helm/frontend/src/`) — dynamically-loaded ES module

- Builds to a standalone ESM bundle (`tsup.config.ts`) loaded by the host via runtime `import()` —
  it is **not** part of the host app's own Vite build, so it can't use the host's `@wailsjs` alias
  or bundle its own copy of `react`/`react-dom`/`@tanstack/react-query`/`@litelens/design-system`/
  `@litelens/core`. Those are all in `external` in `tsup.config.ts` and resolve at runtime through
  the host's import map (`frontend/public/vendor/*.js` in the main app) — this is required so shared
  React/context instances line up when a plugin component mounts inline in the host's fiber tree.
  When adding a new dependency to the frontend, check whether it needs to go in `external` too.
- `src/index.ts` is the plugin's barrel/contract file — the host discovers plugin capabilities only
  through these named exports (`PluginView`, `PLUGIN_NAV_ENTRY`, `PluginEventBridge`,
  `PLUGIN_STYLES`, `PLUGIN_TRAY_FAMILIES`). Internal components keep Helm-specific names; only the
  barrel re-exports under the generic contract names the host expects.
  `PLUGIN_TRAY_FAMILIES` is a runtime-keyed map (e.g. `"helm-chart"`, `"helm-chart-upgrade"`) — the
  host calls `unifiedTray.openTab(family, params)` with these keys and has no static knowledge of
  which families exist or what params they take, so family names are effectively part of the wire
  contract between this plugin and the host's tray system.
- `src/api/wailsBridge.ts` — every exported function calls a shared `fetchWithRetry(method, payload)`
  helper, which `fetch()`s `http://<backendAddr>/api/helm/<camelCaseMethod>` directly (no more Wails
  `InvokePlugin` relay). `method` URL segments and payload field names here **must match** the
  `RegisterRoutes` table in `plugins/helm/internal/api/rest/handlers.go` exactly (URL segments are
  camelCase; payload field names stay Go PascalCase, since request bodies unmarshal directly into
  anonymous Go structs). Adding a backend method means adding both a `RegisterRoutes` route/handler
  and a matching `wailsBridge.ts` export.
  - **Backend-address retry contract**: the backend's `127.0.0.1:<port>` address is fetched once via
    the Wails-bound `GetPluginBackendAddr("helm")` and cached module-level (with promise-dedup via
    `getBackendAddr()`). On a `fetch()` failure, `fetchWithRetry` distinguishes a thrown `TypeError`
    (genuine network/connection failure — the cached address is stale) from an already-parsed
    `{code, message}` error body (a real backend response, not a transport failure): only the former
    triggers `invalidateBackendAddrCache()` + one refetch + one retry; a second failure throws the
    typed `PLUGIN_UNAVAILABLE` error. A `{code, message}` error response is thrown as-is, no retry.
    Plugin authors implementing new backend calls should route them through `fetchWithRetry`, not a
    bare `fetch()`, to get this behavior automatically.
  - `hooks/async-events/usePluginBackendRestarted.tsx` subscribes to the host's
    `plugin:backendRestarted` Wails event and calls `invalidateBackendAddrCache()` while the plugin UI
    is mounted, so a detected plugin-subprocess crash/restart doesn't leave a stale cached address.
- `@litelens/core` usage: `useClusterWideAPI()` (imported from `@litelens/core`) is the host-exposed
  hook plugin components use for cluster-scoped host capabilities — `resourceLinks` (jump to a
  resource's detail drawer, e.g. `HelmReleaseDetailDrawer.tsx` linking a release's owned
  Deployment/Pod/etc.), `unifiedTray`, `activeContext`/`activeNamespaces`, and
  `useRegisterNavEntry`/`useRegisterTrayFamilies`/`useRegisterClusterWideEvents`. It must only be
  called from components rendered inside the host's single-cluster view (`MainLayout`'s subtree) —
  calling it from an app-wide screen throws, because the underlying host context has no provider
  there.
- CSS: Tailwind is compiled to `src/generated-style.css` by `pnpm build:css` _before_ `tsup` runs
  (`package.json`'s `build` script), then imported as a raw string in `src/pluginStyles.ts` and
  exported as `PLUGIN_STYLES` — the plugin ships its styles embedded in the JS bundle, not as a
  separate stylesheet asset. `loader: { ".css": "text" }` in `tsup.config.ts` must stay at the
  top-level config (not inside `esbuildOptions`) because tsup's own CSS-handling plugin reads it
  before `esbuildOptions` runs.
- `hooks/data-access/` (queries) and `hooks/data-mutation/` (mutations) wrap the `api/` layer with
  `@tanstack/react-query`; `hooks/async-events/` handle long-running operations (install/upgrade/
  cleanup) that stream progress via the event bridge (`useRegisterHelmEvents.ts`, subscribed through
  `useClusterWideAPI().useRegisterClusterWideEvents`) rather than a single request/response.

### Local install-mirroring script

`plugins/helm/scripts/deploy-plugin-helm-local.mjs` reproduces what a real plugin install looks like under
`~/.litelens/plugins/helm`, but entirely inside `plugins/helm/.output/` — never touches the real install
directory. Useful when testing changes against the main litelens app without going through the
CI-built release artifact. It intentionally does **not** produce `helm.lock` — that's runtime state
created only by the host app's plugin process loader while the plugin is actually running.

## Conventions

- Go backend and TS frontend payload shapes must be kept in sync by hand across three places: the
  route + handler in `internal/api/rest/handlers.go`, the corresponding export in
  `frontend/src/api/wailsBridge.ts`, and the type in `frontend/src/api/resources.ts`.
- Frontend tests live under `__tests__/` alongside the components they cover, using vitest +
  `@testing-library/react`.
- Go tests are colocated as `*_test.go` in the same package.
