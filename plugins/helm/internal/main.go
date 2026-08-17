package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	helmgo "github.com/litelensapp/litelens-plugins/plugins/helm/internal/helm"
	"github.com/litelensapp/litelens-plugins/plugins/helm/internal/server"
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
	handshake := map[string]any{
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

// dynamicClusterProvider is a ClusterProvider whose active context can be changed
// after construction via SetActiveContext, allowing the app to sync the subprocess's
// live cluster client on every feature call.
type dynamicClusterProvider struct {
	mu             sync.RWMutex
	cs             *kubernetes.Clientset
	rc             *rest.Config
	activeContext  string
	kubeconfigPath string
	ctx            context.Context
}

func (p *dynamicClusterProvider) ActiveClients() (*kubernetes.Clientset, *rest.Config, string, []string) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.cs, p.rc, p.activeContext, []string{p.kubeconfigPath}
}

func (p *dynamicClusterProvider) Ctx() context.Context {
	return p.ctx
}

func (p *dynamicClusterProvider) SetActiveContext(contextName, kubeconfigPath string) error {
	p.mu.RLock()
	unchanged := contextName == p.activeContext && kubeconfigPath == p.kubeconfigPath
	p.mu.RUnlock()
	if unchanged {
		return nil
	}

	var cfg *rest.Config
	var err error
	if kubeconfigPath == "" {
		cfg, err = rest.InClusterConfig()
		if err != nil {
			return fmt.Errorf("in-cluster config: %w", err)
		}
	} else {
		cfg, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			return fmt.Errorf("kubeconfig %q: %w", kubeconfigPath, err)
		}
	}
	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return fmt.Errorf("build clientset: %w", err)
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.cs, p.rc, p.activeContext, p.kubeconfigPath = cs, cfg, contextName, kubeconfigPath
	return nil
}

// buildClusterProvider creates a ClusterProvider from a kubeconfig path.
// If kubeconfig is empty, attempts in-cluster config.
// If kubeconfig is explicitly provided but fails to load, returns an error immediately.
func buildClusterProvider(kubeconfig string) (helmgo.ClusterProvider, error) {
	dp := &dynamicClusterProvider{ctx: context.Background()}

	// Resolve the initial context name from kubeconfig
	var contextName string
	if kubeconfig == "" {
		contextName = "in-cluster"
	} else {
		// Get active context name from kubeconfig
		loader := &clientcmd.ClientConfigLoadingRules{Precedence: []string{kubeconfig}}
		config, err := loader.Load()
		if err != nil || config == nil {
			contextName = "default"
		} else {
			contextName = config.CurrentContext
		}
	}

	// Seed the initial cluster configuration
	if err := dp.SetActiveContext(contextName, kubeconfig); err != nil {
		return nil, err
	}

	return dp, nil
}
