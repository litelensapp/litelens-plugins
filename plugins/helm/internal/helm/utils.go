package helm

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/litelensapp/litelens-plugins/plugins/helm/internal/dto"
	helmchart "helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/helmpath"
	"helm.sh/helm/v3/pkg/registry"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

// mergedValuesYAML merges a release's chart defaults with user-supplied overrides
// and marshals the result to YAML. Returns "" if the chart is nil or marshal fails.
func mergedValuesYAML(rel *release.Release) string {
	if rel == nil || rel.Chart == nil {
		return ""
	}
	merged := make(map[string]any, len(rel.Chart.Values))
	maps.Copy(merged, rel.Chart.Values)
	maps.Copy(merged, rel.Config)
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

// helmHasVerb reports whether the given verb appears in the resource's verb list.
func helmHasVerb(verbs metav1.Verbs, verb string) bool {
	return slices.Contains(verbs, verb)
}

// parseManifestResources splits a multi-document Helm manifest and extracts
// kind/name/namespace from each document, skipping empty or comment-only docs.
// Namespaced resource templates frequently omit `metadata.namespace` and rely
// on the release namespace instead, so releaseNamespace is used as a fallback.
func parseManifestResources(manifest, releaseNamespace string) []dto.HelmReleaseResource {
	var out []dto.HelmReleaseResource
	for doc := range strings.SplitSeq(manifest, "\n---") {
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
