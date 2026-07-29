package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	helmgo "github.com/gknguyen/litelens/plugins/helm"
	"github.com/gknguyen/litelens/plugins/helm/internal/server"
)

var (
	// Version is set via -ldflags at build time
	Version = "dev"
)

func main() {
	kubeconfig := flag.String("kubeconfig", "", "Path to kubeconfig file (if empty, uses in-cluster config)")
	listen := flag.String("listen", "127.0.0.1:0", "gRPC listen address (port 0 = auto-assign)")
	flag.Parse()

	// Build cluster provider from kubeconfig
	cp, err := buildClusterProvider(*kubeconfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: build cluster provider: %v\n", err)
		os.Exit(1)
	}

	// Create helm service
	svc := helmgo.NewService(cp, func(ctx context.Context, eventName string, data any) {
		// No-op event emitter for plugin mode
	})

	// Start gRPC server
	grpcSrv, ln, err := server.NewGRPCServer(svc, *listen)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: create grpc server: %v\n", err)
		os.Exit(1)
	}
	defer grpcSrv.GracefulStop()

	// Extract the actual port from listener address
	grpcAddr := ln.Addr().String()
	// Parse port from address (format: "127.0.0.1:PORT" or "[::1]:PORT")
	_, portStr, err := net.SplitHostPort(grpcAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: parse listener address: %v\n", err)
		os.Exit(1)
	}
	grpcPort, err := strconv.Atoi(portStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: convert port to int: %v\n", err)
		os.Exit(1)
	}

	// Handshake: emit exactly one JSON line to stdout on readiness
	handshake := map[string]interface{}{
		"type":      "READY",
		"version":   Version,
		"grpcPort":  grpcPort,
		"pid":       os.Getpid(),
		"timestamp": time.Now().Format(time.RFC3339),
	}
	data, _ := json.Marshal(handshake)
	fmt.Println(string(data))
	os.Stdout.Sync()

	// Serve
	if err := grpcSrv.Serve(ln); err != nil {
		fmt.Fprintf(os.Stderr, "error: serve: %v\n", err)
		os.Exit(1)
	}
}

// buildClusterProvider creates a ClusterProvider from a kubeconfig path.
// If kubeconfig is empty, attempts in-cluster config.
// If kubeconfig is explicitly provided but fails to load, returns an error immediately.
func buildClusterProvider(kubeconfig string) (helmgo.ClusterProvider, error) {
	ctx := context.Background()

	var cs *kubernetes.Clientset
	var rc *rest.Config
	var activeContext string

	if kubeconfig == "" {
		// In-cluster config
		cfg, err := rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("in-cluster config: %w", err)
		}
		client, err := kubernetes.NewForConfig(cfg)
		if err != nil {
			return nil, fmt.Errorf("create clientset: %w", err)
		}
		cs = client
		rc = cfg
		activeContext = "in-cluster"
	} else {
		// Load from kubeconfig — explicit path must succeed
		cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("kubeconfig %q: %w", kubeconfig, err)
		}
		client, err := kubernetes.NewForConfig(cfg)
		if err != nil {
			return nil, fmt.Errorf("create clientset from kubeconfig: %w", err)
		}

		// Get active context name
		loader := &clientcmd.ClientConfigLoadingRules{Precedence: []string{kubeconfig}}
		config, err := loader.Load()
		if err != nil || config == nil {
			activeContext = "default"
		} else {
			activeContext = config.CurrentContext
		}

		cs = client
		rc = cfg
	}

	return helmgo.NewClusterProviderFunc(
		func() (*kubernetes.Clientset, *rest.Config, string, []string) {
			return cs, rc, activeContext, []string{kubeconfig}
		},
		func() context.Context { return ctx },
	), nil
}
