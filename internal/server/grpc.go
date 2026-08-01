package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net"

	"google.golang.org/grpc"

	"github.com/gknguyen/litelens/internal/dto"
	"github.com/gknguyen/litelens/internal/plugin/pb"
)

// Service interface matches the methods we need from helm.Service
// We define this in the server package to avoid circular imports
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

// GRPCServer implements pb.PluginServer by wrapping Service methods.
type GRPCServer struct {
	pb.UnimplementedPluginServer
	svc Service
}

// NewGRPCServer creates a new gRPC server, registers it, and binds to listen.
func NewGRPCServer(svc Service, listen string) (*grpc.Server, net.Listener, error) {
	// Create listener
	ln, err := net.Listen("tcp", listen)
	if err != nil {
		return nil, nil, fmt.Errorf("listen on %s: %w", listen, err)
	}

	// Create gRPC server
	srv := grpc.NewServer()
	pb.RegisterPluginServer(srv, &GRPCServer{svc: svc})

	return srv, ln, nil
}

// GetCapabilities implements pb.PluginServer
func (s *GRPCServer) GetCapabilities(ctx context.Context, _ *pb.Empty) (*pb.CapabilitiesResponse, error) {
	return &pb.CapabilitiesResponse{
		Version: "dev",
		Ready:   true,
		Features: []string{
			"list-charts",
			"list-repositories",
			"list-releases",
			"install",
			"upgrade",
			"delete",
		},
	}, nil
}

// SetClusterContext implements pb.PluginServer
func (s *GRPCServer) SetClusterContext(ctx context.Context, req *pb.SetClusterContextRequest) (*pb.Empty, error) {
	if err := s.svc.SetActiveContext(req.ContextName, req.KubeconfigPath); err != nil {
		return nil, err
	}
	return &pb.Empty{}, nil
}

// Invoke implements pb.PluginServer
func (s *GRPCServer) Invoke(ctx context.Context, req *pb.InvokeRequest) (*pb.InvokeResponse, error) {
	result, err := s.dispatch(req.Method, req.PayloadJson)
	if err != nil {
		return &pb.InvokeResponse{Error: err.Error()}, nil
	}
	return &pb.InvokeResponse{PayloadJson: result}, nil
}

// marshalResult is a helper to marshal result values to JSON strings
func marshalResult(v any, err error) (string, error) {
	if err != nil {
		return "", err
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// dispatch routes method calls to the appropriate service method and marshals responses
func (s *GRPCServer) dispatch(method, payloadJSON string) (string, error) {
	switch method {
	case "ListHelmCharts":
		return marshalResult(s.svc.ListHelmCharts())
	case "ListHelmRepositories":
		return marshalResult(s.svc.ListHelmRepositories())
	case "ListHelmReleases":
		var req struct{ Namespace string }
		if err := json.Unmarshal([]byte(payloadJSON), &req); err != nil {
			return "", err
		}
		return marshalResult(s.svc.ListHelmReleases(req.Namespace))
	case "ListHelmChartVersions":
		var req struct{ Repository, ChartName string }
		if err := json.Unmarshal([]byte(payloadJSON), &req); err != nil {
			return "", err
		}
		return marshalResult(s.svc.ListHelmChartVersions(req.Repository, req.ChartName))
	case "GetHelmChartDetail":
		var req struct{ Repository, ChartName, Version string }
		if err := json.Unmarshal([]byte(payloadJSON), &req); err != nil {
			return "", err
		}
		return marshalResult(s.svc.GetHelmChartDetail(req.Repository, req.ChartName, req.Version))
	case "GetArtifactHubReadme":
		var req struct{ Repository, ChartName, Version string }
		if err := json.Unmarshal([]byte(payloadJSON), &req); err != nil {
			return "", err
		}
		return marshalResult(s.svc.GetArtifactHubReadme(req.Repository, req.ChartName, req.Version))
	case "InstallHelmChart":
		var req struct{ Namespace, ReleaseName, Repository, ChartName, Version, ValuesYAML string }
		if err := json.Unmarshal([]byte(payloadJSON), &req); err != nil {
			return "", err
		}
		return marshalResult(struct{}{}, s.svc.InstallHelmChart(req.Namespace, req.ReleaseName, req.Repository, req.ChartName, req.Version, req.ValuesYAML))
	case "UpgradeHelmRelease":
		var req struct{ Namespace, ReleaseName, Repository, ChartName, Version, ValuesYAML string }
		if err := json.Unmarshal([]byte(payloadJSON), &req); err != nil {
			return "", err
		}
		return marshalResult(struct{}{}, s.svc.UpgradeHelmRelease(req.Namespace, req.ReleaseName, req.Repository, req.ChartName, req.Version, req.ValuesYAML))
	case "DeleteHelmRelease":
		var req struct{ Namespace, ReleaseName string }
		if err := json.Unmarshal([]byte(payloadJSON), &req); err != nil {
			return "", err
		}
		return marshalResult(struct{}{}, s.svc.DeleteHelmRelease(req.Namespace, req.ReleaseName))
	case "DeleteHelmReleaseWithCleanup":
		var req struct{ Namespace, ReleaseName string }
		if err := json.Unmarshal([]byte(payloadJSON), &req); err != nil {
			return "", err
		}
		return marshalResult(struct{}{}, s.svc.DeleteHelmReleaseWithCleanup(req.Namespace, req.ReleaseName))
	case "GetHelmReleaseByName":
		var req struct{ Namespace, ReleaseName string }
		if err := json.Unmarshal([]byte(payloadJSON), &req); err != nil {
			return "", err
		}
		detail, err := s.svc.GetHelmReleaseByName(req.Namespace, req.ReleaseName)
		if err != nil {
			return "", err
		}
		if detail == nil {
			return "", fmt.Errorf("release not found")
		}
		return marshalResult(detail, nil)
	case "GetHelmChartValues":
		var req struct{ Repository, ChartName, Version string }
		if err := json.Unmarshal([]byte(payloadJSON), &req); err != nil {
			return "", err
		}
		return marshalResult(s.svc.GetHelmChartValues(req.Repository, req.ChartName, req.Version))
	case "GetHelmReleaseHistory":
		var req struct{ Namespace, ReleaseName string }
		if err := json.Unmarshal([]byte(payloadJSON), &req); err != nil {
			return "", err
		}
		return marshalResult(s.svc.GetHelmReleaseHistory(req.Namespace, req.ReleaseName))
	case "RollbackHelmRelease":
		var req struct {
			Namespace, ReleaseName string
			Revision               int
		}
		if err := json.Unmarshal([]byte(payloadJSON), &req); err != nil {
			return "", err
		}
		return marshalResult(struct{}{}, s.svc.RollbackHelmRelease(req.Namespace, req.ReleaseName, req.Revision))
	default:
		return "", fmt.Errorf("unknown method %q", method)
	}
}

// NewHandler is an exported helper that creates a PluginServer from a Service.
// Used by the helm package to embed this plugin in-process without needing to know about internal/server.
func NewHandler(svc Service) pb.PluginServer {
	return &GRPCServer{svc: svc}
}
