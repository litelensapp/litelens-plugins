import { renderErrorToast, renderSuccessToast } from "@litelens/design-system";
import { useQueryClient } from "@tanstack/react-query";
import { EventsOn } from "../../wailsRuntimeBridge";
import { useEffect } from "react";
import { QUERY_KEY_HELM_RELEASE_DETAIL, QUERY_KEY_HELM_RELEASES } from "../../api/api.const";

interface HelmUpgradePayload {
  releaseName: string;
  chartName: string;
  error?: string;
}

export const useHelmUpgradeEvents = () => {
  const queryClient = useQueryClient();

  useEffect(() => {
    const offComplete = EventsOn("helm:upgrade:complete", (payload: unknown) => {
      const { releaseName, chartName } = payload as HelmUpgradePayload;
      renderSuccessToast({
        title: "Chart upgraded",
        description: `${chartName} upgraded in ${releaseName}`,
      });
      queryClient.invalidateQueries({ queryKey: [QUERY_KEY_HELM_RELEASES] });
      queryClient.invalidateQueries({ queryKey: [QUERY_KEY_HELM_RELEASE_DETAIL] });
    });

    const offError = EventsOn("helm:upgrade:error", (payload: unknown) => {
      const { releaseName, chartName, error } = payload as HelmUpgradePayload;
      renderErrorToast({
        title: "Failed to upgrade chart",
        description: `${chartName} (${releaseName}): ${error}`,
      });
    });

    return () => {
      offComplete();
      offError();
    };
  }, [queryClient]);
};
