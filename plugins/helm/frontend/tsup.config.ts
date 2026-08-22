import { createPluginTsupConfig } from "@litelens/core/tsup-preset";
import path from "node:path";
import { fileURLToPath } from "node:url";

export default createPluginTsupConfig({
  pluginRoot: path.dirname(fileURLToPath(import.meta.url)),
});
