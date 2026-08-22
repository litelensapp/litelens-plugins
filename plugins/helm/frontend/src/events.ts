import { appWideAPI } from "@litelens/core";
import { renderErrorToast, renderSuccessToast } from "@litelens/design-system";
import { QUERY_KEY_HELM_RELEASE_DETAIL, QUERY_KEY_HELM_RELEASES } from "./api/api.const";

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

const queryClient = appWideAPI.getQueryClient();

export const eventHandlers = {
  "helm:install:complete": (payload: HelmInstallPayload) => {
    const { releaseName, chartName } = payload;
    renderSuccessToast({
      title: "Chart installed",
      description: releaseName
        ? `${chartName} installed as ${releaseName}`
        : `${chartName} installed`,
    });
    queryClient.invalidateQueries({ queryKey: [QUERY_KEY_HELM_RELEASES] });
  },
  "helm:install:error": (payload: HelmInstallPayload) => {
    const { releaseName, chartName, error } = payload;
    renderErrorToast({
      title: "Failed to install chart",
      description: releaseName
        ? `${chartName} (${releaseName}): ${error}`
        : `${chartName}: ${error}`,
    });
  },
  "helm:upgrade:complete": (payload: HelmUpgradePayload) => {
    const { releaseName, chartName } = payload;
    renderSuccessToast({
      title: "Chart upgraded",
      description: `${chartName} upgraded in ${releaseName}`,
    });
    queryClient.invalidateQueries({ queryKey: [QUERY_KEY_HELM_RELEASES] });
    queryClient.invalidateQueries({ queryKey: [QUERY_KEY_HELM_RELEASE_DETAIL] });
  },
  "helm:upgrade:error": (payload: HelmUpgradePayload) => {
    const { releaseName, chartName, error } = payload;
    renderErrorToast({
      title: "Failed to upgrade chart",
      description: `${chartName} (${releaseName}): ${error}`,
    });
  },
  "helm:cleanup:complete": (payload: CleanupCompletePayload) => {
    const { releaseName, deleted } = payload;
    const description =
      deleted > 0
        ? `${deleted} resource${deleted === 1 ? "" : "s"} cleaned up for ${releaseName}`
        : `No leftover resources found for ${releaseName}`;
    renderSuccessToast({ title: "Resources cleaned up", description: description });
    queryClient.invalidateQueries({ queryKey: [QUERY_KEY_HELM_RELEASES] });
  },
  "helm:cleanup:partial": (payload: CleanupPartialPayload) => {
    const { releaseName, deleted, errors } = payload;
    renderSuccessToast({
      title: "Cleanup partially complete",
      description: `${deleted} resource${deleted === 1 ? "" : "s"} removed for ${releaseName}; ${errors.length} failed to delete`,
    });
    queryClient.invalidateQueries({ queryKey: [QUERY_KEY_HELM_RELEASES] });
  },
  "helm:cleanup:error": (payload: CleanupErrorPayload) => {
    const { releaseName, error } = payload;
    renderErrorToast({
      title: "Resource cleanup failed",
      description: `Could not clean up resources for ${releaseName}: ${error}`,
    });
  },
};
