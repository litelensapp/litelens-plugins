---
name: architecture-call-path
description: "Plugin/host call-path — business calls go frontend-to-backend over HTTP directly; gRPC is a generic pub/sub control channel the plugin dials outbound, auth'd via a stdin-delivered bearer token"
metadata: 
  node_type: memory
  type: project
  originSessionId: 7d5e9c83-be5e-4cf5-9fd6-36ce1356d5fc
  modified: 2026-08-27T13:08:00.926Z
---

The business/data-plane call path is **plugin frontend → plugin backend directly over localhost HTTP**
(`listCharts`, `installChart`, etc.), bypassing the host app for every business call. The host still
owns process lifecycle (spawn, handshake parsing, crash detection) and cluster-context/active-namespace
propagation, since those are inherent to running plugins as separate OS processes.

A gRPC connection still exists but only as a thin, **generic pub/sub control channel** in the
*opposite* direction from the old model: the plugin dials the host (`GrpcClient.Dial`) and
`Subscribe`s to topics — `cluster.context` and `namespaces.active` (via `util.EventTopicClusterContext`
/ `util.EventTopicNamespacesActive` constants from `packages/core`) — rather than the host driving
every plugin call through a method-relay `Invoke`/`dispatch()`. The same `GrpcClient` type also
`Publish`es async progress events outbound, to topic `plugins.<pluginID>.<eventName>` (e.g.
`plugins.helm.helm:install:complete`).

**Auth**: the plugin reads a bearer token from stdin at startup (`util.ReadAuthTokenFromStdin`,
*before* any gRPC operation) and `NewAuthInterceptors` attaches it as `authorization: bearer <token>`
to every outgoing unary/stream gRPC call — this wasn't present in the earlier control-channel design
and has no HTTP-side equivalent (the HTTP server is CORS-hardened to localhost-only instead).

**Two independent watch loops, two independent connections**: `DynamicClusterProvider.WatchClusterContext`
and `.WatchActiveNamespaces` each dial their *own* dedicated `GrpcClient` rather than sharing one —
`GrpcClient.Dial` closes and replaces the instance's connection, so two watch loops sharing one
instance would tear down each other's live stream on every reconnect. Both use exponential backoff
(`kube.BackoffReconnector`, 30s max interval) and block `main.go` via `WaitForInitialSync(5s)` so the
first HTTP business call isn't served against the kubeconfig-derived guess from `BuildClusterProvider`
(wrong cluster / unfiltered namespaces) — a result the frontend would then cache indefinitely.

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
