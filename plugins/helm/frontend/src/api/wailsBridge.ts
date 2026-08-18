// Calls the Helm plugin's HTTP backend directly over localhost,
// replacing the previous Wails InvokePlugin gRPC relay.
//
// The plugin frontend builds to a standalone ES module (see tsup.config.ts)
// and is loaded via dynamic import() from a separate bundle — it cannot
// resolve the `@wailsjs` alias, which only exists inside the main app's own
// Vite build. Since the plugin runs same-origin in the main app's window
// (see architecture notes), `window.go`/`window.runtime` are available at
// runtime regardless of which bundle the calling code came from.

import type {
  HelmChart,
  HelmChartDetail,
  HelmRelease,
  HelmReleaseDetail,
  HelmReleaseRevisionHistory,
  HelmRepository,
} from "./resources";

declare global {
  interface Window {
    go: {
      app: {
        App: {
          GetPluginBackendAddr(pluginID: string): Promise<string>;
        };
      };
    };
  }
}

// Module-level cache for backend address
let backendAddr: string | null = null;
let addressFetchPromise: Promise<string> | null = null;

export type PluginError = {
  code: string;
  message: string;
};

export function invalidateBackendAddrCache(): void {
  backendAddr = null;
  addressFetchPromise = null;
}

async function getBackendAddr(): Promise<string> {
  if (backendAddr) return backendAddr;
  if (addressFetchPromise) return addressFetchPromise;
  addressFetchPromise = window.go.app.App.GetPluginBackendAddr("helm")
    .then((addr) => {
      backendAddr = addr;
      addressFetchPromise = null;
      return addr;
    })
    .catch((err) => {
      addressFetchPromise = null;
      throw err;
    });
  return addressFetchPromise;
}

async function doFetch<T>(method: string, payload: unknown, addr: string): Promise<T> {
  const response = await fetch(`http://${addr}/api/helm/${method}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
  if (!response.ok) {
    const errBody = (await response.json()) as PluginError;
    throw errBody;
  }
  return (await response.json()) as T;
}

const fetchWithRetry = async <T>(method: string, payload: unknown): Promise<T> => {
  const addr = await getBackendAddr();
  try {
    return await doFetch<T>(method, payload, addr);
  } catch (err) {
    // Only a thrown PluginError (backend responded, just with an error body) should NOT retry.
    // A raised TypeError from fetch() itself (network/connection failure) means the cached
    // address is stale -> refetch once and retry once.
    if (err instanceof TypeError) {
      invalidateBackendAddrCache();
      try {
        const freshAddr = await getBackendAddr();
        return await doFetch<T>(method, payload, freshAddr);
      } catch {
        throw {
          code: "PLUGIN_UNAVAILABLE",
          message: "Plugin backend unreachable",
        } as PluginError;
      }
    }
    throw err;
  }
};

export const ListHelmCharts = (): Promise<HelmChart[]> =>
  fetchWithRetry<HelmChart[]>("listCharts", {});

export const ListHelmRepositories = (): Promise<HelmRepository[]> =>
  fetchWithRetry<HelmRepository[]>("listRepositories", {});

export const ListHelmReleases = (namespace: string): Promise<HelmRelease[]> =>
  fetchWithRetry<HelmRelease[]>("listReleases", { Namespace: namespace });

export const ListHelmChartVersions = (repository: string, chartName: string): Promise<string[]> =>
  fetchWithRetry<string[]>("listChartVersions", {
    Repository: repository,
    ChartName: chartName,
  });

export const GetHelmChartDetail = (
  repository: string,
  chartName: string,
  version: string
): Promise<HelmChartDetail> =>
  fetchWithRetry<HelmChartDetail>("getChartDetail", {
    Repository: repository,
    ChartName: chartName,
    Version: version,
  });

export const GetArtifactHubReadme = (
  repo: string,
  chartName: string,
  version: string
): Promise<string> =>
  fetchWithRetry<string>("getArtifactHubReadme", {
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
): Promise<void> =>
  fetchWithRetry<void>("installChart", {
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
  fetchWithRetry<void>("upgradeRelease", {
    Namespace: namespace,
    ReleaseName: releaseName,
    Repository: repository,
    ChartName: chartName,
    Version: version,
    ValuesYAML: valuesYAML,
  });

export const DeleteHelmRelease = (namespace: string, releaseName: string): Promise<void> =>
  fetchWithRetry<void>("deleteRelease", { Namespace: namespace, ReleaseName: releaseName });

export const DeleteHelmReleaseWithCleanup = (
  namespace: string,
  releaseName: string
): Promise<void> =>
  fetchWithRetry<void>("deleteReleaseWithCleanup", {
    Namespace: namespace,
    ReleaseName: releaseName,
  });

export const GetHelmReleaseByName = (
  namespace: string,
  releaseName: string
): Promise<HelmReleaseDetail> =>
  fetchWithRetry<HelmReleaseDetail>("getReleaseByName", {
    Namespace: namespace,
    ReleaseName: releaseName,
  });

export const GetHelmChartValues = (
  repository: string,
  chartName: string,
  version: string
): Promise<string> =>
  fetchWithRetry<string>("getChartValues", {
    Repository: repository,
    ChartName: chartName,
    Version: version,
  });

export const GetHelmReleaseHistory = (
  namespace: string,
  releaseName: string
): Promise<HelmReleaseRevisionHistory[]> =>
  fetchWithRetry<HelmReleaseRevisionHistory[]>("getReleaseHistory", {
    Namespace: namespace,
    ReleaseName: releaseName,
  });

export const RollbackHelmRelease = (
  namespace: string,
  releaseName: string,
  revision: number
): Promise<void> =>
  fetchWithRetry<void>("rollbackRelease", {
    Namespace: namespace,
    ReleaseName: releaseName,
    Revision: revision,
  });
