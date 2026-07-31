import { FC } from "react";
import type { SharedUnifiedTrayContentProps } from "@litelens/design-system";
import { HelmChartVersionTray, type HelmChartVersionTrayTab } from "./HelmChartVersionTray";

export const HelmChartVersionTrayFamily: FC<SharedUnifiedTrayContentProps> = ({
  tab,
  collapsed,
  onClose,
}) => {
  if (tab.family !== "helm-chart") {
    return null;
  }

  return (
    <HelmChartVersionTray
      tab={{ id: tab.id, ...tab.params } as HelmChartVersionTrayTab}
      collapsed={collapsed}
      onClose={onClose}
    />
  );
};
