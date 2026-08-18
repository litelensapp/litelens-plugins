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

### Backend (`plugins/helm/internal/`) — gRPC plugin subprocess

- `main.go` — process entrypoint. Builds a `dynamicClusterProvider` from `-kubeconfig`, starts the
  gRPC server on `-listen` (default `127.0.0.1:0`, auto-assigned port), then emits **exactly one**
  JSON handshake line (`{"type":"READY","grpcPort":...,"pid":...}`) to stdout before serving. The
  host app parses this line to discover the subprocess's port — never add other stdout writes before
  the handshake, and never change its shape without updating the host app in lockstep.
- `internal/helm/` — business logic (`Service`). Talks to the Helm SDK (`helm.sh/helm/v3`) and to
  Kubernetes via a `ClusterProvider` interface (`ActiveClients()`, `Ctx()`). The plugin subprocess's
  provider (`dynamicClusterProvider` in `main.go`) additionally implements `MutableClusterProvider`
  so the host can re-target the active cluster context on every call via `SetActiveContext` — the
  host resends cluster context because the subprocess has no persistent session concept.
  - `plugins/helm/internal/helm/rest/rest.go` — a `genericclioptions.RESTClientGetter` shim wired to a live
    `rest.Config` instead of re-reading kubeconfig from disk, so Helm SDK actions honor the currently
    active cluster.
- `internal/server/` — wraps `Service` behind the gRPC `Plugin` service (`GetCapabilities`,
  `SetClusterContext`, `Invoke`). `Invoke` is a single generic RPC: it takes a `method` string and a
  `payloadJson` string, and `dispatch()` in `grpc.go` switches on `method` to unmarshal into the
  right typed request struct and call the matching `Service` method, marshaling the result back to
  JSON. **Any new backend capability needs a case added to `dispatch()`** — there is no
  reflection-based routing.
  - `internal/server/pb/plugin.proto` — the wire contract. Comment at the top of the file: it must
    stay byte-identical (aside from `go_package`) to the copy in the main litelens app repo
    (`internal/plugin/pb/plugin.proto`) until this plugin is fully split out. Regenerate
    `plugin.pb.go` / `plugin_grpc.pb.go` after editing, and keep both copies of the `.proto` in sync
    manually.

### Frontend (`plugins/helm/frontend/src/`) — dynamically-loaded ES module

- Builds to a standalone ESM bundle (`tsup.config.ts`) loaded by the host via runtime `import()` —
  it is **not** part of the host app's own Vite build, so it can't use the host's `@wailsjs` alias
  or bundle its own copy of `react`/`react-dom`/`@tanstack/react-query`/`@litelens/design-system`.
  Those are all in `external` in `tsup.config.ts` and resolve at runtime through the host's import
  map (`frontend/public/vendor/*.js` in the main app) — this is required so shared React/context
  instances line up when a plugin component mounts inline in the host's fiber tree. When adding a
  new dependency to the frontend, check whether it needs to go in `external` too.
- `src/index.ts` is the plugin's barrel/contract file — the host discovers plugin capabilities only
  through these named exports (`PluginView`, `PLUGIN_NAV_ENTRY`, `PluginEventBridge`,
  `PLUGIN_STYLES`, `PLUGIN_TRAY_FAMILIES`). Internal components keep Helm-specific names; only the
  barrel re-exports under the generic contract names the host expects.
  `PLUGIN_TRAY_FAMILIES` is a runtime-keyed map (e.g. `"helm-chart"`, `"helm-chart-upgrade"`) — the
  host calls `unifiedTray.openTab(family, params)` with these keys and has no static knowledge of
  which families exist or what params they take, so family names are effectively part of the wire
  contract between this plugin and the host's tray system.
- `src/api/wailsBridge.ts` — every exported function calls
  `window.go.app.App.InvokePlugin("helm", method, JSON.stringify(payload))` and JSON-parses the
  response. `method` strings and payload field names here **must match** the `dispatch()` switch in
  `plugins/helm/internal/server/grpc.go` exactly (including Go's PascalCase field names in the payload,
  since it's unmarshaled directly into anonymous Go structs). Adding a backend method means adding
  both a `dispatch()` case and a matching `wailsBridge.ts` export.
- CSS: Tailwind is compiled to `src/generated-style.css` by `pnpm build:css` _before_ `tsup` runs
  (`package.json`'s `build` script), then imported as a raw string in `src/pluginStyles.ts` and
  exported as `PLUGIN_STYLES` — the plugin ships its styles embedded in the JS bundle, not as a
  separate stylesheet asset. `loader: { ".css": "text" }` in `tsup.config.ts` must stay at the
  top-level config (not inside `esbuildOptions`) because tsup's own CSS-handling plugin reads it
  before `esbuildOptions` runs.
- `hooks/data-access/` (queries) and `hooks/data-mutation/` (mutations) wrap the `api/` layer with
  `@tanstack/react-query`; `hooks/async-events/` handle long-running operations (install/upgrade/
  cleanup) that stream progress via the event bridge rather than a single request/response.

### Local install-mirroring script

`plugins/helm/scripts/deploy-plugin-helm-local.mjs` reproduces what a real plugin install looks like under
`~/.litelens/plugins/helm`, but entirely inside `plugins/helm/.output/` — never touches the real install
directory. Useful when testing changes against the main litelens app without going through the
CI-built release artifact. It intentionally does **not** produce `helm.lock` — that's runtime state
created only by the host app's plugin process loader while the plugin is actually running.

## Conventions

- Go backend and TS frontend payload shapes must be kept in sync by hand across three places:
  `dispatch()` in `grpc.go`, the corresponding export in `wailsBridge.ts`, and the type in
  `frontend/src/types/index.ts` / `frontend/src/api/resources.ts`.
- Frontend tests live under `__tests__/` alongside the components they cover, using vitest +
  `@testing-library/react`.
- Go tests are colocated as `*_test.go` in the same package.
