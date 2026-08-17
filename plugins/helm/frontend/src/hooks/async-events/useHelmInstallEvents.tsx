import { renderErrorToast, renderSuccessToast } from "@litelens/design-system";
import { useQueryClient } from "@tanstack/react-query";
import { EventsOn } from "../../wailsRuntimeBridge";
import { useEffect } from "react";
import { QUERY_KEY_HELM_RELEASES } from "../../api/api.const";

interface HelmInstallPayload {
  releaseName: string;
  chartName: string;
  error?: string;
}

export const useHelmInstallEvents = () => {
  const queryClient = useQueryClient();

  useEffect(() => {
    const offComplete = EventsOn("helm:install:complete", (payload: unknown) => {
      const { releaseName, chartName } = payload as HelmInstallPayload;
      renderSuccessToast({
        title: "Chart installed",
        description: releaseName
          ? `${chartName} installed as ${releaseName}`
          : `${chartName} installed`,
      });
      queryClient.invalidateQueries({ queryKey: [QUERY_KEY_HELM_RELEASES] });
    });

    const offError = EventsOn("helm:install:error", (payload: unknown) => {
      const { releaseName, chartName, error } = payload as HelmInstallPayload;
      renderErrorToast({
        title: "Failed to install chart",
        description: releaseName
          ? `${chartName} (${releaseName}): ${error}`
          : `${chartName}: ${error}`,
      });
    });

    return () => {
      offComplete();
      offError();
    };
  }, [queryClient]);
};
