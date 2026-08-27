package helm

import (
	"errors"
	"fmt"
	"io/fs"

	"github.com/litelensapp/litelens-plugins/plugins/helm/internal/applications/dto"
	"github.com/litelensapp/litelens-plugins/plugins/helm/internal/applications/port"
	"helm.sh/helm/v3/pkg/helmpath"
	"helm.sh/helm/v3/pkg/repo"
)

// Service provides helm business logic operations.
type Service struct {
	provider      port.ClusterProvider
	emit          port.EventEmitter
	getterFactory port.RESTClientGetterFactory
}

// NewService creates a new helm Service.
func NewService(provider port.ClusterProvider, emit port.EventEmitter, getterFactory port.RESTClientGetterFactory) *Service {
	return &Service{
		provider:      provider,
		emit:          emit,
		getterFactory: getterFactory,
	}
}

// SetActiveContext updates the provider's active cluster context if it supports dynamic switching.
// Returns an error if the provider does not implement MutableClusterProvider.
func (s *Service) SetActiveContext(contextName, kubeconfigPath string) error {
	mp, ok := s.provider.(port.MutableClusterProvider)
	if !ok {
		return fmt.Errorf("helm: cluster provider does not support dynamic context switching")
	}
	return mp.SetActiveContext(contextName, kubeconfigPath)
}

// ListHelmRepositories returns the list of configured helm repositories.
func (s *Service) ListHelmRepositories() ([]dto.HelmRepository, error) {
	f, err := repo.LoadFile(helmpath.ConfigPath("repositories.yaml"))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return []dto.HelmRepository{}, nil
		}
		return []dto.HelmRepository{}, fmt.Errorf("helm: read repositories: %w", err)
	}
	result := make([]dto.HelmRepository, 0, len(f.Repositories))
	for _, r := range f.Repositories {
		result = append(result, dto.HelmRepository{Name: r.Name, URL: r.URL})
	}
	return result, nil
}
