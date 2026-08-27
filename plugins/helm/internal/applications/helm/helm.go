package helm

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/litelensapp/litelens-plugins/plugins/helm/internal/applications/dto"
	"github.com/litelensapp/litelens-plugins/plugins/helm/internal/applications/port"
	"github.com/litelensapp/litelens/packages/core/kube"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/helmpath"
	"helm.sh/helm/v3/pkg/repo"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// Service provides helm business logic operations.
type Service struct {
	provider      port.ClusterProvider
	emit          port.EventEmitter
	getterFactory port.RESTClientGetterFactory
	cache         *cache
}

// NewService creates a new helm Service.
func NewService(provider port.ClusterProvider, emit port.EventEmitter, getterFactory port.RESTClientGetterFactory) *Service {
	return &Service{
		provider:      provider,
		emit:          emit,
		getterFactory: getterFactory,
		cache:         newCache(),
	}
}

// SetActiveContext updates the provider's active cluster context if it supports dynamic switching.
// Returns an error if the provider does not implement MutableClusterProvider.
func (s *Service) SetActiveContext(contextName, kubeconfigPath string) error {
	mp, ok := s.provider.(port.MutableClusterProvider)
	if !ok {
		return fmt.Errorf("helm: cluster provider does not support dynamic context switching")
	}
	err := mp.SetActiveContext(contextName, kubeconfigPath)
	if err == nil {
		s.cache.invalidateAllConfigs()
	}
	return err
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

// getOrCreateConfig returns a cached action.Configuration for the active context, or creates and caches a new one.
func (s *Service) getOrCreateConfig(namespace, activeCtx string) (*action.Configuration, error) {
	cs, rc, _, kubeconfigPaths := s.provider.ActiveClients()
	if cs == nil || rc == nil {
		return nil, fmt.Errorf("helm: no active kubernetes context or REST config")
	}

	cached, _ := s.cache.getConfig(activeCtx)
	if cached != nil {
		return cached, nil
	}

	getter := s.getterFactory.NewRESTClientGetter(rc, kube.LoadingRules(kubeconfigPaths), &clientcmd.ConfigOverrides{CurrentContext: activeCtx})
	cfg := new(action.Configuration)
	if err := cfg.Init(getter, namespace, "secrets", func(string, ...any) {}); err != nil {
		return nil, fmt.Errorf("helm: init configuration: %w", err)
	}

	s.cache.setConfig(activeCtx, cfg)
	return cfg, nil
}

// getOrCreateIndex loads or returns a cached helm repository index file.
func (s *Service) getOrCreateIndex(repository string) (*repo.IndexFile, map[string]*repo.ChartVersion, error) {
	cachedIndex, cachedVersionMap := s.cache.getIndex(repository)
	if cachedIndex != nil && cachedVersionMap != nil {
		return cachedIndex, cachedVersionMap, nil
	}

	cacheDir := helmpath.CachePath("repository")
	indexPath := filepath.Join(cacheDir, repository+"-index.yaml")
	index, err := repo.LoadIndexFile(indexPath)
	if err != nil {
		return nil, nil, fmt.Errorf("helm: load index %s: %w", indexPath, err)
	}

	versionMap := make(map[string]*repo.ChartVersion)
	for _, versions := range index.Entries {
		for _, v := range versions {
			if v != nil {
				versionMap[v.Version] = v
			}
		}
	}

	s.cache.setIndex(repository, index, versionMap)
	return index, versionMap, nil
}

// getOrCreateDiscoveryMap returns a cached discovery result or fetches it.
func (s *Service) getOrCreateDiscoveryMap(activeCtx string, cs *kubernetes.Clientset) map[string]gvrMeta {
	_, gvrMap := s.cache.getConfig(activeCtx)
	if len(gvrMap) > 0 {
		return gvrMap
	}

	kindMap := make(map[string]gvrMeta)
	serverResources, discoveryErr := cs.Discovery().ServerPreferredResources()
	if discoveryErr != nil {
		return kindMap
	}
	for _, rl := range serverResources {
		gv, err := schema.ParseGroupVersion(rl.GroupVersion)
		if err != nil {
			continue
		}
		for _, r := range rl.APIResources {
			if strings.Contains(r.Name, "/") {
				continue
			}
			if helmHasVerb(r.Verbs, "delete") {
				kindMap[r.Kind] = gvrMeta{
					gvr:        schema.GroupVersionResource{Group: gv.Group, Version: gv.Version, Resource: r.Name},
					namespaced: r.Namespaced,
				}
			}
		}
	}

	s.cache.setDiscovery(activeCtx, kindMap)
	return kindMap
}
