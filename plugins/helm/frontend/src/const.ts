import { NavEntry, SharedUnifiedTrayContentProps } from "@litelens/core";
import { PackageIcon } from "@litelens/design-system";
import type { ComponentType } from "react";
import { HelmChartVersionTrayFamily } from "./components/chart/HelmChartVersionTrayFamily";
import { HelmChartVersionUpgradeTrayFamily } from "./components/chart/HelmChartVersionUpgradeTrayFamily";
import { HelmView, type HelmViewType } from "./types";

export const PLUGIN_ID = "helm";

export const HELM_NAV_ENTRY: NavEntry<HelmViewType> = {
  kind: "group",
  group: {
    id: PLUGIN_ID,
    label: "Helm",
    icon: PackageIcon,
    defaultOpen: true,
    items: [
      { id: HelmView.HelmCharts, label: "Charts", view: HelmView.HelmCharts },
      { id: HelmView.HelmReleases, label: "Releases", view: HelmView.HelmReleases },
    ],
  },
};

export const HELM_TRAY_FAMILIES: Record<string, ComponentType<SharedUnifiedTrayContentProps>> = {
  "helm-chart": HelmChartVersionTrayFamily,
  "helm-chart-upgrade": HelmChartVersionUpgradeTrayFamily,
};
