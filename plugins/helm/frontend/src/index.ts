// Barrel export for plugin dynamic loading via import()
// This is the entry point for the ES module build used via runtime import()
// External consumers use the generic contract names (e.g. PLUGIN_NAV_ENTRY, PluginView)
// Internal Helm components continue using Helm-specific names

export { HelmView as PluginView } from "./components/HelmView";
export { HELM_NAV_ENTRY as PLUGIN_NAV_ENTRY } from "./const";
export { HelmEventBridge as PluginEventBridge } from "./components/HelmEventBridge";
export { PLUGIN_STYLES } from "./pluginStyles";
export type { HelmViewType } from "./types";

// Tray-family content components, keyed by the family name the plugin uses
// when calling `unifiedTray.openTab(family, params)`. The host discovers
// these at runtime and never has static knowledge of the family names or
// their param shapes.
import { HelmChartVersionTrayFamily } from "./components/chart/HelmChartVersionTrayFamily";
import { HelmChartVersionUpgradeTrayFamily } from "./components/chart/HelmChartVersionUpgradeTrayFamily";

export const PLUGIN_TRAY_FAMILIES = {
  "helm-chart": HelmChartVersionTrayFamily,
  "helm-chart-upgrade": HelmChartVersionUpgradeTrayFamily,
};
