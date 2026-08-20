import { useClusterWideAPI } from "@litelens/core";
import { FC } from "react";
import { HELM_NAV_ENTRY, HELM_TRAY_FAMILIES } from "../const";
import { HelmProvider } from "../HelmContext";
import { useRegisterHelmEvents } from "../hooks/async-events/useRegisterHelmEvents";
import { HelmChartsView } from "./chart/HelmChartsView";
import { HelmReleasesView } from "./release/HelmReleasesView";

/**
 * Mounted by the host for every non-uninstalled plugin (hidden when inactive
 * — see litelens's PluginResourceView), so this runs regardless of whether
 * the user has navigated to a Helm view yet. Registering the nav entry and
 * tray families here, on the same component the host already dynamically
 * imports as PluginView, means the host never needs a separate nav/tray-only
 * contract/export.
 */
export const HelmView: FC = () => {
  const { activeResource, useRegisterNavEntry, useRegisterTrayFamilies } = useClusterWideAPI();

  useRegisterNavEntry("helm", "Helm", HELM_NAV_ENTRY);
  useRegisterTrayFamilies("helm", HELM_TRAY_FAMILIES);
  useRegisterHelmEvents();

  return (
    <HelmProvider>
      {activeResource === "helm-charts" && <HelmChartsView />}
      {activeResource === "helm-releases" && <HelmReleasesView />}
    </HelmProvider>
  );
};
