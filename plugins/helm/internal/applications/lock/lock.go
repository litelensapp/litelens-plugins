package lock

import (
	"sync"

	"github.com/litelensapp/litelens-plugins/plugins/helm/internal/applications/dto"
	"github.com/litelensapp/litelens-plugins/plugins/helm/internal/applications/helm"
)

// LockedService wraps Service with a mutex guarding the active cluster client so a
// context switch (SetActiveContext) can never race with an in-flight business call
// still using the old client. Business methods take the read lock — any number can
// run concurrently. SetActiveContext takes the write lock, which sync.RWMutex already
// blocks on until all outstanding read-locked calls finish, then swaps the client.
// New business calls arriving during the swap simply block briefly on RLock().
type LockedService struct {
	mu  sync.RWMutex
	svc *helm.Service
}

// NewService wraps an existing Service.
func NewService(svc *helm.Service) *LockedService {
	return &LockedService{svc: svc}
}

func (l *LockedService) SetActiveContext(contextName, kubeconfigPath string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.svc.SetActiveContext(contextName, kubeconfigPath)
}

func (l *LockedService) ListHelmCharts() ([]dto.HelmChart, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.svc.ListHelmCharts()
}

func (l *LockedService) ListHelmRepositories() ([]dto.HelmRepository, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.svc.ListHelmRepositories()
}

func (l *LockedService) ListHelmReleases() ([]dto.HelmRelease, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.svc.ListHelmReleases()
}

func (l *LockedService) ListHelmChartVersions(repository, chartName string) ([]string, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.svc.ListHelmChartVersions(repository, chartName)
}

func (l *LockedService) GetHelmChartDetail(repository, chartName, version string) (dto.HelmChartDetail, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.svc.GetHelmChartDetail(repository, chartName, version)
}

func (l *LockedService) GetArtifactHubReadme(repository, chartName, version string) (string, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.svc.GetArtifactHubReadme(repository, chartName, version)
}

func (l *LockedService) InstallHelmChart(namespace, releaseName, repository, chartName, version, valuesYAML string) (string, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.svc.InstallHelmChart(namespace, releaseName, repository, chartName, version, valuesYAML)
}

func (l *LockedService) UpgradeHelmRelease(namespace, releaseName, repository, chartName, version, valuesYAML string) error {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.svc.UpgradeHelmRelease(namespace, releaseName, repository, chartName, version, valuesYAML)
}

func (l *LockedService) DeleteHelmRelease(namespace, releaseName string) error {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.svc.DeleteHelmRelease(namespace, releaseName)
}

func (l *LockedService) DeleteHelmReleaseWithCleanup(namespace, releaseName string) error {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.svc.DeleteHelmReleaseWithCleanup(namespace, releaseName)
}

func (l *LockedService) GetHelmReleaseByName(namespace, releaseName string) (*dto.HelmReleaseDetail, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.svc.GetHelmReleaseByName(namespace, releaseName)
}

func (l *LockedService) GetHelmChartValues(repository, chartName, version string) (string, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.svc.GetHelmChartValues(repository, chartName, version)
}

func (l *LockedService) GetHelmReleaseHistory(namespace, releaseName string) ([]dto.HelmReleaseRevisionHistory, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.svc.GetHelmReleaseHistory(namespace, releaseName)
}

func (l *LockedService) RollbackHelmRelease(namespace, releaseName string, revision int) error {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.svc.RollbackHelmRelease(namespace, releaseName, revision)
}
