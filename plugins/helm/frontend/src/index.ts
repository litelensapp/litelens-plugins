// Barrel export for plugin dynamic loading via import()
// This is the entry point for the ES module build used via runtime import()
// External consumers use the generic contract names (e.g. PluginView)
// Internal Helm components continue using Helm-specific names

export { HelmView as PluginView } from "./components/HelmView";
export { PLUGIN_STYLES } from "./pluginStyles";
