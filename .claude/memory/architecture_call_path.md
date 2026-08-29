---
name: architecture-call-path
description: "Plugin/host call-path — business calls go frontend-to-backend over HTTP directly; gRPC is a generic pub/sub control channel the plugin dials outbound, auth'd via a stdin-delivered bearer token"
metadata: 
  node_type: memory
  type: project
  originSessionId: 7d5e9c83-be5e-4cf5-9fd6-36ce1356d5fc
  modified: 2026-08-29T07:11:02.025Z
---

The business/data-plane call path is **plugin frontend → plugin backend directly over localhost HTTP**
(`listCharts`, `installChart`, etc.), bypassing the host app for every business call. The host still
owns process lifecycle (spawn, handshake parsing, crash detection) and cluster-context/active-namespace
propagation, since those are inherent to running plugins as separate OS processes.

A gRPC connection still exists but only as a thin, **generic pub/sub control channel** in the
*opposite* direction from the old model: the plugin dials the host (`GrpcClient.Dial`) and
`Subscribe`s to topics — `cluster.context` and `namespaces.active` (via `async.EventTopicClusterContext`
/ `async.EventTopicNamespacesActive` constants from `packages/core/async`) — rather than the host
driving every plugin call through a method-relay `Invoke`/`dispatch()`. The same `GrpcClient` type also
`Publish`es async progress events outbound, to topic `plugins.<pluginID>.<eventName>` (e.g.
`plugins.helm.helm:install:complete`).

**`packages/core/async` is shared, reusable machinery, not helm-specific.** `GrpcClient`,
`BackoffReconnector`, the generic `EventRoute`/`NewRoute[T]` route framework, and the event DTOs
(`ClusterContextEvent`/`ActiveNamespacesEvent`) were extracted out of this repo into the `litelens`
host repo's `packages/core/async` package so any future plugin can sync `activeContext`/
`activeNamespaces` the same way without reimplementing the watch/backoff/dispatch logic. This repo's
`plugins/helm/internal/adapters/presentations/async/` is now just thin helm-specific wiring:
`Handler` (dispatches to `async.EventReceiver`) and `EventDispatcher` (builds two `async.EventRoute`s
via `async.NewRoute` and loops `go route.Run(...)` in `StartAll` — core does **not** expose its own
`EventDispatcher` anymore, that type was abolished there in favor of each consumer owning its own
route slice + start loop). `plugins/helm/go.mod` pulls `packages/core` in as a tagged, published
version (currently v1.7.11); `frontend/package.json` pulls `@litelens/core` from npm the same way —
both were published 2026-08-29, replacing the earlier local `replace`/`link:` directives into the
sibling `litelens` checkout.

**Auth**: the plugin reads a bearer token from stdin at startup (`util.ReadAuthTokenFromStdin`,
*before* any gRPC operation) and `NewAuthInterceptors` attaches it as `authorization: bearer <token>`
to every outgoing unary/stream gRPC call — this wasn't present in the earlier control-channel design
and has no HTTP-side equivalent (the HTTP server is CORS-hardened to localhost-only instead).

**Two independent watch loops, two independent connections**: each `packages/core/async.eventRoute[T]`
(one per topic — cluster-context, active-namespaces, wired up by
`presentations/async.NewEventDispatcher`) dials its *own* dedicated `GrpcClient` in `runEventLoop`
rather than sharing one — `GrpcClient.Dial` closes and replaces the instance's connection, so two
watch loops sharing one instance would tear down each other's live stream on every reconnect. Both use
exponential backoff (`async.BackoffReconnector`, 30s max interval) and `main.go` blocks via
`dp.WaitForInitialSync(5s)` after `dispatcher.StartAll(...)` so the first HTTP business call isn't
served against the kubeconfig-derived guess from `BuildClusterProvider` (wrong cluster / unfiltered
namespaces) — a result the frontend would then cache indefinitely.

**Why:** documented as `litelens/.claude/plans/plugin-architecture-inversion.md` in the host
(`litelens`) repo — full design rationale for the HTTP-direct call path lives there. The generic
pub/sub (vs. per-topic RPC methods) and the hexagonal package split
([[file-structure]]) came later, in the "migrate to native gRPC pub/sub" and "refactor helm plugin to
use hexagonal architecture" commits.

**How to apply:** when adding a new backend capability, add an HTTP route (never extend gRPC for
business calls) — see [[file-structure]] for the three-place sync required
(`handlers.go` ↔ `wailsBridge.ts` ↔ `resources.ts`). Cluster-context and active-namespace changes have
no HTTP entry point — they're exclusively the two gRPC subscribe streams. `docs/architecture/*.mmd`
mirrors this call path; regenerate the SVGs (`docs/architecture/architecture.md` has the mermaid-cli
command) whenever the wire contract changes.

**Install/upgrade is fire-and-forget over HTTP, reconciled later over gRPC events.** The
`InstallHelmChart`/`UpgradeHelmChart` HTTP handlers only kick off a goroutine (`install.RunWithContext`
etc.) and return once the release name is resolved — before Helm has actually persisted the
pending-install/upgrade record. The frontend mutation hooks (`useInstallHelmChart.tsx`,
`useUpgradeHelmChart.tsx`) therefore don't invalidate+refetch on HTTP success (that would race the
goroutine and read the pre-install list); instead they seed the react-query cache directly with a
synthetic `pending-install`/`pending-upgrade` release row. The definitive reconciliation happens later,
off the `helm:install:complete`/`error` and `helm:upgrade:complete`/`error` gRPC-relayed events (see
[[file-structure]] `src/events.ts`), which do the real invalidate once Helm reaches deployed/failed.
**Why:** async HTTP-trigger + async gRPC-event-confirm is inherent to spawning `install.RunWithContext`
in a goroutine — there's no synchronous "done" signal to wait on over HTTP.

**Backend caches Helm `action.Configuration`/discovery/repo-index lookups** ([[file-structure]]
`cache.go`) instead of rebuilding them per business call — config cache is invalidated wholesale on
active-context switch (`invalidateAllConfigs`, called from `SetActiveContext`) since a new cluster
context makes every cached `action.Configuration`/discovery entry stale.
