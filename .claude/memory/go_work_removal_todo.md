---
name: go-work-removal-todo
description: go.work at repo root is a temporary shim until packages/core is first tagged — remove it once that happens
metadata: 
  node_type: memory
  type: project
  modified: 2026-08-19T03:23:32.733Z
  originSessionId: 32bc02f8-5d0a-4078-9df6-2a80ccf5ce39
---

Root `go.work` (created during Phase 7 of plugin-architecture-inversion) is an interim mechanism only: it lets `plugins/helm/` import `packages/core/{pb,dto,kube}` from the litelens host repo's local checkout because no `packages/core/vX.Y.Z` git tag has ever been published yet — Go can't resolve a versioned module dependency without one.

**Why:** `packages/core` is a separate nested Go module (own `go.mod` in the host repo), so without a tag, `go build ./plugins/helm/...` can't resolve `github.com/litelensapp/litelens/packages/core/...` as a normal dependency. `go.work` substitutes the on-disk directory instead, with no changes to either `go.mod`.

**How to apply:** once the host repo (`litelens`) publishes and tags `packages/core/vX.Y.Z` (watch the host's `.github/workflows/` publish step):
1. Delete `go.work` and `go.work.sum` from this repo root.
2. In `plugins/helm/go.mod`, run `go get github.com/litelensapp/litelens/packages/core@packages/core/vX.Y.Z` to add a normal pinned dependency.
3. Re-run `go test -race ./... && go vet ./...` in `plugins/helm/` to confirm the switch didn't break anything.

Until then, `go.work` carries a `// TODO` comment pointing back to this same removal sequence.
