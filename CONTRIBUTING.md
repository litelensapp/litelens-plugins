# Contributing

This is the official plugin repository for [Litelens](https://github.com/litelensapp/litelens), a
Kubernetes desktop app built with Wails. Each plugin pairs a Go gRPC/HTTP subprocess (backend) with
a dynamically-loaded TypeScript/React ES module (frontend). Today the repo holds one plugin,
`plugins/helm/`; this guide covers both the shared repo layout and the conventions any new plugin
should follow.

## Prerequisites

- Node 26 (see `engines` in `package.json`) and pnpm 11 (`packageManager` field — use
  [corepack](https://nodejs.org/api/corepack.html) or install the matching version directly)
- Go 1.26+ — `staticcheck` is a `tool` directive dependency, not a separate install
- A sibling checkout of the [`litelens`](https://github.com/litelensapp/litelens) host repo. Both
  the Go and JS layers currently depend on unreleased code from it (see
  [Working against an unreleased `packages/core`](#working-against-an-unreleased-packagescore)
  below) — clone it next to this repo, e.g.:

  ```
  parent/
    litelens/            # host app
    litelens-plugins/     # this repo
  ```

## Getting set up

```bash
pnpm install       # installs the JS workspace (plugins/helm/frontend)
```

`pnpm install` requires a local Verdaccio registry running for the `@litelens` npm scope (see
`.npmrc`) — `@litelens/core` has no public npm release yet, so pnpm resolves it from
`http://localhost:4873/` instead. Ask an existing contributor how they run Verdaccio locally, or how
their workspace links `@litelens/design-system` and `@litelens/core`, before assuming `pnpm install`
or `pnpm test:helm:fe` is broken — this is expected friction until those packages ship to the real
registry.

## Repo layout

- **pnpm workspace** — `pnpm-workspace.yaml` includes `plugins/helm/frontend` only; add new plugin
  frontends here as they're created.
- **Go module** — a single root module `github.com/litelensapp/litelens-plugins`, plus one nested
  `go.mod` per plugin whose backend needs heavy dependencies (e.g. `plugins/helm/go.mod` depends on
  `helm.sh/helm/v3` and `k8s.io/client-go`) that shouldn't pollute the root module or every other
  plugin's build.
- **`go.work`** — a temporary workspace shim substituting an on-disk sibling `litelens/packages/core`
  checkout for the unreleased `packages/core` module dependency. See below.

## Working against an unreleased `packages/core`

`plugins/helm/go.mod` requires `github.com/litelensapp/litelens/packages/core@v0.1.0`, and
`plugins/helm/frontend/package.json` requires `@litelens/core` — neither has a real tagged release
yet. Two shims paper over this in the meantime, and **both must be removed together** once
`packages/core` ships a real, aged release:

- root `go.work` — `use (../litelens/packages/core ./plugins/helm)` points Go at the sibling
  checkout instead of the versioned module.
- root `.npmrc` — points the `@litelens` npm scope at a local Verdaccio instance serving
  pre-release `@litelens/core` versions, and `pnpm-workspace.yaml`'s `minimumReleaseAgeExclude`
  allow-lists those exact pre-release versions so pnpm doesn't flag them as suspiciously fresh.

Do not treat either shim as a model for permanent dependency management — they exist solely because
the host repo hasn't cut a real release yet. See `.claude/memory/go_work_removal_todo.md` for the
exact Go-side removal steps once a tag exists.

## Building, testing, linting

Run from the repo root unless noted otherwise:

```bash
pnpm build:helm:fe          # build the helm frontend plugin bundle (tailwind + tsup)
pnpm test:helm:fe           # helm frontend tests (vitest) — requires @litelens/design-system
pnpm test:helm:be           # go test -race -v ./... inside plugins/helm/

pnpm lint:fe                # eslint plugins/helm/frontend/src
pnpm lint:be                # go vet + staticcheck (go tool) inside plugins/helm/

pnpm format                 # prettier --write across ts/tsx/js/jsx/json/css/md/yml
```

Single-test invocations:

```bash
# One Go test
cd plugins/helm && go test -race -run TestHelmAge ./internal/helm/...

# One frontend test file
cd plugins/helm/frontend && pnpm vitest run src/components/release/__tests__/HelmReleaseStatusBadge.test.tsx
```

Build the actual plugin binary (used by CI and local dev alike):

```bash
cd plugins/helm && GOOS=linux GOARCH=amd64 VERSION=1.2.3 ./scripts/build.sh
```

Mirror a full local plugin install (frontend dist + binary + tar.gz + metadata) under
`<repo-root>/.output/<plugin-id>/` (e.g. `.output/helm/`), without touching your real
`~/.litelens/plugins/` install:

```bash
cd plugins/helm && node scripts/deploy-plugin-helm-local.mjs   # run build:helm:fe first
```

## Pre-commit hooks

Husky + lint-staged run on every commit (`.husky/pre-commit`): staged `*.ts`/`*.tsx` files get
`eslint --fix`, and staged `*.{ts,tsx,js,jsx,json,css,md,yml}` files get `prettier --write`. Don't
bypass this with `--no-verify` — fix the underlying lint/format issue instead.

## Conventions when adding or changing a plugin capability

Using `plugins/helm/` as the reference implementation:

- **New backend method** needs three things kept in sync by hand: a route + handler in
  `internal/api/rest/handlers.go` (`POST /api/helm/<camelCaseMethod>`, one route per `Service`
  method — there is no reflection-based routing), a matching export in
  `frontend/src/api/wailsBridge.ts` calling the shared `fetchWithRetry` helper (never a bare
  `fetch()`, to inherit its backend-address-retry behavior), and the corresponding type in
  `frontend/src/api/resources.ts`. URL segments are camelCase; request-body field names stay Go
  PascalCase since they unmarshal directly into anonymous Go structs.
- **New backend error path** should use the existing `writeError`/`writeJSON` contract in
  `internal/api/rest/response.go` — reuse the existing codes (`PLUGIN_UNAVAILABLE` 503,
  `INVALID_REQUEST` 400, `NOT_FOUND` 404, `INTERNAL_ERROR` 500) rather than inventing new ones, since
  the frontend throws on this exact `{code, message}` shape.
- **New frontend dependency**: check whether it needs to go in `external` in `tsup.config.ts` too —
  the frontend bundle is loaded via runtime `import()` by the host and can't ship its own copy of
  `react`/`react-dom`/`@tanstack/react-query`/`@litelens/design-system`/`@litelens/core` (those
  resolve at runtime through the host's import map instead).
- **New plugin capability exposed to the host** (nav entry, tray family, event bridge, styles) must
  be re-exported from `src/index.ts`, the barrel/contract file the host discovers plugin
  capabilities through. Keep Helm-specific names internal; only the barrel exports use the generic
  contract names.
- **Never** add stdout writes to the backend process before its one JSON handshake line
  (`{"type":"READY",...}` in `internal/main.go`), and don't change that JSON shape without updating
  the host app in lockstep — the host parses that exact line to discover the subprocess's port.
- **gRPC contract changes** (`plugin.proto`) are made in the `litelens` host repo
  (`packages/core/pb/plugin.proto`), not here — this repo only consumes the generated `pb` package
  via its `packages/core` dependency (currently the `go.work` shim above).

## Test locations

- Frontend tests live under `__tests__/` alongside the components they cover
  (`@testing-library/react` + vitest).
- Go tests are colocated `*_test.go` files in the same package as the code under test.

## Adding a new plugin

There's no scaffolding script yet — copy `plugins/helm/`'s shape as a starting point: a nested Go
module under `internal/` for the backend, a workspace-registered frontend package under
`frontend/`, and root `package.json` scripts (`build:<plugin>:fe`, `test:<plugin>:fe`,
`test:<plugin>:be`) following the existing `*:helm:*` naming. Register the new frontend path in
`pnpm-workspace.yaml`.

## Developer Certificate of Origin

All commits must be signed off using:

```bash
git commit -s -m "Your commit message"
```

This appends a `Signed-off-by` trailer, certifying that you wrote the code and have the right to
contribute it to the project. See the `DCO` file for the full Developer Certificate of Origin v1.1
text. We use DCO for contribution provenance tracking — no CLA required.

## Submitting changes

- Fork the repository or create a feature branch off `master`
- Keep PRs focused on a single plugin/feature/bug fix
- Describe what you changed and why in the PR description
- Reference any related GitHub Issues

## Reporting Issues

- **Bugs:** Use GitHub Issues with details about reproduction, expected vs. actual behavior, and
  your environment (OS, K8s version, plugin version)
- **Features:** Use GitHub Issues to propose new capabilities
- **Security vulnerabilities:** Do not open a public issue — see `SECURITY.md`

## Code of Conduct

Participation in this project is governed by our `CODE_OF_CONDUCT.md`.

## License

Contributions to this repository are licensed under the Apache License 2.0 (see `LICENSE` file). By
contributing, you agree to license your contribution under the same terms.

---

Thanks for contributing! Please reach out with questions or feedback.

```sh
ln -s $PWD/.output  $HOME/Work/tech-projects/personal/litelens-app/litelens/build/storage/plugins
```
