import { FC, Suspense, lazy } from "react";
import { LoadingSpinner, type SharedUnifiedTrayContentProps } from "@litelens/design-system";
import type { HelmChartVersionTrayTab } from "./HelmChartVersionTray";

// Lazy-loaded so this tray's code (and its data-fetching hooks) ships as a
// separate chunk, only fetched when a user actually opens a chart install tab
// instead of being bundled into the plugin's eagerly-loaded entry chunk.
const HelmChartVersionTray = lazy(() =>
  import("./HelmChartVersionTray").then((m) => ({ default: m.HelmChartVersionTray }))
);

export const HelmChartVersionTrayFamily: FC<SharedUnifiedTrayContentProps> = ({
  tab,
  collapsed,
  onClose,
}) => {
  if (tab.family !== "helm-chart") {
    return null;
  }

  return (
    <Suspense fallback={<LoadingSpinner />}>
      <HelmChartVersionTray
        tab={{ id: tab.id, ...tab.params } as HelmChartVersionTrayTab}
        collapsed={collapsed}
        onClose={onClose}
      />
    </Suspense>
  );
};
