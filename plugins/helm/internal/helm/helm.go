package helm

import (
	"context"
	"fmt"
	"os"

	"github.com/litelensapp/litelens-plugins/plugins/helm/internal/dto"
	"helm.sh/helm/v3/pkg/helmpath"
	"helm.sh/helm/v3/pkg/repo"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// EventEmitter is a callback function to emit events from the helm service.
type EventEmitter func(ctx context.Context, eventName string, data any)

// ClusterProvider provides access to active cluster clients and configuration.
type ClusterProvider interface {
	ActiveClients() (cs *kubernetes.Clientset, rc *rest.Config, activeContext string, kubeconfigPaths []string)
	Ctx() context.Context
}

// MutableClusterProvider is a ClusterProvider whose active context can be changed
// after construction — implemented by the plugin subprocess's dynamic provider so
// the app can sync it to whatever cluster context is currently active, per call.
type MutableClusterProvider interface {
	ClusterProvider
	SetActiveContext(contextName, kubeconfigPath string) error
}

// Service provides helm business logic operations.
type Service struct {
	provider ClusterProvider
	emit     EventEmitter
}

// NewService creates a new helm Service.
func NewService(provider ClusterProvider, emit EventEmitter) *Service {
	return &Service{
		provider: provider,
		emit:     emit,
	}
}

// SetActiveContext updates the provider's active cluster context if it supports dynamic switching.
// Returns an error if the provider does not implement MutableClusterProvider.
func (s *Service) SetActiveContext(contextName, kubeconfigPath string) error {
	mp, ok := s.provider.(MutableClusterProvider)
	if !ok {
		return fmt.Errorf("helm: cluster provider does not support dynamic context switching")
	}
	return mp.SetActiveContext(contextName, kubeconfigPath)
}

// ListHelmRepositories returns the list of configured helm repositories.
func (s *Service) ListHelmRepositories() ([]dto.HelmRepository, error) {
	f, err := repo.LoadFile(helmpath.ConfigPath("repositories.yaml"))
	if err != nil {
		if os.IsNotExist(err) {
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
