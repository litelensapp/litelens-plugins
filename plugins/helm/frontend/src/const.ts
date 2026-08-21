import { PackageIcon, type SharedUnifiedTrayContentProps } from "@litelens/design-system";
import { NavEntry } from "@litelens/core";
import type { ComponentType } from "react";
import type { HelmViewType } from "./types";
import { HelmChartVersionTrayFamily } from "./components/chart/HelmChartVersionTrayFamily";
import { HelmChartVersionUpgradeTrayFamily } from "./components/chart/HelmChartVersionUpgradeTrayFamily";

export const HELM_NAV_ENTRY: NavEntry<HelmViewType> = {
  kind: "group",
  group: {
    id: "helm",
    label: "Helm",
    icon: PackageIcon,
    defaultOpen: true,
    items: [
      { id: "helm-charts", label: "Charts", view: "helm-charts" },
      { id: "helm-releases", label: "Releases", view: "helm-releases" },
    ],
  },
};

export const HELM_TRAY_FAMILIES: Record<string, ComponentType<SharedUnifiedTrayContentProps>> = {
  "helm-chart": HelmChartVersionTrayFamily,
  "helm-chart-upgrade": HelmChartVersionUpgradeTrayFamily,
};
