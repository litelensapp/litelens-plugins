package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/litelensapp/litelens-plugins/plugins/helm/internal/api/grpc"
	"github.com/litelensapp/litelens-plugins/plugins/helm/internal/api/rest"
	"github.com/litelensapp/litelens-plugins/plugins/helm/internal/config"
	"github.com/litelensapp/litelens-plugins/plugins/helm/internal/helm"
	"github.com/litelensapp/litelens-plugins/plugins/helm/internal/kube"
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

	// Build cluster provider from kubeconfig
	clusterProvider, err := kube.BuildClusterProvider(*kubeconfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: build cluster provider: %v\n", err)
		os.Exit(1)
	}

	// Set up event emission to the host (if connected)
	hostPort := config.GetHostGRPCPort()
	var eventEmitter *grpc.GrpcClient
	if hostPort != "" {
		addr := fmt.Sprintf("127.0.0.1:%s", hostPort)
		client := &grpc.GrpcClient{}
		if err := client.Dial(addr, authToken); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to connect to host gRPC server for event emission: %v\n", err)
		} else {
			defer client.Close()
			eventEmitter = client
		}
	}

	// Create helm service, wrapped so business calls and cluster-context switches
	// (subscribed via gRPC Subscribe("cluster.context") stream) never race on the underlying k8s client.
	eventEmitFn := func(ctx context.Context, eventName string, data any) {
		if eventEmitter != nil {
			eventEmitter.Emit(ctx, eventName, "helm", data)
		}
	}
	helmService := helm.NewLockedService(helm.NewService(clusterProvider, eventEmitFn))

	// Subscribe to cluster context and active-namespaces changes from the host
	if hostPort != "" {
		go kube.WatchClusterContext(hostPort, authToken, clusterProvider)
		go kube.WatchActiveNamespaces(hostPort, authToken, clusterProvider)
	}

	// Serve HTTP server (blocks for the process lifetime)
	httpServer, err := rest.NewHttpServer(*listen, Version, helmService)
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
