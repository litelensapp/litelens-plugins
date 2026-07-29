import { defineConfig } from "tsup";

export default defineConfig({
  entry: ["src/index.ts"],
  format: ["esm"],
  outDir: "dist",
  target: "es2020",
  splitting: true,
  sourcemap: false,
  clean: true,
  // These share the host app's own module instances via the import map in
  // frontend/index.html (see frontend/public/vendor/*.js) instead of being
  // bundled into the plugin — required for react-dom/@tanstack/react-query
  // context objects to resolve correctly when a plugin component is mounted
  // inline in the host's fiber tree.
  external: ["react", "react-dom", "@litelens/design-system", "@tanstack/react-query"],
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
});
