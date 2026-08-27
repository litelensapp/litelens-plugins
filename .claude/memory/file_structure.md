---
name: file-structure
description: "Repo layout of litelens-plugins — pnpm workspace + Go module structure, hexagonal-architecture backend/frontend directory map for the helm plugin"
metadata: 
  node_type: memory
  type: reference
  originSessionId: 7d5e9c83-be5e-4cf5-9fd6-36ce1356d5fc
  modified: 2026-08-27T13:07:46.318Z
---

litelens-plugins is the official plugin repo for Litelens (Wails Kubernetes desktop app). Each plugin
pairs a Go gRPC-client subprocess (backend) with a dynamically-loaded TS/React ES module (frontend).
Currently the only plugin is `plugins/helm/`.

Repo-level:
- pnpm workspace (`pnpm-workspace.yaml` includes `plugins/helm/frontend`) at the JS layer.
- Single Go module `github.com/litelensapp/litelens-plugins` at the root, but `plugins/helm/` has its
  own nested `go.mod` (`.../litelens-plugins/plugins/helm`) since it pulls in heavy deps
  (`helm.sh/helm/v3`, `k8s.io/client-go`) that shouldn't pollute the root module.
- `staticcheck` is a Go 1.26+ `tool` directive in `plugins/helm/go.mod`, run via `go tool staticcheck ./...`.
- `docs/architecture/` holds the architecture diagram (`architecture-{light,dark}.mmd` + rendered
  `.svg`, `architecture.md` with the mermaid-cli regen command). Keep in sync with this file and
  [[architecture-call-path]] when the plugin/host wire contract changes.

`plugins/helm/internal/` (Go backend, HTTP + gRPC-client subprocess, **hexagonal architecture** —
refactored from the old flat `internal/api`/`internal/helm`/`internal/kube` layout):
- `internal/main.go` — process entrypoint. Reads an auth token from stdin
  (`util.ReadAuthTokenFromStdin`, before any gRPC ops), dials the host's gRPC server if
  `LITELENS_HOST_GRPC_PORT` is set, builds a `kube.DynamicClusterProvider`, wraps
  `applications/helm.Service` in `applications/lock.LockedService`, spawns
  `DynamicClusterProvider.WatchClusterContext` and `.WatchActiveNamespaces` goroutines (each dials its
  *own* dedicated `GrpcClient` — sharing one would tear down the other's stream on reconnect), calls
  `WaitForInitialSync(5s)` to block the first business call until the host's replay lands, then starts
  `adapters/presentations/rest.HttpServer` (blocks for process lifetime).
- `internal/applications/` — framework-agnostic core:
  - `applications/port/driven.go` — outbound interfaces the core depends on: `ClusterProvider`,
    `MutableClusterProvider`, `RESTClientGetterFactory`, `EventEmitter` func type.
  - `applications/port/driver.go` — `HelmService` interface (what the core exposes inbound), defined
    here rather than in `applications/helm` to avoid circular imports.
  - `applications/helm/` — `Service` (`helm.go`/`helm_chart.go`/`helm_release.go`/`utils.go`), talks to
    Helm SDK (`helm.sh/helm/v3`) + Kubernetes via `port.ClusterProvider`.
  - `applications/lock/lock.go` — `LockedService` wraps `*helm.Service` in `sync.RWMutex` (business
    methods take `RLock`, `SetActiveContext` takes `Lock`) so a context swap can't race an in-flight
    business call.
  - `applications/dto/` — `helm.go` (Helm-specific DTOs, plugin-owned) + `stream.go` (`SubscribeStream`
    type used by the gRPC adapter).
- `internal/adapters/presentations/rest/` — inbound HTTP adapter. `server.go`'s `NewHttpServer` binds
  the listener (hard-fails if not localhost — CORS-reflects `Origin`, so non-localhost would be
  unsafe), wires `handlers.go`'s `RegisterRoutes` (`POST /api/helm/<camelCaseMethod>`, one per
  `port.HelmService` method) onto `http.ServeMux` via `corsMiddleware`. `Serve()` emits the one-line
  JSON handshake (`{"type":"READY","version":...,"httpPort":...,"pid":...,"timestamp":...}`) before
  blocking. `response.go`: `writeError`/`writeJSON`, codes `PLUGIN_UNAVAILABLE` (503),
  `INVALID_REQUEST` (400), `NOT_FOUND` (404), `INTERNAL_ERROR` (500).
- `internal/adapters/infrastructures/app/` (package `grpc`) — outbound gRPC pub/sub client, generic
  (not Helm-specific): `client.go`'s `GrpcClient` (`Dial`, `Subscribe(topic)`, `Publish(ctx, topic,
  payloadJSON)`, `DialAndSubscribe`), `interceptor.go`'s `NewAuthInterceptors` (attaches `bearer
  <token>` to every outgoing unary/stream call), `emitter.go`'s `Emit` (marshals + `Publish`s to
  `plugins.<pluginID>.<eventName>`, async via goroutine + 5s timeout), `receiver.go`'s
  `RecvClusterContext`/`RecvActiveNamespaces` (decode JSON payload off a `SubscribeStream`). Imports
  `pb` from `github.com/litelensapp/litelens/packages/core/pb` — no local `.proto` copy.
- `internal/adapters/infrastructures/kube/` — `cluster.go`'s `DynamicClusterProvider` (mutable active
  cluster client + active-namespace filter, `SetActiveContext`, sync-from-host + `WaitForInitialSync`
  gating, `WatchClusterContext`/`WatchActiveNamespaces` reconnect loops) and `connector.go`'s
  `BackoffReconnector` (exponential backoff, 30s max interval).
- `internal/adapters/infrastructures/restconfig/` — `getter.go`'s `Getter` (implements
  `genericclioptions.RESTClientGetter` off a live `rest.Config`) + `Factory` (implements
  `port.RESTClientGetterFactory`).
- `internal/config/env.go` — `GetHostGRPCPort()` reads `LITELENS_HOST_GRPC_PORT` (empty when unset,
  e.g. running the binary standalone outside the host app).

`plugins/helm/frontend/src/` (TS/React, builds to standalone ESM via `tsup`):
- `src/index.ts` — **registration-based** plugin entrypoint (no longer a named-export barrel): calls
  `appWideAPI.registerStylesheets`, `clusterWideAPI.registerNavEntry`, `.registerTrayFamilies`,
  `.registerEvents`, `.registerViews` from `@litelens/core` at module load. The host discovers plugin
  capabilities by importing this module for its side effects, not by reading exported symbols.
- `src/const.ts` — `PLUGIN_ID`, `HELM_NAV_ENTRY`, `HELM_TRAY_FAMILIES` (tray family keys `"helm-chart"`,
  `"helm-chart-upgrade"` are wire contract with the host's `unifiedTray.openTab(family, params)`).
- `src/events.ts` — `eventHandlers` map keyed by event name (`helm:install:complete`,
  `helm:install:error`, `helm:upgrade:complete`, `helm:upgrade:error`, `helm:cleanup:complete`,
  `helm:cleanup:partial`, `helm:cleanup:error`), registered via `clusterWideAPI.registerEvents`; each
  handler toasts (`@litelens/design-system`) and invalidates react-query keys via
  `appWideAPI.getQueryClient()`.
- `src/api/wailsBridge.ts` — `fetchWithRetry(method, payload)` hits
  `http://<backendAddr>/api/helm/<camelCaseMethod>` directly; payload field names stay Go PascalCase.
  Must match the `RegisterRoutes` table in
  `internal/adapters/presentations/rest/handlers.go` exactly. Backend address from
  `window.go.app.App.GetPluginBackendAddr("helm")`, cached module-level; a thrown `TypeError` (network
  failure) triggers one cache-invalidate + refetch + retry, a parsed `{code,message}` error body does
  not retry.
- `src/api/resources.ts` — TS types mirroring the Go DTOs/handler payloads.
- `hooks/data-access/` — react-query queries; `hooks/data-mutation/` — mutations. (There is no
  `hooks/async-events/` directory anymore — event handling lives in top-level `src/events.ts`.)
- `src/style.css` → compiled by `pnpm build:css` before `tsup`, embedded as `PLUGIN_STYLES` equivalent
  registered via `appWideAPI.registerStylesheets`.
- Component tests live under `__tests__/` alongside the components they cover (vitest +
  @testing-library/react).

Other:
- `plugins/helm/scripts/build.sh` — builds the Go plugin binary (used by local dev + CI).
- `plugins/helm/scripts/deploy-plugin-helm-local.mjs` — mirrors a full local plugin install under
  `<repo-root>/.output/helm/` (frontend dist + binary + tar.gz + metadata); never touches the real
  `~/.litelens/plugins/helm` install dir; does not produce `helm.lock` (runtime-only, created by host).

Three-place payload sync required by hand: `internal/adapters/presentations/rest/handlers.go`
route+handler ↔ `frontend/src/api/wailsBridge.ts` export ↔ `frontend/src/api/resources.ts` type. See
[[architecture-call-path]].
