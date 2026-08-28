import { renderErrorToast, renderSuccessToast } from "@litelens/design-system";
import { useMutation } from "@tanstack/react-query";
import { QUERY_KEY_HELM_RELEASES } from "../../api/api.const";
import { queryClient } from "../../api/query.client";
import { InstallHelmChart, type HelmRelease } from "../../api/resources";

export const useInstallHelmChart = (options?: { onNavigateToReleases?: () => void }) => {
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
    onSuccess: ({ ReleaseName }, { namespace, repository, chartName, version }) => {
      // The Go handler only kicks off a goroutine that calls install.RunWithContext
      // and returns immediately, before Helm has persisted the pending-install
      // release record (that happens deep inside RunWithContext) — so a normal
      // invalidate+refetch here would race the goroutine and return the
      // pre-install list. Instead, seed the cache directly with a synthetic
      // pending-install entry using the release name the backend already
      // resolved (generated one if the user left it blank). The definitive
      // reconciliation happens off "helm:install:complete"/"helm:install:error"
      // (see events.ts), which invalidate for real once Helm reaches
      // deployed/failed.
      const optimisticRelease: HelmRelease = {
        Name: ReleaseName,
        Namespace: namespace,
        Chart: chartName,
        ChartVersion: version,
        AppVersion: "",
        Status: "pending-install",
        Revision: 1,
        Updated: "just now",
        UpdatedAt: new Date().toISOString(),
        Repository: repository,
        EncodedValuesYAML: "",
      };
      queryClient.setQueriesData<HelmRelease[]>({ queryKey: [QUERY_KEY_HELM_RELEASES] }, (old) => [
        optimisticRelease,
        ...(old?.filter((r) => r.Name !== ReleaseName) ?? []),
      ]);

      renderSuccessToast({
        title: "Install started",
        description: `${chartName} is being installed as ${ReleaseName}`,
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
