package rest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/litelensapp/litelens-plugins/plugins/helm/internal/dto"
)

// stubService is a minimal Service implementation for testing the HTTP layer
// in isolation from real Helm/Kubernetes calls.
type stubService struct {
	charts      []dto.HelmChart
	chartsErr   error
	releaseByID *dto.HelmReleaseDetail
	releaseErr  error

	setActiveContextCalls int32

	mu                 sync.Mutex
	lastContextName    string
	lastKubeconfigPath string
}

func (s *stubService) ListHelmCharts() ([]dto.HelmChart, error) { return s.charts, s.chartsErr }
func (s *stubService) ListHelmRepositories() ([]dto.HelmRepository, error) {
	return nil, nil
}
func (s *stubService) ListHelmReleases() ([]dto.HelmRelease, error) {
	return nil, nil
}
func (s *stubService) ListHelmChartVersions(repository, chartName string) ([]string, error) {
	return nil, nil
}
func (s *stubService) GetHelmChartDetail(repository, chartName, version string) (dto.HelmChartDetail, error) {
	return dto.HelmChartDetail{}, nil
}
func (s *stubService) GetArtifactHubReadme(repository, chartName, version string) (string, error) {
	return "", nil
}
func (s *stubService) InstallHelmChart(namespace, releaseName, repository, chartName, version, valuesYAML string) error {
	return nil
}
func (s *stubService) UpgradeHelmRelease(namespace, releaseName, repository, chartName, version, valuesYAML string) error {
	return nil
}
func (s *stubService) DeleteHelmRelease(namespace, releaseName string) error { return nil }
func (s *stubService) DeleteHelmReleaseWithCleanup(namespace, releaseName string) error {
	return nil
}
func (s *stubService) GetHelmReleaseByName(namespace, releaseName string) (*dto.HelmReleaseDetail, error) {
	return s.releaseByID, s.releaseErr
}
func (s *stubService) GetHelmChartValues(repository, chartName, version string) (string, error) {
	return "", nil
}
func (s *stubService) GetHelmReleaseHistory(namespace, releaseName string) ([]dto.HelmReleaseRevisionHistory, error) {
	return nil, nil
}
func (s *stubService) RollbackHelmRelease(namespace, releaseName string, revision int) error {
	return nil
}
func (s *stubService) SetActiveContext(contextName, kubeconfigPath string) error {
	atomic.AddInt32(&s.setActiveContextCalls, 1)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastContextName = contextName
	s.lastKubeconfigPath = kubeconfigPath
	return nil
}

func newTestServer(svc *stubService) *httptest.Server {
	mux := http.NewServeMux()
	NewHandler(svc).RegisterRoutes(mux)
	return httptest.NewServer(mux)
}

func TestListCharts_Success(t *testing.T) {
	svc := &stubService{charts: []dto.HelmChart{{Name: "nginx"}}}
	srv := newTestServer(svc)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/api/helm/listCharts", "application/json", bytes.NewReader(nil))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var got []dto.HelmChart
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(got) != 1 || got[0].Name != "nginx" {
		t.Fatalf("unexpected body: %+v", got)
	}
}

func TestListCharts_ServiceError(t *testing.T) {
	svc := &stubService{chartsErr: fmt.Errorf("cluster client not ready")}
	srv := newTestServer(svc)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/api/helm/listCharts", "application/json", bytes.NewReader(nil))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", resp.StatusCode)
	}
	var got ErrorResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Code != "PLUGIN_UNAVAILABLE" {
		t.Fatalf("expected PLUGIN_UNAVAILABLE, got %q", got.Code)
	}
}

func TestGetReleaseByName_NotFound(t *testing.T) {
	svc := &stubService{releaseByID: nil, releaseErr: nil}
	srv := newTestServer(svc)
	defer srv.Close()

	body, _ := json.Marshal(map[string]string{"Namespace": "default", "ReleaseName": "missing"})
	resp, err := http.Post(srv.URL+"/api/helm/getReleaseByName", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
	var got ErrorResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Code != "NOT_FOUND" {
		t.Fatalf("expected NOT_FOUND, got %q", got.Code)
	}
}

