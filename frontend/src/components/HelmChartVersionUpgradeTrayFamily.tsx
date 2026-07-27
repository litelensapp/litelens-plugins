import { FC } from "react";
import type { SharedUnifiedTrayContentProps } from "@litelens/design-system";
import {
  HelmChartVersionUpgradeTray,
  type HelmChartVersionUpgradeTrayTab,
} from "./HelmChartVersionUpgradeTray";

export const HelmChartVersionUpgradeTrayFamily: FC<SharedUnifiedTrayContentProps> = ({
  tab,
  collapsed,
  onClose,
}) => {
  if (tab.family !== "helm-chart-upgrade") {
    return null;
  }

  return (
    <HelmChartVersionUpgradeTray
      tab={{ id: tab.id, ...tab.params } as HelmChartVersionUpgradeTrayTab}
      collapsed={collapsed}
      onClose={onClose}
    />
  );
};
