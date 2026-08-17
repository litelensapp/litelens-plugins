#!/usr/bin/env node
// Resolves the Helm plugin's logo asset filename (if present), shared by
// deploy-plugin-helm-local.mjs (local dev) and the `manifest`/upload steps
// in .github/workflows/job-build-plugin-helm.yml (CI/release), so both use
// the same source file. The logo ships as an actual file inside the plugin's
// install directory (alongside dist/ and the binary) rather than embedded in
// the manifest — manifest.assets.logo holds this filename, and it's served
// over the same /api/plugins/{pluginID}/* route as the bundle. Looked up by
// filename `helm.<ext>` under the plugin root, first match wins.
import { existsSync } from "node:fs";
import path from "node:path";

// Order matters: first existing file wins if more than one is present.
const CANDIDATES = ["helm.svg", "helm.png", "helm.jpg", "helm.jpeg"];

export function resolveLogoFile(pluginRoot) {
  for (const file of CANDIDATES) {
    if (existsSync(path.join(pluginRoot, file))) return file;
  }
  return undefined;
}

// CLI mode: `node resolve-logo.mjs` prints the filename (or nothing) to
// stdout, for shell callers (CI) to capture — e.g. `LOGO_FILE=$(node resolve-logo.mjs)`.
if (import.meta.url === `file://${process.argv[1]}`) {
  const pluginRoot = path.resolve(import.meta.dirname, "..");
  const file = resolveLogoFile(pluginRoot);
  if (file) process.stdout.write(file);
}
