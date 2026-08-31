export {
  AddHelmRepository,
  DeleteHelmRelease,
  DeleteHelmReleaseWithCleanup,
  GetArtifactHubReadme,
  GetHelmChartDetail,
  GetHelmChartValues,
  GetHelmReleaseByName,
  GetHelmReleaseHistory,
  InstallHelmChart,
  ListHelmCharts,
  ListHelmChartVersions,
  ListHelmReleases,
  ListHelmRepositories,
  RemoveHelmRepository,
  RollbackHelmRelease,
  SearchHelmRepositoryCatalog,
  UpgradeHelmRelease as UpgradeHelmChart,
} from "./bridge";

export interface HelmChart {
  Name: string;
  Description: string;
  Version: string;
  AppVersion: string;
  Repository: string;
  Icon: string;
}

export interface HelmChartDetail {
  Name: string;
  Description: string;
  Version: string;
  AppVersion: string;
  Repository: string;
  Icon: string;
  Home: string;
  Keywords: string[];
  Sources: string[];
  Maintainers: string[];
}

export interface HelmRepository {
  Name: string;
  URL: string;
}

export interface HelmRepositoryCatalogEntry {
  Name: string;
  URL: string;
}

export interface HelmRepositoryCatalogPage {
  Entries: HelmRepositoryCatalogEntry[];
  HasMore: boolean;
}

export interface HelmRelease {
  Name: string;
  Namespace: string;
  Chart: string;
  ChartVersion: string;
  AppVersion: string;
  Status: string;
  Revision: number;
  Updated: string;
  UpdatedAt: string;
  Repository: string;
  /** gzip-compressed, base64-encoded YAML — decode with decodeValuesYAML() before use */
  EncodedValuesYAML: string;
}

export interface HelmReleaseResource {
  Kind: string;
  Name: string;
  Namespace: string;
}

export interface HelmReleaseDetail {
  Name: string;
  Namespace: string;
  Chart: string;
  ChartVersion: string;
  AppVersion: string;
  Status: string;
  Revision: number;
  Updated: string;
  UpdatedAt: string;
  Notes: string;
  ValuesYAML: string;
  Resources: HelmReleaseResource[];
  Repository: string;
}

export interface HelmReleaseRevisionHistory {
  Revision: number;
  ChartVersion: string;
  AppVersion: string;
  Status: string;
  UpdatedAt: string;
}
