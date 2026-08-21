import { prepareVisualizerData, visualizer } from "esbuild-visualizer";
import fs from "node:fs";
import path from "node:path";
import { defineConfig } from "tsup";

const shouldAnalyze = process.env.ANALYZE === "true";

export default defineConfig({
  entry: ["src/index.ts"],
  format: ["esm"],
  outDir: "dist",
  target: "es2020",
  splitting: true,
  sourcemap: false,
  clean: true,
  metafile: shouldAnalyze,
  // These share the host app's own module instances via the import map in
  // frontend/index.html (see frontend/public/vendor/*.js) instead of being
  // bundled into the plugin — required for react-dom/@tanstack/react-query
  // context objects to resolve correctly when a plugin component is mounted
  // inline in the host's fiber tree.
  external: [
    "react",
    "react-dom",
    "@litelens/design-system",
    "@tanstack/react-query",
    "@litelens/core",
  ],
  // Compiled Tailwind CSS is generated to src/generated-style.css before this
  // build runs (see package.json's build script) and imported as a raw string
  // in src/pluginStyles.ts, so it ends up embedded directly in dist/index.js
  // instead of shipped as a separate stylesheet asset. Must be set here (tsup's
  // top-level `loader` option) rather than inside esbuildOptions — tsup's own
  // CSS-handling esbuild plugin reads this value before esbuildOptions runs.
  loader: {
    ".css": "text",
  },
  esbuildOptions: (options) => {
    options.banner = {
      js: "/* Plugin bundle - loaded dynamically */",
    };
  },
  onSuccess: async () => {
    // Generate bundle report (treemap HTML + raw-data JSON) from the esbuild metafile
    const metafilePath = path.join(__dirname, "dist", "metafile-esm.json");
    if (shouldAnalyze && fs.existsSync(metafilePath)) {
      const statsDir = path.join(__dirname, "dist", "stats");
      fs.mkdirSync(statsDir, { recursive: true });

      const metadata = JSON.parse(fs.readFileSync(metafilePath, "utf-8"));

      const html = await visualizer(metadata, {
        title: "@litelens/helm-plugin-frontend Bundle Report",
        template: "treemap",
      });
      fs.writeFileSync(path.join(statsDir, "bundle-report.html"), html);
      fs.writeFileSync(path.join(statsDir, "bundle-stats.json"), prepareVisualizerData(metadata));
      fs.renameSync(metafilePath, path.join(statsDir, "metafile-esm.json"));
    }
  },
});
