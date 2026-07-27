// Calls the main app's Wails-bound App struct directly via the global
// `window.go` bridge that Wails injects into the webview at runtime.
//
// The plugin frontend builds to a standalone ES module (see tsup.config.ts)
// and is loaded via dynamic import() from a separate bundle — it cannot
// resolve the `@wailsjs` alias, which only exists inside the main app's own
// Vite build. Since the plugin runs same-origin in the main app's window
// (see architecture notes), `window.go`/`window.runtime` are available at
// runtime regardless of which bundle the calling code came from — this file
// mirrors the shape of the generated `wailsjs/go/app/App.js` bindings by
// hand, scoped to just the RPCs the helm plugin proxies through App.
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
        App: Record<string, (...args: unknown[]) => Promise<unknown>>;
      };
    };
  }
}

const App = () => window.go.app.App;

export const ListHelmCharts = (): Promise<HelmChart[]> =>
  App().ListHelmCharts() as Promise<HelmChart[]>;

export const ListHelmRepositories = (): Promise<HelmRepository[]> =>
  App().ListHelmRepositories() as Promise<HelmRepository[]>;

export const ListHelmReleases = (namespace: string): Promise<HelmRelease[]> =>
  App().ListHelmReleases(namespace) as Promise<HelmRelease[]>;

export const ListHelmChartVersions = (repository: string, chartName: string): Promise<string[]> =>
  App().ListHelmChartVersions(repository, chartName) as Promise<string[]>;

export const GetHelmChartDetail = (
  repository: string,
  chartName: string,
  version: string
): Promise<HelmChartDetail> =>
  App().GetHelmChartDetail(repository, chartName, version) as Promise<HelmChartDetail>;

export const GetArtifactHubReadme = (
  repo: string,
  chartName: string,
  version: string
): Promise<string> => App().GetArtifactHubReadme(repo, chartName, version) as Promise<string>;

export const InstallHelmChart = (
  namespace: string,
  releaseName: string,
  repository: string,
  chartName: string,
  version: string,
  valuesYAML: string
): Promise<void> =>
  App().InstallHelmChart(
    namespace,
    releaseName,
    repository,
    chartName,
    version,
    valuesYAML
  ) as Promise<void>;

export const UpgradeHelmRelease = (
  namespace: string,
  releaseName: string,
  repository: string,
  chartName: string,
  version: string,
  valuesYAML: string
): Promise<void> =>
  App().UpgradeHelmRelease(
    namespace,
    releaseName,
    repository,
    chartName,
    version,
    valuesYAML
  ) as Promise<void>;

export const DeleteHelmRelease = (namespace: string, releaseName: string): Promise<void> =>
  App().DeleteHelmRelease(namespace, releaseName) as Promise<void>;

export const DeleteHelmReleaseWithCleanup = (
  namespace: string,
  releaseName: string
): Promise<void> => App().DeleteHelmReleaseWithCleanup(namespace, releaseName) as Promise<void>;

export const GetHelmReleaseByName = (
  namespace: string,
  releaseName: string
): Promise<HelmReleaseDetail> =>
  App().GetHelmReleaseByName(namespace, releaseName) as Promise<HelmReleaseDetail>;

export const GetHelmChartValues = (
  repository: string,
  chartName: string,
  version: string
): Promise<string> => App().GetHelmChartValues(repository, chartName, version) as Promise<string>;

export const GetHelmReleaseHistory = (
  namespace: string,
  releaseName: string
): Promise<HelmReleaseRevisionHistory[]> =>
  App().GetHelmReleaseHistory(namespace, releaseName) as Promise<HelmReleaseRevisionHistory[]>;

export const RollbackHelmRelease = (
  namespace: string,
  releaseName: string,
  revision: number
): Promise<void> => App().RollbackHelmRelease(namespace, releaseName, revision) as Promise<void>;
