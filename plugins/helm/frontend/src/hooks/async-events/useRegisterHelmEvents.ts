import { useClusterWideAPI } from "@litelens/core";
import { renderErrorToast, renderSuccessToast } from "@litelens/design-system";
import { useQueryClient } from "@tanstack/react-query";
import { useMemo } from "react";
import { QUERY_KEY_HELM_RELEASE_DETAIL, QUERY_KEY_HELM_RELEASES } from "../../api/api.const";

interface HelmInstallPayload {
  releaseName: string;
  chartName: string;
  error?: string;
}

interface HelmUpgradePayload {
  releaseName: string;
  chartName: string;
  error?: string;
}

interface CleanupCompletePayload {
  releaseName: string;
  deleted: number;
}

interface CleanupPartialPayload {
  releaseName: string;
  deleted: number;
  errors: string[];
}

interface CleanupErrorPayload {
  releaseName: string;
  error: string;
}

export function useRegisterHelmEvents(): void {
  const { useRegisterClusterWideEvents } = useClusterWideAPI();
  const queryClient = useQueryClient();

  const eventHandlers = useMemo(
    () => ({
      "helm:install:complete": (payload: unknown) => {
        const { releaseName, chartName } = payload as HelmInstallPayload;
        renderSuccessToast({
          title: "Chart installed",
          description: releaseName
            ? `${chartName} installed as ${releaseName}`
            : `${chartName} installed`,
        });
        queryClient.invalidateQueries({ queryKey: [QUERY_KEY_HELM_RELEASES] });
      },
      "helm:install:error": (payload: unknown) => {
        const { releaseName, chartName, error } = payload as HelmInstallPayload;
        renderErrorToast({
          title: "Failed to install chart",
          description: releaseName
            ? `${chartName} (${releaseName}): ${error}`
            : `${chartName}: ${error}`,
        });
      },
      "helm:upgrade:complete": (payload: unknown) => {
        const { releaseName, chartName } = payload as HelmUpgradePayload;
        renderSuccessToast({
          title: "Chart upgraded",
          description: `${chartName} upgraded in ${releaseName}`,
        });
        queryClient.invalidateQueries({ queryKey: [QUERY_KEY_HELM_RELEASES] });
        queryClient.invalidateQueries({ queryKey: [QUERY_KEY_HELM_RELEASE_DETAIL] });
      },
      "helm:upgrade:error": (payload: unknown) => {
        const { releaseName, chartName, error } = payload as HelmUpgradePayload;
        renderErrorToast({
          title: "Failed to upgrade chart",
          description: `${chartName} (${releaseName}): ${error}`,
        });
      },
      "helm:cleanup:complete": (payload: unknown) => {
        const { releaseName, deleted } = payload as CleanupCompletePayload;
        const description =
          deleted > 0
            ? `${deleted} resource${deleted === 1 ? "" : "s"} cleaned up for ${releaseName}`
            : `No leftover resources found for ${releaseName}`;
        renderSuccessToast({ title: "Resources cleaned up", description: description });
        queryClient.invalidateQueries({ queryKey: [QUERY_KEY_HELM_RELEASES] });
      },
      "helm:cleanup:partial": (payload: unknown) => {
        const { releaseName, deleted, errors } = payload as CleanupPartialPayload;
        renderSuccessToast({
          title: "Cleanup partially complete",
          description: `${deleted} resource${deleted === 1 ? "" : "s"} removed for ${releaseName}; ${errors.length} failed to delete`,
        });
        queryClient.invalidateQueries({ queryKey: [QUERY_KEY_HELM_RELEASES] });
      },
      "helm:cleanup:error": (payload: unknown) => {
        const { releaseName, error } = payload as CleanupErrorPayload;
        renderErrorToast({
          title: "Resource cleanup failed",
          description: `Could not clean up resources for ${releaseName}: ${error}`,
        });
      },
    }),
    [queryClient]
  );

  useRegisterClusterWideEvents(eventHandlers);
}
