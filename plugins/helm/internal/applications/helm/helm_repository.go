package helm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/litelensapp/litelens-plugins/plugins/helm/internal/applications/dto"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/helmpath"
	"helm.sh/helm/v3/pkg/repo"
)

// SearchHelmRepositoryCatalog searches ArtifactHub's public catalog of known helm
// repositories, for the "add repository" one-click UI. An empty query browses the
// catalog in its default (alphabetical) order. Results are paginated via offset/limit.
func (s *Service) SearchHelmRepositoryCatalog(query string, offset, limit int) (dto.HelmRepositoryCatalogPage, error) {
	reqCtx, reqCancel := context.WithTimeout(s.provider.Ctx(), 15*time.Second)
	defer reqCancel()

	reqURL := fmt.Sprintf(
		"https://artifacthub.io/api/v1/repositories/search?kind=0&offset=%d&limit=%d",
		offset, limit,
	)
	if query != "" {
		reqURL += "&name=" + url.QueryEscape(query)
	}
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, reqURL, nil)
	if err != nil {
		return dto.HelmRepositoryCatalogPage{}, fmt.Errorf("artifacthub: new request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return dto.HelmRepositoryCatalogPage{}, fmt.Errorf("artifacthub: search repositories: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return dto.HelmRepositoryCatalogPage{}, fmt.Errorf("artifacthub: search repositories: HTTP %d", resp.StatusCode)
	}

	var matches []struct {
		Name     string `json:"name"`
		URL      string `json:"url"`
		Disabled bool   `json:"disabled"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&matches); err != nil {
		return dto.HelmRepositoryCatalogPage{}, fmt.Errorf("artifacthub: decode repository search: %w", err)
	}

	entries := make([]dto.HelmRepositoryCatalogEntry, 0, len(matches))
	for _, m := range matches {
		if m.Disabled {
			continue
		}
		entries = append(entries, dto.HelmRepositoryCatalogEntry{Name: m.Name, URL: m.URL})
	}
	return dto.HelmRepositoryCatalogPage{Entries: entries, HasMore: len(matches) == limit}, nil
}

// AddHelmRepository adds a helm repository by name/URL and downloads its index,
// mirroring the Helm CLI's `helm repo add`.
func (s *Service) AddHelmRepository(name, repoURL string) error {
	repoFile := helmpath.ConfigPath("repositories.yaml")

	if err := os.MkdirAll(filepath.Dir(repoFile), 0o755); err != nil {
		return fmt.Errorf("helm: create repository config dir: %w", err)
	}

	f, err := repo.LoadFile(repoFile)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("helm: load repositories.yaml: %w", err)
		}
		f = repo.NewFile()
	}

	entry := repo.Entry{Name: name, URL: repoURL}

	settings := cli.New()
	chartRepo, err := repo.NewChartRepository(&entry, getter.All(settings))
	if err != nil {
		return fmt.Errorf("helm: init chart repository %q: %w", name, err)
	}
	if _, err := chartRepo.DownloadIndexFile(); err != nil {
		return fmt.Errorf("helm: %q is not a valid chart repository or cannot be reached: %w", repoURL, err)
	}

	f.Update(&entry)
	if err := f.WriteFile(repoFile, 0o600); err != nil {
		return fmt.Errorf("helm: write repositories.yaml: %w", err)
	}

	s.cache.invalidateIndex(name)
	return nil
}

// RemoveHelmRepository removes a configured helm repository and its cached index,
// mirroring the Helm CLI's `helm repo remove`.
func (s *Service) RemoveHelmRepository(name string) error {
	repoFile := helmpath.ConfigPath("repositories.yaml")

	f, err := repo.LoadFile(repoFile)
	if err != nil {
		return fmt.Errorf("helm: load repositories.yaml: %w", err)
	}
	if !f.Remove(name) {
		return fmt.Errorf("helm: no repository named %q found", name)
	}
	if err := f.WriteFile(repoFile, 0o600); err != nil {
		return fmt.Errorf("helm: write repositories.yaml: %w", err)
	}

	indexPath := filepath.Join(helmpath.CachePath("repository"), name+"-index.yaml")
	if err := os.Remove(indexPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("helm: remove cached index for %q: %w", name, err)
	}

	s.cache.invalidateIndex(name)
	return nil
}
