import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";
import path from "node:path";

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: [
      // A handful of components reach into the host app's shared UI (e.g.
      // UnifiedTrayContext) the same way they do at runtime via the frontend's
      // own "@" alias — mirrored here so those tests resolve standalone.
      { find: "@", replacement: path.resolve(__dirname, "../../../frontend/src") },
    ],
  },
  test: {
    environment: "jsdom",
    setupFiles: ["./src/test-setup.ts"],
    coverage: {
      provider: "v8",
      reportsDirectory: "./coverage",
      reporter: ["text", "html", ["text-summary", { file: "./summary.txt" }]],
      include: ["src/**/*.{ts,tsx}"],
      exclude: ["src/**/*.test.{ts,tsx}", "src/test-setup.ts", "src/index.ts"],
    },
  },
});
