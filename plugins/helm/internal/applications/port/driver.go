package port

import "github.com/litelensapp/litelens-plugins/plugins/helm/internal/applications/dto"

// HelmService interface matches the methods we need from helm.Service.
// We define this in the port package to avoid circular imports.
type HelmService interface {
	ListHelmCharts() ([]dto.HelmChart, error)
	ListHelmRepositories() ([]dto.HelmRepository, error)
	ListHelmReleases() ([]dto.HelmRelease, error)
	ListHelmChartVersions(repository, chartName string) ([]string, error)
	GetHelmChartDetail(repository, chartName, version string) (dto.HelmChartDetail, error)
	GetArtifactHubReadme(repository, chartName, version string) (string, error)
	InstallHelmChart(namespace, releaseName, repository, chartName, version, valuesYAML string) (string, error)
	UpgradeHelmRelease(namespace, releaseName, repository, chartName, version, valuesYAML string) error
	DeleteHelmRelease(namespace, releaseName string) error
	DeleteHelmReleaseWithCleanup(namespace, releaseName string) error
	GetHelmReleaseByName(namespace, releaseName string) (*dto.HelmReleaseDetail, error)
	GetHelmChartValues(repository, chartName, version string) (string, error)
	GetHelmReleaseHistory(namespace, releaseName string) ([]dto.HelmReleaseRevisionHistory, error)
	RollbackHelmRelease(namespace, releaseName string, revision int) error
	SetActiveContext(contextName, kubeconfigPath string) error
}
