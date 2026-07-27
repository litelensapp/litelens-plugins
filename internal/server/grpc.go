package server

import (
	"context"
	"fmt"
	"net"

	"google.golang.org/grpc"

	"github.com/gknguyen/litelens/plugins/helm"
	"github.com/gknguyen/litelens/plugins/helm/pb"
)

// GRPCServer implements pb.HelmServer by wrapping helm.Service methods.
type GRPCServer struct {
	pb.UnimplementedHelmServer
	svc *helm.Service
}

// NewGRPCServer creates a new gRPC server, registers it, and binds to listen.
func NewGRPCServer(svc *helm.Service, listen string) (*grpc.Server, net.Listener, error) {
	// Create listener
	ln, err := net.Listen("tcp", listen)
	if err != nil {
		return nil, nil, fmt.Errorf("listen on %s: %w", listen, err)
	}

	// Create gRPC server
	srv := grpc.NewServer()
	pb.RegisterHelmServer(srv, &GRPCServer{svc: svc})

	return srv, ln, nil
}

// ListHelmCharts implements pb.HelmServer
func (s *GRPCServer) ListHelmCharts(ctx context.Context, _ *pb.Empty) (*pb.ListHelmChartsResponse, error) {
	charts, err := s.svc.ListHelmCharts()
	if err != nil {
		return nil, err
	}
	result := make([]*pb.HelmChart, 0, len(charts))
	for _, c := range charts {
		result = append(result, &pb.HelmChart{
			Name:        c.Name,
			Description: c.Description,
			Version:     c.Version,
			AppVersion:  c.AppVersion,
			Repository:  c.Repository,
			Icon:        c.Icon,
		})
	}
	return &pb.ListHelmChartsResponse{Charts: result}, nil
}

// ListHelmRepositories implements pb.HelmServer
func (s *GRPCServer) ListHelmRepositories(ctx context.Context, _ *pb.Empty) (*pb.ListHelmRepositoriesResponse, error) {
	repos, err := s.svc.ListHelmRepositories()
	if err != nil {
		return nil, err
	}
	result := make([]*pb.HelmRepository, 0, len(repos))
	for _, r := range repos {
		result = append(result, &pb.HelmRepository{
			Name: r.Name,
			Url:  r.URL,
		})
	}
	return &pb.ListHelmRepositoriesResponse{Repositories: result}, nil
}

// ListHelmReleases implements pb.HelmServer
func (s *GRPCServer) ListHelmReleases(ctx context.Context, req *pb.ListHelmReleasesRequest) (*pb.ListHelmReleasesResponse, error) {
	releases, err := s.svc.ListHelmReleases(req.Namespace)
	if err != nil {
		return nil, err
	}
	result := make([]*pb.HelmRelease, 0, len(releases))
	for _, r := range releases {
		result = append(result, &pb.HelmRelease{
			Name:              r.Name,
			Namespace:         r.Namespace,
			Chart:             r.Chart,
			ChartVersion:      r.ChartVersion,
			AppVersion:        r.AppVersion,
			Status:            r.Status,
			Revision:          int32(r.Revision),
			Updated:           r.Updated,
			UpdatedAt:         r.UpdatedAt,
			Repository:        r.Repository,
			EncodedValuesYAML: r.EncodedValuesYAML,
		})
	}
	return &pb.ListHelmReleasesResponse{Releases: result}, nil
}

// ListHelmChartVersions implements pb.HelmServer
func (s *GRPCServer) ListHelmChartVersions(ctx context.Context, req *pb.ListHelmChartVersionsRequest) (*pb.ListHelmChartVersionsResponse, error) {
	versions, err := s.svc.ListHelmChartVersions(req.Repository, req.ChartName)
	if err != nil {
		return nil, err
	}
	return &pb.ListHelmChartVersionsResponse{Versions: versions}, nil
}

// GetHelmChartDetail implements pb.HelmServer
func (s *GRPCServer) GetHelmChartDetail(ctx context.Context, req *pb.GetHelmChartDetailRequest) (*pb.HelmChartDetail, error) {
	detail, err := s.svc.GetHelmChartDetail(req.Repository, req.ChartName, req.Version)
	if err != nil {
		return nil, err
	}
	return &pb.HelmChartDetail{
		Name:        detail.Name,
		Description: detail.Description,
		Version:     detail.Version,
		AppVersion:  detail.AppVersion,
		Repository:  detail.Repository,
		Icon:        detail.Icon,
		Home:        detail.Home,
		Keywords:    detail.Keywords,
		Sources:     detail.Sources,
		Maintainers: detail.Maintainers,
	}, nil
}

// GetArtifactHubReadme implements pb.HelmServer
func (s *GRPCServer) GetArtifactHubReadme(ctx context.Context, req *pb.GetArtifactHubReadmeRequest) (*pb.GetArtifactHubReadmeResponse, error) {
	readme, err := s.svc.GetArtifactHubReadme(req.Repo, req.ChartName, req.Version)
	if err != nil {
		return nil, err
	}
	return &pb.GetArtifactHubReadmeResponse{Readme: readme}, nil
}

// InstallHelmChart implements pb.HelmServer
func (s *GRPCServer) InstallHelmChart(ctx context.Context, req *pb.InstallHelmChartRequest) (*pb.Empty, error) {
	err := s.svc.InstallHelmChart(req.Namespace, req.ReleaseName, req.Repository, req.ChartName, req.Version, req.ValuesYAML)
	if err != nil {
		return nil, err
	}
	return &pb.Empty{}, nil
}

// UpgradeHelmRelease implements pb.HelmServer
func (s *GRPCServer) UpgradeHelmRelease(ctx context.Context, req *pb.UpgradeHelmReleaseRequest) (*pb.Empty, error) {
	err := s.svc.UpgradeHelmRelease(req.Namespace, req.ReleaseName, req.Repository, req.ChartName, req.Version, req.ValuesYAML)
	if err != nil {
		return nil, err
	}
	return &pb.Empty{}, nil
}

// DeleteHelmRelease implements pb.HelmServer
func (s *GRPCServer) DeleteHelmRelease(ctx context.Context, req *pb.DeleteHelmReleaseRequest) (*pb.Empty, error) {
	err := s.svc.DeleteHelmRelease(req.Namespace, req.ReleaseName)
	if err != nil {
		return nil, err
	}
	return &pb.Empty{}, nil
}

// DeleteHelmReleaseWithCleanup implements pb.HelmServer
func (s *GRPCServer) DeleteHelmReleaseWithCleanup(ctx context.Context, req *pb.DeleteHelmReleaseRequest) (*pb.Empty, error) {
	err := s.svc.DeleteHelmReleaseWithCleanup(req.Namespace, req.ReleaseName)
	if err != nil {
		return nil, err
	}
	return &pb.Empty{}, nil
}

// GetHelmReleaseByName implements pb.HelmServer
func (s *GRPCServer) GetHelmReleaseByName(ctx context.Context, req *pb.GetHelmReleaseByNameRequest) (*pb.HelmReleaseDetail, error) {
	detail, err := s.svc.GetHelmReleaseByName(req.Namespace, req.ReleaseName)
	if err != nil {
		return nil, err
	}
	if detail == nil {
		return nil, fmt.Errorf("release not found")
	}

	resources := make([]*pb.HelmReleaseResource, 0, len(detail.Resources))
	for _, r := range detail.Resources {
		resources = append(resources, &pb.HelmReleaseResource{
			Kind:      r.Kind,
			Name:      r.Name,
			Namespace: r.Namespace,
		})
	}

	return &pb.HelmReleaseDetail{
		Name:         detail.Name,
		Namespace:    detail.Namespace,
		Chart:        detail.Chart,
		ChartVersion: detail.ChartVersion,
		AppVersion:   detail.AppVersion,
		Status:       detail.Status,
		Revision:     int32(detail.Revision),
		Updated:      detail.Updated,
		UpdatedAt:    detail.UpdatedAt,
		Notes:        detail.Notes,
		ValuesYAML:   detail.ValuesYAML,
		Resources:    resources,
		Repository:   detail.Repository,
	}, nil
}

// GetHelmChartValues implements pb.HelmServer
func (s *GRPCServer) GetHelmChartValues(ctx context.Context, req *pb.GetHelmChartValuesRequest) (*pb.GetHelmChartValuesResponse, error) {
	values, err := s.svc.GetHelmChartValues(req.Repository, req.ChartName, req.Version)
	if err != nil {
		return nil, err
	}
	return &pb.GetHelmChartValuesResponse{ValuesYAML: values}, nil
}

// GetHelmReleaseHistory implements pb.HelmServer
func (s *GRPCServer) GetHelmReleaseHistory(ctx context.Context, req *pb.GetHelmReleaseHistoryRequest) (*pb.GetHelmReleaseHistoryResponse, error) {
	history, err := s.svc.GetHelmReleaseHistory(req.Namespace, req.ReleaseName)
	if err != nil {
		return nil, err
	}
	result := make([]*pb.HelmReleaseRevisionHistory, 0, len(history))
	for _, h := range history {
		result = append(result, &pb.HelmReleaseRevisionHistory{
			Revision:     int32(h.Revision),
			ChartVersion: h.ChartVersion,
			AppVersion:   h.AppVersion,
			Status:       h.Status,
			UpdatedAt:    h.UpdatedAt,
		})
	}
	return &pb.GetHelmReleaseHistoryResponse{History: result}, nil
}

// RollbackHelmRelease implements pb.HelmServer
func (s *GRPCServer) RollbackHelmRelease(ctx context.Context, req *pb.RollbackHelmReleaseRequest) (*pb.Empty, error) {
	err := s.svc.RollbackHelmRelease(req.Namespace, req.ReleaseName, int(req.Revision))
	if err != nil {
		return nil, err
	}
	return &pb.Empty{}, nil
}

// GetCapabilities implements pb.HelmServer
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
