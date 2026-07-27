import { defineConfig } from "tsup";

export default defineConfig({
  entry: ["src/index.ts"],
  format: ["esm"],
  outDir: "dist",
  target: "es2020",
  splitting: true,
  sourcemap: false,
  clean: true,
  external: ["react", "react-dom", "@litelens/design-system"],
  esbuildOptions: (options) => {
    options.banner = {
      js: "/* Plugin bundle - loaded dynamically */",
    };
  },
});
