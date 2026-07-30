package helm

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gknguyen/litelens/internal/dto"
	"github.com/gknguyen/litelens/internal/kube"
	"helm.sh/helm/v3/pkg/action"
	helmchart "helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/helmpath"
	"helm.sh/helm/v3/pkg/registry"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	memorycache "k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/yaml"
)

// HelmRepositoryLabel is the release label key used to store the chart's source repository.
const HelmRepositoryLabel = "meta.litelens.io/helm-repository-name"

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

// clusterProviderFunc adapts plain accessor closures to ClusterProvider, so
// callers (e.g. package app) can satisfy this interface without exposing
// ActiveClients/Ctx as public methods on a Wails-bound struct. Wails scans
// exported methods on bound structs for JS binding generation; leaking
// client-go internals like *kubernetes.Clientset via such a method makes
// Wails walk into types it can't model (e.g. schema.GroupVersion),
// producing binding warnings in `wails dev`.
type clusterProviderFunc struct {
	activeClients func() (*kubernetes.Clientset, *rest.Config, string, []string)
	ctx           func() context.Context
}

func (f *clusterProviderFunc) ActiveClients() (*kubernetes.Clientset, *rest.Config, string, []string) {
	return f.activeClients()
}

func (f *clusterProviderFunc) Ctx() context.Context {
	return f.ctx()
}

// NewClusterProviderFunc builds a ClusterProvider from plain accessor closures.
func NewClusterProviderFunc(
	activeClients func() (*kubernetes.Clientset, *rest.Config, string, []string),
	ctx func() context.Context,
) ClusterProvider {
	return &clusterProviderFunc{activeClients: activeClients, ctx: ctx}
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

// helmRestGetter implements genericclioptions.RESTClientGetter using an existing rest.Config.
type helmRestGetter struct {
	rc        *rest.Config
	rules     *clientcmd.ClientConfigLoadingRules
	overrides *clientcmd.ConfigOverrides
}

func (g *helmRestGetter) ToRESTConfig() (*rest.Config, error) {
	return rest.CopyConfig(g.rc), nil
}

func (g *helmRestGetter) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	dc, err := discovery.NewDiscoveryClientForConfig(g.rc)
	if err != nil {
		return nil, err
	}
	return memorycache.NewMemCacheClient(dc), nil
}

func (g *helmRestGetter) ToRESTMapper() (meta.RESTMapper, error) {
	dc, err := g.ToDiscoveryClient()
	if err != nil {
		return nil, err
	}
	return restmapper.NewDeferredDiscoveryRESTMapper(dc), nil
}

func (g *helmRestGetter) ToRawKubeConfigLoader() clientcmd.ClientConfig {
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(g.rules, g.overrides)
}

// ListHelmCharts reads the local helm repo cache and returns available charts.
func (s *Service) ListHelmCharts() ([]dto.HelmChart, error) {
	cacheDir := helmpath.CachePath("repository")

	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []dto.HelmChart{}, nil
		}
		return []dto.HelmChart{}, fmt.Errorf("helm: read cache dir: %w", err)
	}

	var charts []dto.HelmChart
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), "-index.yaml") {
			continue
		}
		repoAlias := strings.TrimSuffix(filepath.Base(entry.Name()), "-index.yaml")
		indexPath := filepath.Join(cacheDir, entry.Name())
		index, err := repo.LoadIndexFile(indexPath)
		if err != nil {
			log.Printf("helm: load index %s: %v", indexPath, err)
			continue
		}
		index.SortEntries()
		for chartName, versions := range index.Entries {
			if len(versions) == 0 {
				continue
			}
			latest := versions[0]
			charts = append(charts, dto.HelmChart{
				Name:        chartName,
				Description: latest.Description,
				Version:     latest.Version,
				AppVersion:  latest.AppVersion,
				Repository:  repoAlias,
				Icon:        latest.Icon,
			})
		}
	}

	if charts == nil {
		return []dto.HelmChart{}, nil
	}
	return charts, nil
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

// mergedValuesYAML merges a release's chart defaults with user-supplied overrides
// and marshals the result to YAML. Returns "" if the chart is nil or marshal fails.
func mergedValuesYAML(rel *release.Release) string {
	if rel == nil || rel.Chart == nil {
		return ""
	}
	merged := make(map[string]interface{}, len(rel.Chart.Values))
	for k, v := range rel.Chart.Values {
		merged[k] = v
	}
	for k, v := range rel.Config {
		merged[k] = v
	}
	yamlBytes, err := yaml.Marshal(merged)
	if err != nil {
		return ""
	}
	return string(yamlBytes)
}

// compressValuesYAML gzip-compresses valuesYAML and base64-encodes the result so it
// can be transmitted as a JSON string over Wails IPC without bloating the payload of
// ListHelmReleases (called for every release row, unlike the detail endpoint).
// Returns "" if input is empty or compression fails.
func compressValuesYAML(valuesYAML string) string {
	if valuesYAML == "" {
		return ""
	}
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write([]byte(valuesYAML)); err != nil {
		return ""
	}
	if err := gz.Close(); err != nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

func (s *Service) ListHelmReleases(namespace string) ([]dto.HelmRelease, error) {
	cs, rc, activeCtx, kubeconfigPaths := s.provider.ActiveClients()
	if cs == nil {
		return []dto.HelmRelease{}, nil
	}
	if rc == nil {
		return []dto.HelmRelease{}, nil
	}

	getter := &helmRestGetter{
		rc:        rc,
		rules:     kube.LoadingRules(kubeconfigPaths),
		overrides: &clientcmd.ConfigOverrides{CurrentContext: activeCtx},
	}
	cfg := new(action.Configuration)
	if err := cfg.Init(getter, namespace, "secrets", func(string, ...any) {}); err != nil {
		return []dto.HelmRelease{}, fmt.Errorf("helm: init configuration: %w", err)
	}

	list := action.NewList(cfg)
	list.AllNamespaces = namespace == ""
	list.StateMask = action.ListAll

	releases, err := list.Run()
	if err != nil {
		return []dto.HelmRelease{}, fmt.Errorf("helm: list releases: %w", err)
	}

	result := make([]dto.HelmRelease, 0, len(releases))
	for _, r := range releases {
		if r.Chart == nil || r.Chart.Metadata == nil {
			continue
		}
		repository := ""
		if r.Labels != nil {
			repository = r.Labels[HelmRepositoryLabel]
		}
		result = append(result, dto.HelmRelease{
			Name:              r.Name,
			Namespace:         r.Namespace,
			Chart:             r.Chart.Metadata.Name,
			ChartVersion:      r.Chart.Metadata.Version,
			AppVersion:        r.Chart.Metadata.AppVersion,
			Status:            r.Info.Status.String(),
			Revision:          r.Version,
			Updated:           helmAge(r.Info.LastDeployed.Time),
			UpdatedAt:         r.Info.LastDeployed.Time.Format(time.RFC3339),
			Repository:        repository,
			EncodedValuesYAML: compressValuesYAML(mergedValuesYAML(r)),
		})
	}
	return result, nil
}

// ListHelmChartVersions returns all available versions of a chart from a repository.
func (s *Service) ListHelmChartVersions(repository, chartName string) ([]string, error) {
	cacheDir := helmpath.CachePath("repository")
	indexPath := filepath.Join(cacheDir, repository+"-index.yaml")
	index, err := repo.LoadIndexFile(indexPath)
	if err != nil {
		return []string{}, fmt.Errorf("helm: load index %s: %w", indexPath, err)
	}
	versions, ok := index.Entries[chartName]
	if !ok || len(versions) == 0 {
		return []string{}, fmt.Errorf("helm: chart %q not found in repo %q", chartName, repository)
	}
	result := make([]string, 0, len(versions))
	for _, v := range versions {
		if v != nil {
			result = append(result, v.Version)
		}
	}
	return result, nil
}

// GetHelmChartDetail returns detailed metadata for a single chart from a local repo cache.
// If version is empty, returns the latest version entry.
func (s *Service) GetHelmChartDetail(repository, chartName, version string) (dto.HelmChartDetail, error) {
	cacheDir := helmpath.CachePath("repository")
	indexPath := filepath.Join(cacheDir, repository+"-index.yaml")
	index, err := repo.LoadIndexFile(indexPath)
	if err != nil {
		return dto.HelmChartDetail{}, fmt.Errorf("helm: load index %s: %w", indexPath, err)
	}
	versions, ok := index.Entries[chartName]
	if !ok || len(versions) == 0 {
		return dto.HelmChartDetail{}, fmt.Errorf("helm: chart %q not found in repo %q", chartName, repository)
	}

	var chartVersion = versions[0]
	if version != "" {
		found := false
		for _, v := range versions {
			if v != nil && v.Version == version {
				chartVersion = v
				found = true
				break
			}
		}
		if !found {
			return dto.HelmChartDetail{}, fmt.Errorf("helm: version %q of chart %q not found", version, chartName)
		}
	}

	maintainers := make([]string, 0, len(chartVersion.Maintainers))
	for _, m := range chartVersion.Maintainers {
		if m == nil {
			continue
		}
		if m.Email != "" {
			maintainers = append(maintainers, m.Name+" <"+m.Email+">")
		} else {
			maintainers = append(maintainers, m.Name)
		}
	}
	keywords := chartVersion.Keywords
	if keywords == nil {
		keywords = []string{}
	}
	sources := chartVersion.Sources
	if sources == nil {
		sources = []string{}
	}
	return dto.HelmChartDetail{
		Name:        chartName,
		Description: chartVersion.Description,
		Version:     chartVersion.Version,
		AppVersion:  chartVersion.AppVersion,
		Repository:  repository,
		Icon:        chartVersion.Icon,
		Home:        chartVersion.Home,
		Keywords:    keywords,
		Sources:     sources,
		Maintainers: maintainers,
	}, nil
}

// resolveArtifactHubRepoName resolves a local Helm repository alias to its
// registered name on Artifact Hub. The local alias (whatever name the user
// passed to `helm repo add <alias> <url>`) frequently does not match Artifact
// Hub's canonical repository name for that same URL — e.g. a local alias of
// "jenkin" for https://charts.jenkins.io resolves to Artifact Hub's "jenkinsci".
// Artifact Hub's packages API is keyed by its own repository name, not by
// arbitrary local aliases, so this lookup is required before any package
// request can succeed.
func (s *Service) resolveArtifactHubRepoName(ctx context.Context, repository string) (string, error) {
	repoFile, err := repo.LoadFile(helmpath.ConfigPath("repositories.yaml"))
	if err != nil {
		return "", fmt.Errorf("artifacthub: load repositories.yaml: %w", err)
	}
	var repoURL string
	for _, r := range repoFile.Repositories {
		if r.Name == repository {
			repoURL = r.URL
			break
		}
	}
	if repoURL == "" {
		return "", fmt.Errorf("artifacthub: repository %q not found in repositories.yaml", repository)
	}

	reqURL := "https://artifacthub.io/api/v1/repositories/search?url=" + url.QueryEscape(repoURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return "", fmt.Errorf("artifacthub: new request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("artifacthub: resolve repo: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("artifacthub: resolve repo: HTTP %d", resp.StatusCode)
	}
	var matches []struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&matches); err != nil {
		return "", fmt.Errorf("artifacthub: decode repo search: %w", err)
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("artifacthub: no repository indexed for %q", repoURL)
	}
	return matches[0].Name, nil
}

// GetArtifactHubReadme fetches chart documentation from ArtifactHub.
func (s *Service) GetArtifactHubReadme(repository, chartName, version string) (string, error) {
	reqCtx, reqCancel := context.WithTimeout(s.provider.Ctx(), 60*time.Second)
	defer reqCancel()

	ahRepoName, err := s.resolveArtifactHubRepoName(reqCtx, repository)
	if err != nil {
		return "", err
	}

	reqURL := fmt.Sprintf("https://artifacthub.io/api/v1/packages/helm/%s/%s", ahRepoName, chartName)
	if version != "" {
		reqURL += "/" + version
	}
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, reqURL, nil)
	if err != nil {
		return "", fmt.Errorf("artifacthub: new request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("artifacthub: do request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("artifacthub: HTTP %d", resp.StatusCode)
	}
	var body struct {
		Readme string `json:"readme"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", fmt.Errorf("artifacthub: decode: %w", err)
	}
	return body.Readme, nil
}

// resolveChartURL returns an absolute URL for a chart download entry.
// OCI and HTTP(S) absolute URLs are returned unchanged.
// Relative URLs are resolved against the repo's configured URL from repositories.yaml.
func resolveChartURL(repository, rawURL string) (string, error) {
	if strings.HasPrefix(rawURL, "//") {
		return "", fmt.Errorf("helm: protocol-relative URLs are not allowed: %q", rawURL)
	}
	if strings.HasPrefix(rawURL, "oci://") ||
		strings.HasPrefix(rawURL, "http://") ||
		strings.HasPrefix(rawURL, "https://") {
		return rawURL, nil
	}
	repoFile, err := repo.LoadFile(helmpath.ConfigPath("repositories.yaml"))
	if err != nil {
		return "", fmt.Errorf("helm: load repositories.yaml: %w", err)
	}
	for _, r := range repoFile.Repositories {
		if r.Name == repository {
			base, err := url.Parse(strings.TrimRight(r.URL, "/") + "/")
			if err != nil {
				return "", fmt.Errorf("helm: parse repo URL %q: %w", r.URL, err)
			}
			ref, err := url.Parse(rawURL)
			if err != nil {
				return "", fmt.Errorf("helm: parse chart URL %q: %w", rawURL, err)
			}
			return base.ResolveReference(ref).String(), nil
		}
	}
	return "", fmt.Errorf("helm: repository %q not found in repositories.yaml", repository)
}

// downloadChart fetches and parses a Helm chart from an HTTP(S) or OCI URL.
func downloadChart(ctx context.Context, chartURL string) (*helmchart.Chart, error) {
	if ref, ok := strings.CutPrefix(chartURL, "oci://"); ok {
		rc, err := registry.NewClient()
		if err != nil {
			return nil, fmt.Errorf("helm: create OCI registry client: %w", err)
		}
		result, err := rc.Pull(ref, registry.PullOptWithChart(true))
		if err != nil {
			return nil, fmt.Errorf("helm: pull OCI chart %q: %w", ref, err)
		}
		if result.Chart == nil || len(result.Chart.Data) == 0 {
			return nil, fmt.Errorf("helm: OCI pull returned no chart data for %q", ref)
		}
		ch, err := loader.LoadArchive(bytes.NewReader(result.Chart.Data))
		if err != nil {
			return nil, fmt.Errorf("helm: load OCI chart archive: %w", err)
		}
		return ch, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, chartURL, nil)
	if err != nil {
		return nil, fmt.Errorf("helm: build request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("helm: download chart: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("helm: download chart: status %d", resp.StatusCode)
	}
	ch, err := loader.LoadArchive(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("helm: load chart archive: %w", err)
	}
	return ch, nil
}

// InstallHelmChart installs a helm chart into the specified namespace.
func (s *Service) InstallHelmChart(namespace, releaseName, repository, chartName, version, valuesYAML string) error {
	cs, rc, activeCtx, kubeconfigPaths := s.provider.ActiveClients()
	if cs == nil {
		return fmt.Errorf("helm: no active kubernetes context")
	}
	if rc == nil {
		return fmt.Errorf("helm: no REST config for active context")
	}

	// Load the chart index to get the download URL
	cacheDir := helmpath.CachePath("repository")
	indexPath := filepath.Join(cacheDir, repository+"-index.yaml")
	index, err := repo.LoadIndexFile(indexPath)
	if err != nil {
		return fmt.Errorf("helm: load index: %w", err)
	}

	versions, ok := index.Entries[chartName]
	if !ok || len(versions) == 0 {
		return fmt.Errorf("helm: chart %q not found in repo %q", chartName, repository)
	}

	var chartVersion = versions[0]
	if version != "" {
		found := false
		for _, v := range versions {
			if v != nil && v.Version == version {
				chartVersion = v
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("helm: version %q of chart %q not found", version, chartName)
		}
	}

	if len(chartVersion.URLs) == 0 {
		return fmt.Errorf("helm: no download URLs found for chart %q version %q", chartName, chartVersion.Version)
	}
	chartURL, err := resolveChartURL(repository, chartVersion.URLs[0])
	if err != nil {
		return fmt.Errorf("helm: resolve chart URL: %w", err)
	}

	dlCtx, dlCancel := context.WithTimeout(s.provider.Ctx(), 60*time.Second)
	defer dlCancel()
	chart, err := downloadChart(dlCtx, chartURL)
	if err != nil {
		return err
	}

	// Set up helm configuration wired to the active cluster context.
	getter := &helmRestGetter{
		rc:        rc,
		rules:     kube.LoadingRules(kubeconfigPaths),
		overrides: &clientcmd.ConfigOverrides{CurrentContext: activeCtx},
	}
	cfg := new(action.Configuration)
	if err := cfg.Init(getter, namespace, "secrets", func(string, ...any) {}); err != nil {
		return fmt.Errorf("helm: init configuration: %w", err)
	}

	// Create install action (synchronous setup only)
	install := action.NewInstall(cfg)
	install.Namespace = namespace
	if releaseName == "" {
		install.ReleaseName = fmt.Sprintf("%s-%d", chartName, time.Now().Unix())
	} else {
		install.ReleaseName = releaseName
	}
	install.CreateNamespace = false
	install.Labels = map[string]string{
		HelmRepositoryLabel: repository,
	}

	// Parse custom values YAML if provided
	var vals map[string]interface{}
	if valuesYAML != "" {
		if err := yaml.Unmarshal([]byte(valuesYAML), &vals); err != nil {
			return fmt.Errorf("helm: parse custom values: %w", err)
		}
	}

	// Run the actual install asynchronously so the frontend receives control
	// immediately after setup and can display the pending-install release.
	relName := install.ReleaseName
	go func() {
		installCtx, installCancel := context.WithTimeout(s.provider.Ctx(), 5*time.Minute)
		defer installCancel()
		if _, err := install.RunWithContext(installCtx, chart, vals); err != nil {
			s.emit(s.provider.Ctx(), "helm:install:error", map[string]string{
				"releaseName": relName,
				"chartName":   chartName,
				"error":       err.Error(),
			})
			return
		}
		s.emit(s.provider.Ctx(), "helm:install:complete", map[string]string{
			"releaseName": relName,
			"chartName":   chartName,
		})
	}()

	return nil
}

// UpgradeHelmRelease upgrades an existing helm release to a different chart version.
func (s *Service) UpgradeHelmRelease(namespace, releaseName, repository, chartName, version, valuesYAML string) error {
	cs, rc, activeCtx, kubeconfigPaths := s.provider.ActiveClients()
	if cs == nil {
		return fmt.Errorf("helm: no active kubernetes context")
	}
	if rc == nil {
		return fmt.Errorf("helm: no REST config for active context")
	}

	// Load the chart index to get the download URL for the target version
	cacheDir := helmpath.CachePath("repository")
	indexPath := filepath.Join(cacheDir, repository+"-index.yaml")
	index, err := repo.LoadIndexFile(indexPath)
	if err != nil {
		return fmt.Errorf("helm: load index: %w", err)
	}

	versions, ok := index.Entries[chartName]
	if !ok || len(versions) == 0 {
		return fmt.Errorf("helm: chart %q not found in repo %q", chartName, repository)
	}

	var chartVersion = versions[0]
	if version != "" {
		found := false
		for _, v := range versions {
			if v != nil && v.Version == version {
				chartVersion = v
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("helm: version %q of chart %q not found", version, chartName)
		}
	}

	if len(chartVersion.URLs) == 0 {
		return fmt.Errorf("helm: no download URLs found for chart %q version %q", chartName, chartVersion.Version)
	}
	chartURL, err := resolveChartURL(repository, chartVersion.URLs[0])
	if err != nil {
		return fmt.Errorf("helm: resolve chart URL: %w", err)
	}

	dlCtx, dlCancel := context.WithTimeout(s.provider.Ctx(), 60*time.Second)
	defer dlCancel()
	chart, err := downloadChart(dlCtx, chartURL)
	if err != nil {
		return err
	}

	// Set up helm configuration wired to the active cluster context.
	getter := &helmRestGetter{
		rc:        rc,
		rules:     kube.LoadingRules(kubeconfigPaths),
		overrides: &clientcmd.ConfigOverrides{CurrentContext: activeCtx},
	}
	cfg := new(action.Configuration)
	if err := cfg.Init(getter, namespace, "secrets", func(string, ...any) {}); err != nil {
		return fmt.Errorf("helm: init configuration: %w", err)
	}

	// Parse custom values YAML if provided, otherwise use empty map
	var vals map[string]any
	if valuesYAML != "" {
		if err := yaml.Unmarshal([]byte(valuesYAML), &vals); err != nil {
			return fmt.Errorf("helm: parse custom values: %w", err)
		}
	}

	// Run the upgrade asynchronously
	relName := releaseName
	go func() {
		upgradeCtx, upgradeCancel := context.WithTimeout(s.provider.Ctx(), 5*time.Minute)
		defer upgradeCancel()

		upgrade := action.NewUpgrade(cfg)
		upgrade.Namespace = namespace
		upgrade.Labels = map[string]string{
			HelmRepositoryLabel: repository,
		}

		if _, err := upgrade.RunWithContext(upgradeCtx, relName, chart, vals); err != nil {
			s.emit(s.provider.Ctx(), "helm:upgrade:error", map[string]string{
				"releaseName": relName,
				"chartName":   chartName,
				"error":       err.Error(),
			})
			return
		}
		s.emit(s.provider.Ctx(), "helm:upgrade:complete", map[string]string{
			"releaseName": relName,
			"chartName":   chartName,
		})
	}()

	return nil
}

// DeleteHelmRelease uninstalls a helm release from the specified namespace.
func (s *Service) DeleteHelmRelease(namespace, releaseName string) error {
	cs, rc, activeCtx, kubeconfigPaths := s.provider.ActiveClients()
	if cs == nil {
		return fmt.Errorf("helm: no active kubernetes context")
	}
	if rc == nil {
		return fmt.Errorf("helm: no REST config for active context")
	}

	// Set up helm configuration wired to the active cluster context.
	getter := &helmRestGetter{
		rc:        rc,
		rules:     kube.LoadingRules(kubeconfigPaths),
		overrides: &clientcmd.ConfigOverrides{CurrentContext: activeCtx},
	}
	cfg := new(action.Configuration)
	if err := cfg.Init(getter, namespace, "secrets", func(string, ...any) {}); err != nil {
		return fmt.Errorf("helm: init configuration: %w", err)
	}

	// Create and run uninstall action
	uninstall := action.NewUninstall(cfg)

	_, err := uninstall.Run(releaseName)
	if err != nil {
		return fmt.Errorf("helm: uninstall release: %w", err)
	}

	return nil
}

// DeleteHelmReleaseWithCleanup uninstalls a helm release synchronously, then
// asynchronously deletes any resources from the release manifest that the
// helm uninstall left behind (e.g. hook jobs with keep delete policy).
// Emits helm:cleanup:complete, helm:cleanup:partial, or helm:cleanup:error
// when the background sweep finishes.
func (s *Service) DeleteHelmReleaseWithCleanup(namespace, releaseName string) error {
	cs, rc, activeCtx, kubeconfigPaths := s.provider.ActiveClients()
	if cs == nil {
		return fmt.Errorf("helm: no active kubernetes context")
	}
	if rc == nil {
		return fmt.Errorf("helm: no REST config for active context")
	}

	getter := &helmRestGetter{
		rc:        rc,
		rules:     kube.LoadingRules(kubeconfigPaths),
		overrides: &clientcmd.ConfigOverrides{CurrentContext: activeCtx},
	}
	cfg := new(action.Configuration)
	if err := cfg.Init(getter, namespace, "secrets", func(string, ...any) {}); err != nil {
		return fmt.Errorf("helm: init configuration: %w", err)
	}

	// Capture the full manifest before uninstalling so we know exactly which
	// resources Helm deployed — regardless of what labels they carry.
	rel, err := action.NewGet(cfg).Run(releaseName)
	if err != nil {
		return fmt.Errorf("helm: get release manifest: %w", err)
	}
	manifestResources := parseManifestResources(rel.Manifest, namespace)

	// Hooks (pre-/post-install jobs etc.) live in rel.Hooks, not rel.Manifest.
	for _, hook := range rel.Hooks {
		if hook == nil {
			continue
		}
		manifestResources = append(manifestResources, parseManifestResources(hook.Manifest, namespace)...)
	}

	// Helm uninstall is synchronous — the release record is removed before we return.
	uninstall := action.NewUninstall(cfg)
	if _, err := uninstall.Run(releaseName); err != nil {
		return fmt.Errorf("helm: uninstall release: %w", err)
	}

	// Delete remaining manifest resources asynchronously. Helm already removes
	// most resources; we delete whatever is left (e.g. hook jobs with keep
	// delete policy). NotFound errors are silently ignored.
	go func() {
		dynClient, err := dynamic.NewForConfig(rc)
		if err != nil {
			s.emit(s.provider.Ctx(), "helm:cleanup:error", map[string]any{
				"releaseName": releaseName,
				"error":       fmt.Sprintf("build dynamic client: %s", err.Error()),
			})
			return
		}

		cleanCtx, cleanCancel := context.WithTimeout(s.provider.Ctx(), 60*time.Second)
		defer cleanCancel()

		var deleted int
		var cleanupErrs []string

		// Build Kind → GVR + namespaced flag from discovery so we can look up
		// resources by Kind (the only type info present in a manifest).
		// ServerPreferredResources can return partial results alongside an error;
		// record any error but still process whatever groups were returned.
		type gvrMeta struct {
			gvr        schema.GroupVersionResource
			namespaced bool
		}
		kindMap := map[string]gvrMeta{}
		serverResources, discoveryErr := cs.Discovery().ServerPreferredResources()
		if discoveryErr != nil {
			cleanupErrs = append(cleanupErrs, fmt.Sprintf("resource discovery: %s", discoveryErr.Error()))
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

		// Step 1: Build resource identity set.
		type resKey struct {
			kind      string
			namespace string
			name      string
		}
		inManifest := map[resKey]bool{}
		for _, res := range manifestResources {
			ns := res.Namespace
			if ns == "" {
				ns = namespace
			}
			inManifest[resKey{res.Kind, ns, res.Name}] = true
		}

		// Step 2: Fetch ownerReferences for each resource.
		ownerOf := map[resKey][]resKey{}
		for _, res := range manifestResources {
			gm, ok := kindMap[res.Kind]
			if !ok {
				continue
			}
			ns := res.Namespace
			if ns == "" {
				ns = namespace
			}
			var ri dynamic.ResourceInterface
			if gm.namespaced {
				ri = dynClient.Resource(gm.gvr).Namespace(ns)
			} else {
				ri = dynClient.Resource(gm.gvr)
			}
			obj, err := ri.Get(cleanCtx, res.Name, metav1.GetOptions{})
			if err != nil {
				if !k8serrors.IsNotFound(err) {
					cleanupErrs = append(cleanupErrs, fmt.Sprintf("%s/%s: get: %s", res.Kind, res.Name, err.Error()))
				}
				// IsNotFound = helm already deleted it; no owners to fetch
				continue
			}
			for _, ref := range obj.GetOwnerReferences() {
				ownerKey := resKey{ref.Kind, ns, ref.Name}
				if inManifest[ownerKey] {
					ownerOf[resKey{res.Kind, ns, res.Name}] = append(ownerOf[resKey{res.Kind, ns, res.Name}], ownerKey)
				}
			}
		}

		// Step 3: Topological deletion loop.
		deletedSet := map[resKey]bool{}

		for {
			if cleanCtx.Err() != nil {
				break
			}
			progress := false
			for _, res := range manifestResources {
				gm, ok := kindMap[res.Kind]
				if !ok {
					continue
				}
				ns := res.Namespace
				if ns == "" {
					ns = namespace
				}
				key := resKey{res.Kind, ns, res.Name}
				if deletedSet[key] {
					continue
				}

				// Only delete if all manifest-owners have already been deleted.
				ready := true
				for _, ownerKey := range ownerOf[key] {
					if !deletedSet[ownerKey] {
						ready = false
						break
					}
				}
				if !ready {
					continue
				}

				var ri dynamic.ResourceInterface
				if gm.namespaced {
					ri = dynClient.Resource(gm.gvr).Namespace(ns)
				} else {
					ri = dynClient.Resource(gm.gvr)
				}
				// Background propagation: object marked for deletion immediately; owned pods
				// drain concurrently. Label sweeps run after a grace period (see below).
				if err := ri.Delete(cleanCtx, res.Name, metav1.DeleteOptions{}); err != nil {
					if !k8serrors.IsNotFound(err) {
						cleanupErrs = append(cleanupErrs, fmt.Sprintf("%s/%s: %s", res.Kind, res.Name, err.Error()))
					}
				} else {
					deleted++
				}
				deletedSet[key] = true
				progress = true
			}
			if !progress {
				break // no more resources can be unblocked
			}
		}

		// Allow operator pods (deleted with Background propagation) to finish their
		// reconciliation cycle before label-based sweeps run. Background deletion
		// marks the parent for deletion but pods continue running briefly (~30s grace).
		// Use a context-aware select so the sleep exits immediately on cancellation.
		select {
		case <-time.After(5 * time.Second):
		case <-cleanCtx.Done():
		}
		if cleanCtx.Err() != nil {
			cleanupErrs = append(cleanupErrs, fmt.Sprintf("cleanup cancelled before label sweeps: %s", cleanCtx.Err()))
		} else {

			// ConfigMaps created by running operators are not in the manifest;
			// sweep them by the standard Helm instance label.
			cmList, cmListErr := cs.CoreV1().ConfigMaps(namespace).List(
				cleanCtx,
				metav1.ListOptions{LabelSelector: fmt.Sprintf("app.kubernetes.io/instance=%s", releaseName)},
			)
			if cmListErr != nil {
				cleanupErrs = append(cleanupErrs, fmt.Sprintf("list ConfigMaps: %s", cmListErr.Error()))
			} else {
				for _, cm := range cmList.Items {
					if err := cs.CoreV1().ConfigMaps(namespace).Delete(
						cleanCtx, cm.Name, metav1.DeleteOptions{},
					); err != nil {
						if !k8serrors.IsNotFound(err) {
							cleanupErrs = append(cleanupErrs, fmt.Sprintf("ConfigMap/%s: %s", cm.Name, err.Error()))
						}
					} else {
						deleted++
					}
				}
			}

			// PVCs created by StatefulSet volumeClaimTemplates are not in the manifest;
			// sweep them by the standard Helm instance label.
			pvcList, pvcListErr := cs.CoreV1().PersistentVolumeClaims(namespace).List(
				cleanCtx,
				metav1.ListOptions{LabelSelector: fmt.Sprintf("app.kubernetes.io/instance=%s", releaseName)},
			)
			if pvcListErr != nil {
				cleanupErrs = append(cleanupErrs, fmt.Sprintf("list PVCs: %s", pvcListErr.Error()))
			} else {
				for _, pvc := range pvcList.Items {
					if err := cs.CoreV1().PersistentVolumeClaims(namespace).Delete(
						cleanCtx, pvc.Name, metav1.DeleteOptions{},
					); err != nil {
						if !k8serrors.IsNotFound(err) {
							cleanupErrs = append(cleanupErrs, fmt.Sprintf("PersistentVolumeClaim/%s: %s", pvc.Name, err.Error()))
						}
					} else {
						deleted++
					}
				}
			}

			// Pods created by hook Jobs become orphaned once the Job is deleted;
			// delete terminal (Failed/Succeeded) pods that carry the release label.
			podList, podListErr := cs.CoreV1().Pods(namespace).List(
				cleanCtx,
				metav1.ListOptions{LabelSelector: fmt.Sprintf("app.kubernetes.io/instance=%s", releaseName)},
			)
			if podListErr != nil {
				cleanupErrs = append(cleanupErrs, fmt.Sprintf("list Pods: %s", podListErr.Error()))
			} else {
				for _, pod := range podList.Items {
					if pod.Status.Phase != corev1.PodFailed && pod.Status.Phase != corev1.PodSucceeded {
						continue
					}
					if err := cs.CoreV1().Pods(namespace).Delete(
						cleanCtx, pod.Name, metav1.DeleteOptions{},
					); err != nil {
						if !k8serrors.IsNotFound(err) {
							cleanupErrs = append(cleanupErrs, fmt.Sprintf("Pod/%s: %s", pod.Name, err.Error()))
						}
					} else {
						deleted++
					}
				}
			}
		} // end of context-check else block

		switch {
		case len(cleanupErrs) == 0:
			s.emit(s.provider.Ctx(), "helm:cleanup:complete", map[string]any{
				"releaseName": releaseName,
				"deleted":     deleted,
			})
		case deleted > 0:
			s.emit(s.provider.Ctx(), "helm:cleanup:partial", map[string]any{
				"releaseName": releaseName,
				"deleted":     deleted,
				"errors":      cleanupErrs,
			})
		default:
			s.emit(s.provider.Ctx(), "helm:cleanup:error", map[string]any{
				"releaseName": releaseName,
				"error":       strings.Join(cleanupErrs, "; "),
			})
		}
	}()

	return nil
}

// helmHasVerb reports whether the given verb appears in the resource's verb list.
func helmHasVerb(verbs metav1.Verbs, verb string) bool {
	for _, v := range verbs {
		if v == verb {
			return true
		}
	}
	return false
}

// parseManifestResources splits a multi-document Helm manifest and extracts
// kind/name/namespace from each document, skipping empty or comment-only docs.
// Namespaced resource templates frequently omit `metadata.namespace` and rely
// on the release namespace instead, so releaseNamespace is used as a fallback.
func parseManifestResources(manifest, releaseNamespace string) []dto.HelmReleaseResource {
	var out []dto.HelmReleaseResource
	for _, doc := range strings.Split(manifest, "\n---") {
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}
		var meta struct {
			Kind     string `json:"kind"`
			Metadata struct {
				Name      string `json:"name"`
				Namespace string `json:"namespace"`
			} `json:"metadata"`
		}
		if err := yaml.Unmarshal([]byte(doc), &meta); err != nil || meta.Kind == "" || meta.Metadata.Name == "" {
			continue
		}
		ns := meta.Metadata.Namespace
		if ns == "" {
			ns = releaseNamespace
		}
		out = append(out, dto.HelmReleaseResource{
			Kind:      meta.Kind,
			Name:      meta.Metadata.Name,
			Namespace: ns,
		})
	}
	return out
}

// GetHelmReleaseByName returns detailed metadata for a single helm release.
func (s *Service) GetHelmReleaseByName(namespace, releaseName string) (*dto.HelmReleaseDetail, error) {
	cs, rc, activeCtx, kubeconfigPaths := s.provider.ActiveClients()
	if cs == nil {
		return nil, fmt.Errorf("helm: no active kubernetes context")
	}
	if rc == nil {
		return nil, fmt.Errorf("helm: no REST config for active context")
	}

	// Set up helm configuration wired to the active cluster context.
	getter := &helmRestGetter{
		rc:        rc,
		rules:     kube.LoadingRules(kubeconfigPaths),
		overrides: &clientcmd.ConfigOverrides{CurrentContext: activeCtx},
	}
	cfg := new(action.Configuration)
	if err := cfg.Init(getter, namespace, "secrets", func(string, ...any) {}); err != nil {
		return nil, fmt.Errorf("helm: init configuration: %w", err)
	}

	// Create and run get action
	client := action.NewGet(cfg)
	rel, err := client.Run(releaseName)
	if err != nil {
		return nil, fmt.Errorf("helm: get release: %w", err)
	}

	var appVersion, chartVersion string
	if rel.Chart != nil && rel.Chart.Metadata != nil {
		appVersion = rel.Chart.Metadata.AppVersion
		chartVersion = rel.Chart.Metadata.Version
	}

	// Merge chart defaults with user-supplied overrides so the viewer shows all values.
	valuesYAML := mergedValuesYAML(rel)

	// Resolve the chart's source repository from the release label.
	// The label is set at install/upgrade time; pre-existing releases will have "" until re-upgraded.
	repository := ""
	if rel.Labels != nil {
		if repo, ok := rel.Labels[HelmRepositoryLabel]; ok && repo != "" {
			repository = repo
		}
	}

	return &dto.HelmReleaseDetail{
		Name:         rel.Name,
		Namespace:    rel.Namespace,
		Chart:        rel.Chart.Metadata.Name,
		ChartVersion: chartVersion,
		AppVersion:   appVersion,
		Status:       rel.Info.Status.String(),
		Revision:     rel.Version,
		Updated:      helmAge(rel.Info.LastDeployed.Time),
		UpdatedAt:    rel.Info.LastDeployed.Time.Format(time.RFC3339),
		Notes:        rel.Info.Notes,
		ValuesYAML:   valuesYAML,
		Resources:    parseManifestResources(rel.Manifest, rel.Namespace),
		Repository:   repository,
	}, nil
}

// GetHelmChartValues returns the values.yaml content for a chart version.
// If version is empty, returns the latest version.
func (s *Service) GetHelmChartValues(repository, chartName, version string) (string, error) {
	cacheDir := helmpath.CachePath("repository")
	indexPath := filepath.Join(cacheDir, repository+"-index.yaml")
	index, err := repo.LoadIndexFile(indexPath)
	if err != nil {
		return "", fmt.Errorf("helm: load index %s: %w", indexPath, err)
	}
	versions, ok := index.Entries[chartName]
	if !ok || len(versions) == 0 {
		return "", fmt.Errorf("helm: chart %q not found in repo %q", chartName, repository)
	}

	var chartVersion = versions[0]
	if version != "" {
		found := false
		for _, v := range versions {
			if v != nil && v.Version == version {
				chartVersion = v
				found = true
				break
			}
		}
		if !found {
			return "", fmt.Errorf("helm: version %q of chart %q not found", version, chartName)
		}
	}

	if len(chartVersion.URLs) == 0 {
		return "", fmt.Errorf("helm: no download URLs found for chart %q version %q", chartName, chartVersion.Version)
	}
	chartURL, err := resolveChartURL(repository, chartVersion.URLs[0])
	if err != nil {
		return "", fmt.Errorf("helm: resolve chart URL: %w", err)
	}

	dlCtx, dlCancel := context.WithTimeout(s.provider.Ctx(), 60*time.Second)
	defer dlCancel()
	ch, err := downloadChart(dlCtx, chartURL)
	if err != nil {
		return "", err
	}

	// Prefer raw file content to preserve comments.
	for _, f := range ch.Raw {
		if f != nil && f.Name == "values.yaml" {
			return string(f.Data), nil
		}
	}
	// Fallback: marshal parsed values (no comments).
	if len(ch.Values) == 0 {
		return "", nil
	}
	out, err := yaml.Marshal(ch.Values)
	if err != nil {
		return "", fmt.Errorf("helm: marshal values: %w", err)
	}
	return string(out), nil
}

// GetHelmReleaseHistory returns revision history for a release, sorted newest-first.
func (s *Service) GetHelmReleaseHistory(namespace, releaseName string) ([]dto.HelmReleaseRevisionHistory, error) {
	cs, rc, activeCtx, kubeconfigPaths := s.provider.ActiveClients()
	if cs == nil {
		return []dto.HelmReleaseRevisionHistory{}, fmt.Errorf("helm: no active kubernetes context")
	}
	if rc == nil {
		return []dto.HelmReleaseRevisionHistory{}, fmt.Errorf("helm: no REST config for active context")
	}

	// Set up helm configuration wired to the active cluster context.
	getter := &helmRestGetter{
		rc:        rc,
		rules:     kube.LoadingRules(kubeconfigPaths),
		overrides: &clientcmd.ConfigOverrides{CurrentContext: activeCtx},
	}
	cfg := new(action.Configuration)
	if err := cfg.Init(getter, namespace, "secrets", func(string, ...any) {}); err != nil {
		return []dto.HelmReleaseRevisionHistory{}, fmt.Errorf("helm: init configuration: %w", err)
	}

	// Get release history
	historyAction := action.NewHistory(cfg)
	releases, err := historyAction.Run(releaseName)
	if err != nil {
		return []dto.HelmReleaseRevisionHistory{}, fmt.Errorf("helm: get history: %w", err)
	}

	// Map releases to revision history DTOs
	result := make([]dto.HelmReleaseRevisionHistory, 0, len(releases))
	for _, rel := range releases {
		if rel == nil {
			continue
		}

		chartVersion := ""
		appVersion := ""
		if rel.Chart != nil && rel.Chart.Metadata != nil {
			chartVersion = rel.Chart.Metadata.Version
			appVersion = rel.Chart.Metadata.AppVersion
		}

		updatedAt := ""
		if rel.Info != nil && !rel.Info.LastDeployed.IsZero() {
			updatedAt = rel.Info.LastDeployed.Format(time.RFC3339)
		}

		status := ""
		if rel.Info != nil {
			status = rel.Info.Status.String()
		}

		result = append(result, dto.HelmReleaseRevisionHistory{
			Revision:     rel.Version,
			ChartVersion: chartVersion,
			AppVersion:   appVersion,
			Status:       status,
			UpdatedAt:    updatedAt,
		})
	}

	// Sort descending by Revision (newest first)
	sort.Slice(result, func(i, j int) bool {
		return result[i].Revision > result[j].Revision
	})

	return result, nil
}

// RollbackHelmRelease rolls back a release to a previous revision, synchronously.
func (s *Service) RollbackHelmRelease(namespace, releaseName string, revision int) error {
	cs, rc, activeCtx, kubeconfigPaths := s.provider.ActiveClients()
	if cs == nil {
		return fmt.Errorf("helm: no active kubernetes context")
	}
	if rc == nil {
		return fmt.Errorf("helm: no REST config for active context")
	}

	// Set up helm configuration wired to the active cluster context.
	getter := &helmRestGetter{
		rc:        rc,
		rules:     kube.LoadingRules(kubeconfigPaths),
		overrides: &clientcmd.ConfigOverrides{CurrentContext: activeCtx},
	}
	cfg := new(action.Configuration)
	if err := cfg.Init(getter, namespace, "secrets", func(string, ...any) {}); err != nil {
		return fmt.Errorf("helm: init configuration: %w", err)
	}

	// Create and run rollback action
	rollback := action.NewRollback(cfg)
	rollback.Version = revision

	err := rollback.Run(releaseName)
	if err != nil {
		return fmt.Errorf("helm: rollback: %w", err)
	}

	return nil
}

// helmAge formats a duration as a human-readable "2d", "3h", "45m" style string.
func helmAge(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	d := time.Since(t)
	d = max(d, 0)
	switch {
	case d >= 24*time.Hour:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	case d >= time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
}
