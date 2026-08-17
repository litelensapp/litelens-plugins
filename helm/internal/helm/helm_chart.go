package helm

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/litelensapp/litelens-plugins/helm/internal/dto"
	"helm.sh/helm/v3/pkg/helmpath"
	"helm.sh/helm/v3/pkg/repo"
	"sigs.k8s.io/yaml"
)

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
