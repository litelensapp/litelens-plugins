// Calls the main app's Wails-bound App struct through the generic InvokePlugin
// bridge that Wails injects into the webview at runtime. This plugin now uses
// the same gRPC Invoke protocol as any subprocess plugin would.
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
          InvokePlugin(pluginID: string, method: string, payloadJson: string): Promise<string>;
        };
      };
    };
  }
}

const App = () => window.go.app.App;

// Helper to invoke a plugin method and parse the JSON response
const invoke = async <T>(method: string, payload: unknown): Promise<T> => {
  const raw = await App().InvokePlugin("helm", method, JSON.stringify(payload));
  return JSON.parse(raw as string) as T;
};

export const ListHelmCharts = (): Promise<HelmChart[]> => invoke<HelmChart[]>("ListHelmCharts", {});

export const ListHelmRepositories = (): Promise<HelmRepository[]> =>
  invoke<HelmRepository[]>("ListHelmRepositories", {});

export const ListHelmReleases = (namespace: string): Promise<HelmRelease[]> =>
  invoke<HelmRelease[]>("ListHelmReleases", { Namespace: namespace });

export const ListHelmChartVersions = (repository: string, chartName: string): Promise<string[]> =>
  invoke<string[]>("ListHelmChartVersions", { Repository: repository, ChartName: chartName });

export const GetHelmChartDetail = (
  repository: string,
  chartName: string,
  version: string
): Promise<HelmChartDetail> =>
  invoke<HelmChartDetail>("GetHelmChartDetail", {
    Repository: repository,
    ChartName: chartName,
    Version: version,
  });

export const GetArtifactHubReadme = (
  repo: string,
  chartName: string,
  version: string
): Promise<string> =>
  invoke<string>("GetArtifactHubReadme", {
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
  invoke<void>("InstallHelmChart", {
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
  invoke<void>("UpgradeHelmRelease", {
    Namespace: namespace,
    ReleaseName: releaseName,
    Repository: repository,
    ChartName: chartName,
    Version: version,
    ValuesYAML: valuesYAML,
  });

export const DeleteHelmRelease = (namespace: string, releaseName: string): Promise<void> =>
  invoke<void>("DeleteHelmRelease", { Namespace: namespace, ReleaseName: releaseName });

export const DeleteHelmReleaseWithCleanup = (
  namespace: string,
  releaseName: string
): Promise<void> =>
  invoke<void>("DeleteHelmReleaseWithCleanup", { Namespace: namespace, ReleaseName: releaseName });

export const GetHelmReleaseByName = (
  namespace: string,
  releaseName: string
): Promise<HelmReleaseDetail> =>
  invoke<HelmReleaseDetail>("GetHelmReleaseByName", {
    Namespace: namespace,
    ReleaseName: releaseName,
  });

export const GetHelmChartValues = (
  repository: string,
  chartName: string,
  version: string
): Promise<string> =>
  invoke<string>("GetHelmChartValues", {
    Repository: repository,
    ChartName: chartName,
    Version: version,
  });

export const GetHelmReleaseHistory = (
  namespace: string,
  releaseName: string
): Promise<HelmReleaseRevisionHistory[]> =>
  invoke<HelmReleaseRevisionHistory[]>("GetHelmReleaseHistory", {
    Namespace: namespace,
    ReleaseName: releaseName,
  });

export const RollbackHelmRelease = (
  namespace: string,
  releaseName: string,
  revision: number
): Promise<void> =>
  invoke<void>("RollbackHelmRelease", {
    Namespace: namespace,
    ReleaseName: releaseName,
    Revision: revision,
  });
