// Package rest provides HTTP handlers plus a genericclioptions.RESTClientGetter
// implementation backed by an existing rest.Config, so Helm actions can be wired to
// the active cluster context without re-reading kubeconfig from disk.
package rest

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/discovery"
	memorycache "k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	grpclib "google.golang.org/grpc"

	"github.com/litelensapp/litelens-plugins/plugins/helm/internal/api/grpc"
	"github.com/litelensapp/litelens-plugins/plugins/helm/internal/dto"
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
func NewHttpServer(listen, version string, svc Service) (*HttpServer, error) {
	ln, err := net.Listen("tcp", listen)
	if err != nil {
		return nil, fmt.Errorf("listen for HTTP: %w", err)
	}

	_, portStr, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		ln.Close()
		return nil, fmt.Errorf("parse http listener address: %w", err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		ln.Close()
		return nil, fmt.Errorf("convert http port to int: %w", err)
	}

	mux := http.NewServeMux()
	NewHandler(svc).RegisterRoutes(mux)

	return &HttpServer{
		httpSrv: &http.Server{Handler: corsMiddleware(mux)},
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

// Getter implements genericclioptions.RESTClientGetter using an existing rest.Config.
type Getter struct {
	RC        *rest.Config
	Rules     *clientcmd.ClientConfigLoadingRules
	Overrides *clientcmd.ConfigOverrides
}

func (g *Getter) ToRESTConfig() (*rest.Config, error) {
	return rest.CopyConfig(g.RC), nil
}

func (g *Getter) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	dc, err := discovery.NewDiscoveryClientForConfig(g.RC)
	if err != nil {
		return nil, err
	}
	return memorycache.NewMemCacheClient(dc), nil
}

func (g *Getter) ToRESTMapper() (meta.RESTMapper, error) {
	dc, err := g.ToDiscoveryClient()
	if err != nil {
		return nil, err
	}
	return restmapper.NewDeferredDiscoveryRESTMapper(dc), nil
}

func (g *Getter) ToRawKubeConfigLoader() clientcmd.ClientConfig {
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(g.Rules, g.Overrides)
}

// Service interface matches the methods we need from helm.Service
// We define this in the rest package to avoid circular imports
type Service interface {
	ListHelmCharts() ([]dto.HelmChart, error)
	ListHelmRepositories() ([]dto.HelmRepository, error)
	ListHelmReleases(namespace string) ([]dto.HelmRelease, error)
	ListHelmChartVersions(repository, chartName string) ([]string, error)
	GetHelmChartDetail(repository, chartName, version string) (dto.HelmChartDetail, error)
	GetArtifactHubReadme(repository, chartName, version string) (string, error)
	InstallHelmChart(namespace, releaseName, repository, chartName, version, valuesYAML string) error
	UpgradeHelmRelease(namespace, releaseName, repository, chartName, version, valuesYAML string) error
	DeleteHelmRelease(namespace, releaseName string) error
	DeleteHelmReleaseWithCleanup(namespace, releaseName string) error
	GetHelmReleaseByName(namespace, releaseName string) (*dto.HelmReleaseDetail, error)
	GetHelmChartValues(repository, chartName, version string) (string, error)
	GetHelmReleaseHistory(namespace, releaseName string) ([]dto.HelmReleaseRevisionHistory, error)
	RollbackHelmRelease(namespace, releaseName string, revision int) error
	SetActiveContext(contextName, kubeconfigPath string) error
}

// HostEventEmitter is a type alias for the grpc HostEventEmitter
type HostEventEmitter = grpc.HostEventEmitter

// NewHostConnection establishes a gRPC connection to the host server.
func NewHostConnection(addr string) (*grpclib.ClientConn, error) {
	return grpclib.NewClient(addr, grpclib.WithInsecure())
}

// NewHostEventEmitter creates a new event emitter using an existing connection.
func NewHostEventEmitter(conn grpclib.ClientConnInterface) *HostEventEmitter {
	return grpc.NewHostEventEmitter(conn)
}
