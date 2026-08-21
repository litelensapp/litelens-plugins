#!/usr/bin/env bash
set -euo pipefail

# Builds the plugin-helm Go binary (the gRPC plugin subprocess launched by the
# host app). Shared by local dev (deploy-plugin-helm-local.mjs) and CI
# (job-build-plugin-helm.yml) so both use identical build flags.
#
# Env vars (all optional):
#   GOOS      - target OS (default: host OS via `go env GOOS`)
#   GOARCH    - target arch (default: host arch via `go env GOARCH`)
#   VERSION   - embedded via -X main.Version (default: "dev")
#   OUTPUT    - output binary path, relative paths resolve against plugins/helm/
#               (default: .output/plugin-helm[.exe])

cd "$(dirname "${BASH_SOURCE[0]}")/.."

GOOS="${GOOS:-$(go env GOOS)}"
GOARCH="${GOARCH:-$(go env GOARCH)}"
VERSION="${VERSION:-dev}"

EXT=""
if [[ "$GOOS" == "windows" ]]; then
  EXT=".exe"
fi

OUTPUT="${OUTPUT:-.output/plugin-helm${EXT}}"

mkdir -p "$(dirname "$OUTPUT")"

echo "-> Building plugin-helm (GOOS=$GOOS GOARCH=$GOARCH VERSION=$VERSION) -> $OUTPUT"

GOOS="$GOOS" GOARCH="$GOARCH" go build \
  -o "$OUTPUT" \
  -ldflags "-s -w -X main.Version=$VERSION" \
  -trimpath \
  ./internal

echo "done: $OUTPUT"
