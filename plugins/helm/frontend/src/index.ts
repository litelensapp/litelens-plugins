// Entry point for the ES module build used via runtime import().
// Registers this plugin's view, stylesheet, nav entry, tray families, and
// event handlers with the host instead of relying on named exports or a
// mount-time hook.
import { appWideAPI, clusterWideAPI } from "@litelens/core";
import { HelmChartsView } from "./components/chart/HelmChartsView";
import { HelmReleasesView } from "./components/release/HelmReleasesView";
import { HELM_NAV_ENTRY, HELM_TRAY_FAMILIES, PLUGIN_ID } from "./const";
import { eventHandlers } from "./events";
import { HelmView } from "./types";

appWideAPI.registerStylesheets(PLUGIN_ID, [import("./style.css")]);

clusterWideAPI.registerNavEntry(PLUGIN_ID, HELM_NAV_ENTRY);
clusterWideAPI.registerTrayFamilies(PLUGIN_ID, HELM_TRAY_FAMILIES);
clusterWideAPI.registerEvents(PLUGIN_ID, eventHandlers);
clusterWideAPI.registerViews(PLUGIN_ID, [
  { name: HelmView.HelmCharts, component: HelmChartsView },
  { name: HelmView.HelmReleases, component: HelmReleasesView },
]);
