import { renderErrorToast, renderSuccessToast } from "@litelens/design-system";
import { useMutation } from "@tanstack/react-query";
import { QUERY_KEY_HELM_RELEASE_DETAIL, QUERY_KEY_HELM_RELEASES } from "../../api/api.const";
import { queryClient } from "../../api/query.client";
import { UpgradeHelmChart, type HelmRelease, type HelmReleaseDetail } from "../../api/resources";

export const useUpgradeHelmChart = () => {
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
    onSuccess: (_, { namespace, chartName, releaseName }) => {
      // Don't invalidate+refetch here: the Go handler kicks off a goroutine that
      // calls upgrade.RunWithContext and returns immediately, before Helm has
      // persisted the new release revision — a refetch now would race that
      // goroutine and return stale data. Instead, flip the existing release's
      // status to pending-upgrade directly in the cache. The definitive
      // reconciliation happens off "helm:upgrade:complete"/"helm:upgrade:error"
      // (see events.ts), which invalidate for real once Helm reaches
      // deployed/failed.
      queryClient.setQueriesData<HelmRelease[]>({ queryKey: [QUERY_KEY_HELM_RELEASES] }, (old) =>
        old?.map((r) => (r.Name === releaseName ? { ...r, Status: "pending-upgrade" } : r))
      );
      queryClient.setQueriesData<HelmReleaseDetail>(
        {
          queryKey: [QUERY_KEY_HELM_RELEASE_DETAIL],
          predicate: (query) =>
            query.queryKey[2] === namespace && query.queryKey[3] === releaseName,
        },
        (old) => (old ? { ...old, Status: "pending-upgrade" } : old)
      );

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
