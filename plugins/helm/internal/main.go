package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"

	grpc "github.com/litelensapp/litelens-plugins/plugins/helm/internal/adapters/infrastructures/app"
	"github.com/litelensapp/litelens-plugins/plugins/helm/internal/adapters/infrastructures/kube"
	"github.com/litelensapp/litelens-plugins/plugins/helm/internal/adapters/infrastructures/restconfig"
	"github.com/litelensapp/litelens-plugins/plugins/helm/internal/adapters/presentations/rest"
	"github.com/litelensapp/litelens-plugins/plugins/helm/internal/applications/helm"
	"github.com/litelensapp/litelens-plugins/plugins/helm/internal/applications/lock"
	"github.com/litelensapp/litelens-plugins/plugins/helm/internal/config"
	"github.com/litelensapp/litelens/packages/core/pluginsdk"
)

var (
	// Version is set via -ldflags at build time
	Version = "dev"
	// authToken is the authorization token read from stdin, used by the gRPC client interceptors
	authToken string
)

func main() {
	kubeconfig := flag.String("kubeconfig", "", "Path to kubeconfig file (if empty, uses in-cluster config)")
	listen := flag.String("listen", "127.0.0.1:0", "HTTP listen address (port 0 = auto-assign)")
	flag.Parse()

	// Read authorization token from stdin before any gRPC operations
	token, err := pluginsdk.ReadAuthTokenFromStdin()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: read auth token from stdin: %v\n", err)
		os.Exit(1)
	}
	authToken = token

	// Set up the single gRPC client shared by event emission and by both
	// cluster-context/active-namespaces watch loops (if the host is reachable)
	hostPort := config.GetHostGRPCPort()
	var hostClient *grpc.GrpcClient
	if hostPort != "" {
		addr := fmt.Sprintf("127.0.0.1:%s", hostPort)
		client := &grpc.GrpcClient{}
		if err := client.Dial(addr, authToken); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to connect to host gRPC server: %v\n", err)
		} else {
			defer client.Close()
			hostClient = client
		}
	}

	// Build cluster provider from kubeconfig
	clusterProvider, err := kube.BuildClusterProvider(*kubeconfig, hostClient)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: build cluster provider: %v\n", err)
		os.Exit(1)
	}

	// Create helm service, wrapped so business calls and cluster-context switches
	// (subscribed via gRPC Subscribe("cluster.context") stream) never race on the underlying k8s client.
	eventEmitFn := func(ctx context.Context, eventName string, data any) {
		if hostClient != nil {
			hostClient.Emit(ctx, eventName, "helm", data)
		}
	}
	helmService := helm.NewService(clusterProvider, eventEmitFn, restconfig.Factory{})
	lockService := lock.NewService(helmService)

	// Subscribe to cluster context and active-namespaces changes from the host
	if hostClient != nil {
		dp, ok := clusterProvider.(*kube.DynamicClusterProvider)
		if !ok {
			fmt.Fprintf(os.Stderr, "error: cluster provider is not *kube.DynamicClusterProvider\n")
			os.Exit(1)
		}
		go dp.WatchClusterContext(hostPort, authToken)
		go dp.WatchActiveNamespaces(hostPort, authToken)
	}

	// Serve HTTP server (blocks for the process lifetime)
	httpServer, err := rest.NewHttpServer(*listen, Version, lockService)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer httpServer.Close()
	if err := httpServer.Serve(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		fmt.Fprintf(os.Stderr, "error: http serve: %v\n", err)
		os.Exit(1)
	}
}
