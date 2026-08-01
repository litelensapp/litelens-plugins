#!/usr/bin/env node
// Mirrors what a real Helm plugin install looks like under
// ~/.litelens/plugins/helm, but entirely inside plugins/helm/.output —
// for local verification only. Does NOT touch ~/.litelens/plugins/helm;
// that directory is populated exclusively by the real install/update flow
// (InstallPlugin downloading the CI-built binary + tar.gz archive).
//
// Produces, under plugins/helm/.output/:
//   dist/                          - the built frontend bundle (index.js + chunks)
//   helm-plugin-frontend.tar.gz    - the same archive CI ships, for checksum parity
//   plugin-helm                    - the Go plugin binary, built for the host platform
//   .plugin-metadata.json          - mirrors the real installed metadata shape
//
// Not produced: helm.lock. That file is runtime state created and removed by
// the plugin process loader (internal/plugin/loader.go) while the plugin is
// actually running — it has no meaningful "build" representation to mirror.
import { createHash } from "node:crypto";
import { execFileSync } from "node:child_process";
import { existsSync, cpSync, rmSync, mkdirSync, writeFileSync, readFileSync, statSync } from "node:fs";
import path from "node:path";

const pluginRoot = path.resolve(import.meta.dirname, "..");
const builtDist = path.join(pluginRoot, "frontend", "dist");
const outputDir = path.join(pluginRoot, ".output");
const outputDist = path.join(outputDir, "dist");
const archivePath = path.join(outputDir, "helm-plugin-frontend.tar.gz");
const binaryName = process.platform === "win32" ? "plugin-helm.exe" : "plugin-helm";
const binaryPath = path.join(outputDir, binaryName);
const metadataPath = path.join(outputDir, ".plugin-metadata.json");

if (!existsSync(builtDist)) {
  console.error(`Build output not found at ${builtDist}. Run the frontend build first.`);
  process.exit(1);
}

mkdirSync(outputDir, { recursive: true });

// dist/
rmSync(outputDist, { recursive: true, force: true });
cpSync(builtDist, outputDist, { recursive: true });

// tar.gz archive (same shape as the CI-shipped asset, for checksum parity)
execFileSync("tar", ["-C", outputDist, "-czf", archivePath, "."]);
const bundleSha256 = createHash("sha256").update(readFileSync(archivePath)).digest("hex");

// Go binary, built for the host platform via the shared build.sh (same script CI uses)
execFileSync("bash", [path.join(pluginRoot, "scripts", "build.sh")], {
  cwd: pluginRoot,
  env: { ...process.env, VERSION: "local-dev", OUTPUT: binaryPath },
  stdio: "inherit",
});

// Metadata, mirroring pluginMetadata in internal/app/plugin.go (which in turn
// mirrors plugin.Manifest / the frontend's PluginManifest interface) plus the
// two install-specific fields, releaseTag and installedAt.
const binarySize = statSync(binaryPath).size;
writeFileSync(
  metadataPath,
  JSON.stringify(
    {
      id: "helm",
      name: "Helm",
      description: "Manage Helm charts and releases in your Kubernetes clusters",
      version: "local-dev",
      repository: "https://github.com/gknguyen/litelens/releases",
      homepage: "https://helm.sh",
      minimumHostVersion: "0.1.0",
      maximumHostVersion: "999.999.999",
      os: {
        linux: ["amd64"],
        darwin: ["arm64"],
        windows: ["amd64"],
      },
      bundle: {
        sha256: bundleSha256,
        size: statSync(archivePath).size,
      },
      binary: {
        sha256: createHash("sha256").update(readFileSync(binaryPath)).digest("hex"),
        size: binarySize,
      },
      capabilities: ["helm-charts", "helm-releases"],
      releaseTag: "local-dev",
      installedAt: new Date().toISOString(),
    },
    null,
    2
  )
);

console.log(`Mirrored local Helm plugin build to ${outputDir}`);
console.log(
  `(no helm.lock — that's runtime state created only while the plugin process is actually running)`
);
