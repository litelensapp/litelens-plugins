import { renderErrorToast, renderSuccessToast } from "@litelens/design-system";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { QUERY_KEY_HELM_RELEASES } from "../../api/api.const";
import { InstallHelmChart } from "../../api/resources";

export const useInstallHelmChart = (options?: { onNavigateToReleases?: () => void }) => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      namespace,
      releaseName = "",
      repository,
      chartName,
      version,
      valuesYAML = "",
    }: {
      namespace: string;
      releaseName?: string;
      repository: string;
      chartName: string;
      version: string;
      valuesYAML?: string;
    }) => InstallHelmChart(namespace, releaseName, repository, chartName, version, valuesYAML),
    onSuccess: (_, { releaseName, chartName }) => {
      // Go returned — goroutine is running. Invalidate so HelmReleasesView shows
      // the release at pending-install immediately.
      queryClient.invalidateQueries({ queryKey: [QUERY_KEY_HELM_RELEASES] });
      renderSuccessToast({
        title: "Install started",
        description: releaseName
          ? `${chartName} is being installed as ${releaseName}`
          : `${chartName} is being installed`,
        action: options?.onNavigateToReleases
          ? { label: "View Releases", onClick: options.onNavigateToReleases }
          : undefined,
      });
    },
    onError: (err, { chartName, releaseName }) =>
      renderErrorToast({
        title: "Failed to start install",
        description: releaseName
          ? `${chartName} (${releaseName}): ${String(err)}`
          : `${chartName}: ${String(err)}`,
      }),
  });
};
