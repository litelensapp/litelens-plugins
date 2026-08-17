import { FC, Suspense, lazy } from "react";
import { LoadingSpinner, type SharedUnifiedTrayContentProps } from "@litelens/design-system";
import type { HelmChartVersionUpgradeTrayTab } from "./HelmChartVersionUpgradeTray";

// Lazy-loaded so this tray's code ships as a separate chunk, only fetched
// when a user actually opens an upgrade tab instead of being bundled into
// the plugin's eagerly-loaded entry chunk.
const HelmChartVersionUpgradeTray = lazy(() =>
  import("./HelmChartVersionUpgradeTray").then((m) => ({
    default: m.HelmChartVersionUpgradeTray,
  }))
);

export const HelmChartVersionUpgradeTrayFamily: FC<SharedUnifiedTrayContentProps> = ({
  tab,
  collapsed,
  onClose,
}) => {
  if (tab.family !== "helm-chart-upgrade") {
    return null;
  }

  return (
    <Suspense fallback={<LoadingSpinner />}>
      <HelmChartVersionUpgradeTray
        tab={{ id: tab.id, ...tab.params } as HelmChartVersionUpgradeTrayTab}
        collapsed={collapsed}
        onClose={onClose}
      />
    </Suspense>
  );
};
