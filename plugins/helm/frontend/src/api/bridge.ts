// Calls the Helm plugin's HTTP backend directly over localhost,
// replacing the previous Wails InvokePlugin gRPC relay.
//
// The fetch/retry/backend-address-caching machinery is shared across plugins
// via @litelens/core's createPluginBridge — see that module for the
// `window.go`/same-origin rationale. Only the per-endpoint payload shapes
// below are helm-specific.

import { createPluginBridge } from "@litelens/core";
import { PLUGIN_ID } from "../const";
import type {
  HelmChart,
  HelmChartDetail,
  HelmRelease,
  HelmReleaseDetail,
  HelmReleaseRevisionHistory,
  HelmRepository,
} from "./resources";

export type { PluginError } from "@litelens/core";

// Exported only so tests can reset the cached backend address between cases
// (bridge.invalidateBackendAddrCache) — no application code needs it, since
// fetchWithRetry already self-heals on a stale-address TypeError.
export const bridge = createPluginBridge(PLUGIN_ID);

export const ListHelmCharts = (): Promise<HelmChart[]> =>
  bridge.fetchWithRetry<HelmChart[]>("listCharts", {});

export const ListHelmRepositories = (): Promise<HelmRepository[]> =>
  bridge.fetchWithRetry<HelmRepository[]>("listRepositories", {});

export const ListHelmReleases = (): Promise<HelmRelease[]> =>
  bridge.fetchWithRetry<HelmRelease[]>("listReleases", {});

export const ListHelmChartVersions = (repository: string, chartName: string): Promise<string[]> =>
  bridge.fetchWithRetry<string[]>("listChartVersions", {
    Repository: repository,
    ChartName: chartName,
  });

export const GetHelmChartDetail = (
  repository: string,
  chartName: string,
  version: string
): Promise<HelmChartDetail> =>
  bridge.fetchWithRetry<HelmChartDetail>("getChartDetail", {
    Repository: repository,
    ChartName: chartName,
    Version: version,
  });

export const GetArtifactHubReadme = (
  repo: string,
  chartName: string,
  version: string
): Promise<string> =>
  bridge.fetchWithRetry<string>("getArtifactHubReadme", {
    Repository: repo,
    ChartName: chartName,
    Version: version,
  });

export const InstallHelmChart = (
  namespace: string,
  releaseName: string,
  repository: string,
  chartName: string,
  version: string,
  valuesYAML: string
): Promise<{ ReleaseName: string }> =>
  bridge.fetchWithRetry<{ ReleaseName: string }>("installChart", {
    Namespace: namespace,
    ReleaseName: releaseName,
    Repository: repository,
    ChartName: chartName,
    Version: version,
    ValuesYAML: valuesYAML,
  });

export const UpgradeHelmRelease = (
  namespace: string,
  releaseName: string,
  repository: string,
  chartName: string,
  version: string,
  valuesYAML: string
): Promise<void> =>
  bridge.fetchWithRetry<void>("upgradeRelease", {
    Namespace: namespace,
    ReleaseName: releaseName,
    Repository: repository,
    ChartName: chartName,
    Version: version,
    ValuesYAML: valuesYAML,
  });

export const DeleteHelmRelease = (namespace: string, releaseName: string): Promise<void> =>
  bridge.fetchWithRetry<void>("deleteRelease", { Namespace: namespace, ReleaseName: releaseName });

export const DeleteHelmReleaseWithCleanup = (
  namespace: string,
  releaseName: string
): Promise<void> =>
  bridge.fetchWithRetry<void>("deleteReleaseWithCleanup", {
    Namespace: namespace,
    ReleaseName: releaseName,
  });

export const GetHelmReleaseByName = (
  namespace: string,
  releaseName: string
): Promise<HelmReleaseDetail> =>
  bridge.fetchWithRetry<HelmReleaseDetail>("getReleaseByName", {
    Namespace: namespace,
    ReleaseName: releaseName,
  });

export const GetHelmChartValues = (
  repository: string,
  chartName: string,
  version: string
): Promise<string> =>
  bridge.fetchWithRetry<string>("getChartValues", {
    Repository: repository,
    ChartName: chartName,
    Version: version,
  });

export const GetHelmReleaseHistory = (
  namespace: string,
  releaseName: string
): Promise<HelmReleaseRevisionHistory[]> =>
  bridge.fetchWithRetry<HelmReleaseRevisionHistory[]>("getReleaseHistory", {
    Namespace: namespace,
    ReleaseName: releaseName,
  });

export const RollbackHelmRelease = (
  namespace: string,
  releaseName: string,
  revision: number
): Promise<void> =>
  bridge.fetchWithRetry<void>("rollbackRelease", {
    Namespace: namespace,
    ReleaseName: releaseName,
    Revision: revision,
  });
