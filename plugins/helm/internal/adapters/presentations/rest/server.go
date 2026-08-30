// Package rest provides the plugin's HTTP business API — localhost-only handlers
// wired to the port.HelmService interface.
package rest

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/litelensapp/litelens-plugins/plugins/helm/internal/applications/port"
)

// HttpServer wraps the plugin's HTTP server — localhost-only. The Wails webview
// loads the host app UI from its own origin (e.g. http://wails.localhost:34115 in
// dev, a custom scheme in production), so a fetch() to this server's 127.0.0.1:port
// origin IS cross-origin from the browser's perspective and triggers a CORS
// preflight; corsMiddleware answers it and tags real responses so the browser
// accepts them.
type HttpServer struct {
	httpSrv *http.Server
	ln      net.Listener
	version string

	// Port is the resolved listening port (useful when listen uses port 0 to auto-assign).
	Port int
}

// NewHttpServer binds a listener on listen and wires svc's routes onto it, but does not
// start serving — call Serve to block and accept connections.
func NewHttpServer(listen, version string, svc port.HelmService) (*HttpServer, error) {
	ln, err := net.Listen("tcp", listen)
	if err != nil {
		return nil, fmt.Errorf("listen for HTTP: %w", err)
	}

	host, portStr, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		ln.Close()
		return nil, fmt.Errorf("parse http listener address: %w", err)
	}

	// CORS hardening: ensure the server only listens on localhost.
	// The corsMiddleware reflects the Origin header, which is safe only when the listener
	// is localhost-only and cannot be reached by untrusted clients.
	if host != "127.0.0.1" && host != "::1" && host != "localhost" {
		ln.Close()
		return nil, fmt.Errorf("CORS hardening: listener must be localhost-only; got %q", host)
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		ln.Close()
		return nil, fmt.Errorf("convert http port to int: %w", err)
	}

	router := chi.NewRouter()
	router.Use(corsMiddleware)

	h := NewHandler(svc)
	router.Post("/api/helm/listCharts", h.listCharts)
	router.Post("/api/helm/listRepositories", h.listRepositories)
	router.Post("/api/helm/listReleases", h.listReleases)
	router.Post("/api/helm/listChartVersions", h.listChartVersions)
	router.Post("/api/helm/getChartDetail", h.getChartDetail)
	router.Post("/api/helm/getArtifactHubReadme", h.getArtifactHubReadme)
	router.Post("/api/helm/installChart", h.installChart)
	router.Post("/api/helm/upgradeRelease", h.upgradeRelease)
	router.Post("/api/helm/deleteRelease", h.deleteRelease)
	router.Post("/api/helm/deleteReleaseWithCleanup", h.deleteReleaseWithCleanup)
	router.Post("/api/helm/getReleaseByName", h.getReleaseByName)
	router.Post("/api/helm/getChartValues", h.getChartValues)
	router.Post("/api/helm/getReleaseHistory", h.getReleaseHistory)
	router.Post("/api/helm/rollbackRelease", h.rollbackRelease)
	router.Post("/api/helm/searchRepositoryCatalog", h.searchRepositoryCatalog)
	router.Post("/api/helm/addRepository", h.addRepository)
	router.Post("/api/helm/removeRepository", h.removeRepository)

	return &HttpServer{
		httpSrv: &http.Server{Handler: router},
		ln:      ln,
		version: version,
		Port:    port,
	}, nil
}

// corsMiddleware answers CORS preflight requests and tags every response so the
// Wails webview (a different origin than this loopback server) is allowed to read
// the result. The server only ever listens on 127.0.0.1, so the origin is
// reflected rather than restricted to a fixed value — dev and production Wails
// builds use different origins/schemes and there's no other client that could
// reach this port.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Serve emits the readiness handshake — exactly one JSON line to stdout, since that's
// the only way the host (which launched this as a subprocess with an auto-assigned
// port) learns which port to talk to — then blocks, accepting connections until Close
// is called.
func (s *HttpServer) Serve() error {
	handshake := map[string]any{
		"type":      "READY",
		"version":   s.version,
		"httpPort":  s.Port,
		"pid":       os.Getpid(),
		"timestamp": time.Now().Format(time.RFC3339),
	}
	data, _ := json.Marshal(handshake)
	fmt.Println(string(data))
	os.Stdout.Sync()

	return s.httpSrv.Serve(s.ln)
}

// Close shuts down the server, releasing the listener.
func (s *HttpServer) Close() error {
	return s.httpSrv.Close()
}
