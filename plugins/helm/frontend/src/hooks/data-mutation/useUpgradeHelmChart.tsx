import { renderErrorToast, renderSuccessToast } from "@litelens/design-system";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { QUERY_KEY_HELM_RELEASE_DETAIL, QUERY_KEY_HELM_RELEASES } from "../../api/api.const";
import { UpgradeHelmChart } from "../../api/resources";

export const useUpgradeHelmChart = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      namespace,
      releaseName,
      repository,
      chartName,
      version,
      valuesYAML = "",
    }: {
      namespace: string;
      releaseName: string;
      repository: string;
      chartName: string;
      version: string;
      valuesYAML?: string;
    }) => UpgradeHelmChart(namespace, releaseName, repository, chartName, version, valuesYAML),
    onSuccess: (_, { chartName, releaseName }) => {
      queryClient.invalidateQueries({ queryKey: [QUERY_KEY_HELM_RELEASES] });
      queryClient.invalidateQueries({ queryKey: [QUERY_KEY_HELM_RELEASE_DETAIL] });
      renderSuccessToast({
        title: "Upgrade started",
        description: `${chartName} is being upgraded in ${releaseName}`,
      });
    },
    onError: (err, { chartName, releaseName }) =>
      renderErrorToast({
        title: "Failed to start upgrade",
        description: `${chartName} (${releaseName}): ${String(err)}`,
      }),
  });
};
