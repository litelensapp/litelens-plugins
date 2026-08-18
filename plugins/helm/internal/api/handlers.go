package api

import (
	"encoding/json"
	"net/http"

	"github.com/litelensapp/litelens-plugins/plugins/helm/internal/server"
)

// Handler exposes helm.Service business methods over plain HTTP POST endpoints,
// mirroring internal/server/grpc.go's dispatch() switch so the plugin frontend can
// eventually call this backend directly over localhost instead of the host's gRPC relay.
type Handler struct {
	svc server.Service
}

func NewHandler(svc server.Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes wires all business + control endpoints onto mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/helm/listCharts", h.listCharts)
	mux.HandleFunc("POST /api/helm/listRepositories", h.listRepositories)
	mux.HandleFunc("POST /api/helm/listReleases", h.listReleases)
	mux.HandleFunc("POST /api/helm/listChartVersions", h.listChartVersions)
	mux.HandleFunc("POST /api/helm/getChartDetail", h.getChartDetail)
	mux.HandleFunc("POST /api/helm/getArtifactHubReadme", h.getArtifactHubReadme)
	mux.HandleFunc("POST /api/helm/installChart", h.installChart)
	mux.HandleFunc("POST /api/helm/upgradeRelease", h.upgradeRelease)
	mux.HandleFunc("POST /api/helm/deleteRelease", h.deleteRelease)
	mux.HandleFunc("POST /api/helm/deleteReleaseWithCleanup", h.deleteReleaseWithCleanup)
	mux.HandleFunc("POST /api/helm/getReleaseByName", h.getReleaseByName)
	mux.HandleFunc("POST /api/helm/getChartValues", h.getChartValues)
	mux.HandleFunc("POST /api/helm/getReleaseHistory", h.getReleaseHistory)
	mux.HandleFunc("POST /api/helm/rollbackRelease", h.rollbackRelease)
	mux.HandleFunc("POST /internal/setClusterContext", h.setClusterContext)
}

func decodeBody(r *http.Request, v any) bool {
	return json.NewDecoder(r.Body).Decode(v) == nil
}

func (h *Handler) listCharts(w http.ResponseWriter, r *http.Request) {
	result, err := h.svc.ListHelmCharts()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "PLUGIN_UNAVAILABLE", err.Error())
		return
	}
	writeJSON(w, result)
}

func (h *Handler) listRepositories(w http.ResponseWriter, r *http.Request) {
	result, err := h.svc.ListHelmRepositories()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "PLUGIN_UNAVAILABLE", err.Error())
		return
	}
	writeJSON(w, result)
}

func (h *Handler) listReleases(w http.ResponseWriter, r *http.Request) {
	var req struct{ Namespace string }
	if !decodeBody(r, &req) {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	result, err := h.svc.ListHelmReleases(req.Namespace)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "PLUGIN_UNAVAILABLE", err.Error())
		return
	}
	writeJSON(w, result)
}

func (h *Handler) listChartVersions(w http.ResponseWriter, r *http.Request) {
	var req struct{ Repository, ChartName string }
	if !decodeBody(r, &req) {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	result, err := h.svc.ListHelmChartVersions(req.Repository, req.ChartName)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "PLUGIN_UNAVAILABLE", err.Error())
		return
	}
	writeJSON(w, result)
}

func (h *Handler) getChartDetail(w http.ResponseWriter, r *http.Request) {
	var req struct{ Repository, ChartName, Version string }
	if !decodeBody(r, &req) {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	result, err := h.svc.GetHelmChartDetail(req.Repository, req.ChartName, req.Version)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "PLUGIN_UNAVAILABLE", err.Error())
		return
	}
	writeJSON(w, result)
}

func (h *Handler) getArtifactHubReadme(w http.ResponseWriter, r *http.Request) {
	var req struct{ Repository, ChartName, Version string }
	if !decodeBody(r, &req) {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	result, err := h.svc.GetArtifactHubReadme(req.Repository, req.ChartName, req.Version)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "PLUGIN_UNAVAILABLE", err.Error())
		return
	}
	writeJSON(w, result)
}

func (h *Handler) installChart(w http.ResponseWriter, r *http.Request) {
	var req struct{ Namespace, ReleaseName, Repository, ChartName, Version, ValuesYAML string }
	if !decodeBody(r, &req) {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	if err := h.svc.InstallHelmChart(req.Namespace, req.ReleaseName, req.Repository, req.ChartName, req.Version, req.ValuesYAML); err != nil {
		writeError(w, http.StatusServiceUnavailable, "PLUGIN_UNAVAILABLE", err.Error())
		return
	}
	writeJSON(w, struct{}{})
}

func (h *Handler) upgradeRelease(w http.ResponseWriter, r *http.Request) {
	var req struct{ Namespace, ReleaseName, Repository, ChartName, Version, ValuesYAML string }
	if !decodeBody(r, &req) {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	if err := h.svc.UpgradeHelmRelease(req.Namespace, req.ReleaseName, req.Repository, req.ChartName, req.Version, req.ValuesYAML); err != nil {
		writeError(w, http.StatusServiceUnavailable, "PLUGIN_UNAVAILABLE", err.Error())
		return
	}
	writeJSON(w, struct{}{})
}

func (h *Handler) deleteRelease(w http.ResponseWriter, r *http.Request) {
	var req struct{ Namespace, ReleaseName string }
	if !decodeBody(r, &req) {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	if err := h.svc.DeleteHelmRelease(req.Namespace, req.ReleaseName); err != nil {
		writeError(w, http.StatusServiceUnavailable, "PLUGIN_UNAVAILABLE", err.Error())
		return
	}
	writeJSON(w, struct{}{})
}

func (h *Handler) deleteReleaseWithCleanup(w http.ResponseWriter, r *http.Request) {
	var req struct{ Namespace, ReleaseName string }
	if !decodeBody(r, &req) {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	if err := h.svc.DeleteHelmReleaseWithCleanup(req.Namespace, req.ReleaseName); err != nil {
		writeError(w, http.StatusServiceUnavailable, "PLUGIN_UNAVAILABLE", err.Error())
		return
	}
	writeJSON(w, struct{}{})
}

func (h *Handler) getReleaseByName(w http.ResponseWriter, r *http.Request) {
	var req struct{ Namespace, ReleaseName string }
	if !decodeBody(r, &req) {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	result, err := h.svc.GetHelmReleaseByName(req.Namespace, req.ReleaseName)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "PLUGIN_UNAVAILABLE", err.Error())
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
	if !decodeBody(r, &req) {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	result, err := h.svc.GetHelmChartValues(req.Repository, req.ChartName, req.Version)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "PLUGIN_UNAVAILABLE", err.Error())
		return
	}
	writeJSON(w, result)
}

func (h *Handler) getReleaseHistory(w http.ResponseWriter, r *http.Request) {
	var req struct{ Namespace, ReleaseName string }
	if !decodeBody(r, &req) {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	result, err := h.svc.GetHelmReleaseHistory(req.Namespace, req.ReleaseName)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "PLUGIN_UNAVAILABLE", err.Error())
		return
	}
	writeJSON(w, result)
}

func (h *Handler) rollbackRelease(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Namespace, ReleaseName string
		Revision               int
	}
	if !decodeBody(r, &req) {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	if err := h.svc.RollbackHelmRelease(req.Namespace, req.ReleaseName, req.Revision); err != nil {
		writeError(w, http.StatusServiceUnavailable, "PLUGIN_UNAVAILABLE", err.Error())
		return
	}
	writeJSON(w, struct{}{})
}

// setClusterContext handles the host's push of the active cluster context.
// Localhost-only, no auth (same trust model as the business endpoints). Idempotent.
func (h *Handler) setClusterContext(w http.ResponseWriter, r *http.Request) {
	var req struct{ ContextName, KubeconfigPath string }
	if !decodeBody(r, &req) {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	if err := h.svc.SetActiveContext(req.ContextName, req.KubeconfigPath); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, struct{}{})
}
