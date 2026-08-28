package rest

import (
	"fmt"
	"net/http"
	"os"

	"github.com/litelensapp/litelens-plugins/plugins/helm/internal/applications/port"
)

// Handler exposes helm.Service business methods over plain HTTP POST endpoints.
type Handler struct {
	svc port.HelmService
}

func NewHandler(svc port.HelmService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) listCharts(w http.ResponseWriter, r *http.Request) {
	result, err := h.svc.ListHelmCharts()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: list charts: %v\n", err)
		writeError(w, http.StatusServiceUnavailable, "PLUGIN_UNAVAILABLE", "failed to list charts")
		return
	}
	writeJSON(w, result)
}

func (h *Handler) listRepositories(w http.ResponseWriter, r *http.Request) {
	result, err := h.svc.ListHelmRepositories()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: list repositories: %v\n", err)
		writeError(w, http.StatusServiceUnavailable, "PLUGIN_UNAVAILABLE", "failed to list repositories")
		return
	}
	writeJSON(w, result)
}

func (h *Handler) listReleases(w http.ResponseWriter, r *http.Request) {
	result, err := h.svc.ListHelmReleases()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: list releases: %v\n", err)
		writeError(w, http.StatusServiceUnavailable, "PLUGIN_UNAVAILABLE", "failed to list releases")
		return
	}
	writeJSON(w, result)
}

func (h *Handler) listChartVersions(w http.ResponseWriter, r *http.Request) {
	var req struct{ Repository, ChartName string }
	if ok, err := decodeBody(r, &req); !ok {
		fmt.Fprintf(os.Stderr, "error: decode list chart versions request: %v\n", err)
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	result, err := h.svc.ListHelmChartVersions(req.Repository, req.ChartName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: list chart versions: %v\n", err)
		writeError(w, http.StatusServiceUnavailable, "PLUGIN_UNAVAILABLE", "failed to list chart versions")
		return
	}
	writeJSON(w, result)
}

func (h *Handler) getChartDetail(w http.ResponseWriter, r *http.Request) {
	var req struct{ Repository, ChartName, Version string }
	if ok, err := decodeBody(r, &req); !ok {
		fmt.Fprintf(os.Stderr, "error: decode get chart detail request: %v\n", err)
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	result, err := h.svc.GetHelmChartDetail(req.Repository, req.ChartName, req.Version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: get chart detail: %v\n", err)
		writeError(w, http.StatusServiceUnavailable, "PLUGIN_UNAVAILABLE", "failed to get chart details")
		return
	}
	writeJSON(w, result)
}

func (h *Handler) getArtifactHubReadme(w http.ResponseWriter, r *http.Request) {
	var req struct{ Repository, ChartName, Version string }
	if ok, err := decodeBody(r, &req); !ok {
		fmt.Fprintf(os.Stderr, "error: decode get artifact hub readme request: %v\n", err)
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	result, err := h.svc.GetArtifactHubReadme(req.Repository, req.ChartName, req.Version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: get artifact hub readme: %v\n", err)
		writeError(w, http.StatusServiceUnavailable, "PLUGIN_UNAVAILABLE", "failed to fetch readme")
		return
	}
	writeJSON(w, result)
}

func (h *Handler) installChart(w http.ResponseWriter, r *http.Request) {
	var req struct{ Namespace, ReleaseName, Repository, ChartName, Version, ValuesYAML string }
	if ok, err := decodeBody(r, &req); !ok {
		fmt.Fprintf(os.Stderr, "error: decode install chart request: %v\n", err)
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	relName, err := h.svc.InstallHelmChart(req.Namespace, req.ReleaseName, req.Repository, req.ChartName, req.Version, req.ValuesYAML)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: install chart: %v\n", err)
		writeError(w, http.StatusServiceUnavailable, "PLUGIN_UNAVAILABLE", "failed to install chart")
		return
	}
	writeJSON(w, struct{ ReleaseName string }{ReleaseName: relName})
}

func (h *Handler) upgradeRelease(w http.ResponseWriter, r *http.Request) {
	var req struct{ Namespace, ReleaseName, Repository, ChartName, Version, ValuesYAML string }
	if ok, err := decodeBody(r, &req); !ok {
		fmt.Fprintf(os.Stderr, "error: decode upgrade release request: %v\n", err)
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	if err := h.svc.UpgradeHelmRelease(req.Namespace, req.ReleaseName, req.Repository, req.ChartName, req.Version, req.ValuesYAML); err != nil {
		fmt.Fprintf(os.Stderr, "error: upgrade release: %v\n", err)
		writeError(w, http.StatusServiceUnavailable, "PLUGIN_UNAVAILABLE", "failed to upgrade release")
		return
	}
	writeJSON(w, struct{}{})
}

func (h *Handler) deleteRelease(w http.ResponseWriter, r *http.Request) {
	var req struct{ Namespace, ReleaseName string }
	if ok, err := decodeBody(r, &req); !ok {
		fmt.Fprintf(os.Stderr, "error: decode delete release request: %v\n", err)
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	if err := h.svc.DeleteHelmRelease(req.Namespace, req.ReleaseName); err != nil {
		fmt.Fprintf(os.Stderr, "error: delete release: %v\n", err)
		writeError(w, http.StatusServiceUnavailable, "PLUGIN_UNAVAILABLE", "failed to delete release")
		return
	}
	writeJSON(w, struct{}{})
}

func (h *Handler) deleteReleaseWithCleanup(w http.ResponseWriter, r *http.Request) {
	var req struct{ Namespace, ReleaseName string }
	if ok, err := decodeBody(r, &req); !ok {
		fmt.Fprintf(os.Stderr, "error: decode delete release with cleanup request: %v\n", err)
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	if err := h.svc.DeleteHelmReleaseWithCleanup(req.Namespace, req.ReleaseName); err != nil {
		fmt.Fprintf(os.Stderr, "error: delete release with cleanup: %v\n", err)
		writeError(w, http.StatusServiceUnavailable, "PLUGIN_UNAVAILABLE", "failed to delete release")
		return
	}
	writeJSON(w, struct{}{})
}

func (h *Handler) getReleaseByName(w http.ResponseWriter, r *http.Request) {
	var req struct{ Namespace, ReleaseName string }
	if ok, err := decodeBody(r, &req); !ok {
		fmt.Fprintf(os.Stderr, "error: decode get release by name request: %v\n", err)
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	result, err := h.svc.GetHelmReleaseByName(req.Namespace, req.ReleaseName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: get release by name: %v\n", err)
		writeError(w, http.StatusServiceUnavailable, "PLUGIN_UNAVAILABLE", "failed to get release")
		return
	}
	if result == nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "release not found")
		return
	}
	writeJSON(w, result)
}

func (h *Handler) getChartValues(w http.ResponseWriter, r *http.Request) {
	var req struct{ Repository, ChartName, Version string }
	if ok, err := decodeBody(r, &req); !ok {
		fmt.Fprintf(os.Stderr, "error: decode get chart values request: %v\n", err)
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	result, err := h.svc.GetHelmChartValues(req.Repository, req.ChartName, req.Version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: get chart values: %v\n", err)
		writeError(w, http.StatusServiceUnavailable, "PLUGIN_UNAVAILABLE", "failed to get chart values")
		return
	}
	writeJSON(w, result)
}

func (h *Handler) getReleaseHistory(w http.ResponseWriter, r *http.Request) {
	var req struct{ Namespace, ReleaseName string }
	if ok, err := decodeBody(r, &req); !ok {
		fmt.Fprintf(os.Stderr, "error: decode get release history request: %v\n", err)
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	result, err := h.svc.GetHelmReleaseHistory(req.Namespace, req.ReleaseName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: get release history: %v\n", err)
		writeError(w, http.StatusServiceUnavailable, "PLUGIN_UNAVAILABLE", "failed to get release history")
		return
	}
	writeJSON(w, result)
}

func (h *Handler) rollbackRelease(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Namespace, ReleaseName string
		Revision               int
	}
	if ok, err := decodeBody(r, &req); !ok {
		fmt.Fprintf(os.Stderr, "error: decode rollback release request: %v\n", err)
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	if err := h.svc.RollbackHelmRelease(req.Namespace, req.ReleaseName, req.Revision); err != nil {
		fmt.Fprintf(os.Stderr, "error: rollback release: %v\n", err)
		writeError(w, http.StatusServiceUnavailable, "PLUGIN_UNAVAILABLE", "failed to rollback release")
		return
	}
	writeJSON(w, struct{}{})
}
