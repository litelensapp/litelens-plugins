---
name: file-structure
description: "Repo layout of litelens-plugins — pnpm workspace + Go module structure, hexagonal-architecture backend/frontend directory map for the helm plugin"
metadata: 
  node_type: memory
  type: reference
  originSessionId: 7d5e9c83-be5e-4cf5-9fd6-36ce1356d5fc
  modified: 2026-08-28T14:36:35.726Z
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
- The reusable gRPC pub/sub sync machinery (client, backoff, generic event-route/dispatch framework,
  event DTOs) lives in `packages/core/async` in the sibling `litelens` (host) repo, not in this repo —
  `plugins/helm/go.mod` currently depends on it via a **temporary local `replace` directive** pending a
  real tagged release. See [[architecture-call-path]].

`plugins/helm/internal/` (Go backend, HTTP + gRPC-client subprocess, **hexagonal architecture** —
refactored from the old flat `internal/api`/`internal/helm`/`internal/kube` layout):
- `internal/main.go` — process entrypoint. Reads an auth token from stdin
  (`util.ReadAuthTokenFromStdin`, before any gRPC ops), dials the host's gRPC server if
  `LITELENS_HOST_GRPC_PORT` is set (via `coreasync.GrpcClient` — aliased `coreasync` in this file only,
  since `main` also imports the plugin's own local `presentations/async` package under the bare
  identifier `async`), builds a `kube.DynamicClusterProvider`, wraps `applications/helm.Service` in
  `applications/lock.LockedService`, builds `presentations/async.NewEventDispatcher(dp)` (two
  `coreasync.EventRoute`s — cluster-context, active-namespaces — each still dialing its *own* dedicated
  `GrpcClient` internally, sharing one would tear down the other's stream on reconnect) and calls
  `.StartAll(...)`, then `dp.WaitForInitialSync(5s)` to block the first business call until the host's
  replay lands, then starts `adapters/presentations/rest.HttpServer` (blocks for process lifetime).
- `internal/applications/` — framework-agnostic core:
  - `applications/port/driven.go` — outbound interfaces the core depends on: `ClusterProvider`,
    `MutableClusterProvider`, `RESTClientGetterFactory`, `EventEmitter` func type. No longer holds an
    `EventReceiver` alias — `kube.DynamicClusterProvider` implements
    `github.com/litelensapp/litelens/packages/core/async.EventReceiver` directly (the port-layer
    indirection for this interface was abolished).
  - `applications/port/driver.go` — `HelmService` interface (what the core exposes inbound), defined
    here rather than in `applications/helm` to avoid circular imports.
  - `applications/helm/` — `Service` (`helm.go`/`helm_chart.go`/`helm_release.go`/`utils.go`), talks to
    Helm SDK (`helm.sh/helm/v3`) + Kubernetes via `port.ClusterProvider`.
  - `applications/lock/lock.go` — `LockedService` wraps `*helm.Service` in `sync.RWMutex` (business
    methods take `RLock`, `SetActiveContext` takes `Lock`) so a context swap can't race an in-flight
    business call.
  - `applications/dto/` — `helm.go` only (Helm-specific DTOs, plugin-owned). The event DTOs
    (`ClusterContextEvent`/`ActiveNamespacesEvent`, formerly `dto/kube.go`) and the `SubscribeStream`
    type (formerly `dto/stream.go`) moved to `packages/core/async` (`dto.go`/`stream.go`) — they're
    shared plugin-sync types, not helm-specific.
  - `applications/helm/cache.go` — in-memory `cache` used by `Service` to avoid rebuilding
    `action.Configuration` / re-parsing repo index files / re-running discovery on every business
    call. Keyed by `configCacheKey{context, namespace}` (namespace-keyed because
    `action.Configuration.Init` binds a fixed storage namespace — sharing across namespaces in the
    same context returns the wrong one); discovery results are cluster-wide so they're cached
    separately per-context; repo index entries carry a 10-minute TTL. `invalidateAllConfigs` clears
    both config and discovery caches on active-context switch. See [[architecture-call-path]] for why
    context switches must invalidate this.
- `internal/adapters/presentations/rest/` — inbound HTTP adapter. `server.go`'s `NewHttpServer` binds
  the listener (hard-fails if not localhost — CORS-reflects `Origin`, so non-localhost would be
  unsafe), wires `handlers.go`'s `RegisterRoutes` (`POST /api/helm/<camelCaseMethod>`, one per
  `port.HelmService` method) onto `http.ServeMux` via `corsMiddleware`. `Serve()` emits the one-line
  JSON handshake (`{"type":"READY","version":...,"httpPort":...,"pid":...,"timestamp":...}`) before
  blocking. `response.go`: `writeError`/`writeJSON`, codes `PLUGIN_UNAVAILABLE` (503),
  `INVALID_REQUEST` (400), `NOT_FOUND` (404), `INTERNAL_ERROR` (500).
- `internal/adapters/infrastructures/app/` no longer exists — the outbound gRPC pub/sub client
  (`GrpcClient.Dial`/`Subscribe`/`Publish`/`DialAndSubscribe`), auth interceptor
  (`NewAuthInterceptors`), event emitter (`Emit`), and backoff reconnector (`BackoffReconnector`) were
  all generic (not Helm-specific) and have been extracted into `packages/core/async` in the sibling
  `litelens` repo (see [[architecture-call-path]]), reusable by any future plugin. `pb` is still
  imported from `github.com/litelensapp/litelens/packages/core/pb` — no local `.proto` copy.
- `internal/adapters/presentations/async/` — thin helm-specific wiring over the shared
  `packages/core/async` event-route framework (unaliased import — this package's own name is also
  `async`, so no collision): `handlers.go`'s `Handler` (dispatches deserialized `async.
  ClusterContextEvent`/`async.ActiveNamespacesEvent` to an `async.EventReceiver`) and
  `dispatcher.go`'s `EventDispatcher` (holds `[]async.EventRoute` built via `async.NewRoute(topic,
  handler, deserializer)` for the two topics, `StartAll` loops and `go route.Run(grpcAddr, authToken)`
  for each — core no longer exposes its own `EventDispatcher`/`NewEventDispatcher`, that type was
  abolished in core and this plugin-local wrapper now owns the route slice + start loop directly).
- `internal/adapters/infrastructures/kube/` — `cluster.go`'s `DynamicClusterProvider` (mutable active
  cluster client + active-namespace filter, `SetActiveContext`, `SyncClusterContext`/
  `SyncActiveNamespaces` implementing `async.EventReceiver`, `WaitForInitialSync` gating). No longer
  has its own `WatchClusterContext`/`WatchActiveNamespaces` reconnect loops or a local
  `connector.go`/`BackoffReconnector` — that reconnect-loop logic now lives generically inside each
  `packages/core/async.eventRoute[T].Run`, driven by `presentations/async.EventDispatcher`.
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
- `src/api/query.client.ts` — exports `queryClient` (`appWideAPI.getQueryClient()`), a shared
  singleton. All data-mutation hooks import this instead of calling `useQueryClient()` or
  `appWideAPI.getQueryClient()` themselves — keeps cache reads/writes on one instance.
- `hooks/data-access/` — react-query queries; `hooks/data-mutation/` — mutations
  (`useInstallHelmChart.tsx`, `useUpgradeHelmChart.tsx` seed an optimistic `pending-install` /
  `pending-upgrade` release into the `QUERY_KEY_HELM_RELEASES` cache in `onSuccess`, ahead of the
  async `helm:install:complete`/`upgrade:complete` event that does the definitive invalidate — see
  [[architecture-call-path]]). (There is no `hooks/async-events/` directory anymore — event handling
  lives in top-level `src/events.ts`.)
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
