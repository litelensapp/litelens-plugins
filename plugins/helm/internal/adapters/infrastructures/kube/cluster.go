package kube

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	grpcclient "github.com/litelensapp/litelens-plugins/plugins/helm/internal/adapters/infrastructures/app"
	"github.com/litelensapp/litelens-plugins/plugins/helm/internal/applications/dto"
	"github.com/litelensapp/litelens-plugins/plugins/helm/internal/applications/port"
	"github.com/litelensapp/litelens/packages/core/util"
)

// DynamicClusterProvider is a ClusterProvider whose active context can be changed
// after construction via SetActiveContext, allowing the app to sync the subprocess's
// live cluster client on every feature call.
type DynamicClusterProvider struct {
	mu               sync.RWMutex
	cs               *kubernetes.Clientset
	rc               *rest.Config
	activeContext    string
	kubeconfigPath   string
	activeNamespaces []string
	ctx              context.Context
	client           *grpcclient.GrpcClient
}

// NewDynamicClusterProvider returns a DynamicClusterProvider bound to ctx, with no
// active cluster client yet — call SetActiveContext to seed it. client is the gRPC
// client used by WatchClusterContext/WatchActiveNamespaces to dial the host.
func NewDynamicClusterProvider(ctx context.Context, client *grpcclient.GrpcClient) *DynamicClusterProvider {
	return &DynamicClusterProvider{ctx: ctx, client: client}
}

func (p *DynamicClusterProvider) ActiveClients() (*kubernetes.Clientset, *rest.Config, string, []string) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.cs, p.rc, p.activeContext, []string{p.kubeconfigPath}
}

func (p *DynamicClusterProvider) ActiveNamespaces() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.activeNamespaces
}

// SetActiveNamespaces updates the locally-synced namespace filter, pushed from the
// host over the Subscribe("namespaces.active") gRPC stream.
func (p *DynamicClusterProvider) SetActiveNamespaces(namespaces []string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.activeNamespaces = namespaces
	return nil
}

func (p *DynamicClusterProvider) Ctx() context.Context {
	return p.ctx
}

func (p *DynamicClusterProvider) SetActiveContext(contextName, kubeconfigPath string) error {
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
		loader := &clientcmd.ClientConfigLoadingRules{Precedence: []string{kubeconfigPath}}
		clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			loader,
			&clientcmd.ConfigOverrides{CurrentContext: contextName},
		)
		cfg, err = clientConfig.ClientConfig()
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

// ProcessWatchStream reads events from stream until Recv() errors (connection lost,
// clean server-side stream close, etc.), invoking syncFn for each event. A syncFn
// error is logged but does not stop processing — the stream is only abandoned when
// Recv() itself errors. Always returns a non-nil error (the one that ended the stream).
func (p *DynamicClusterProvider) ProcessWatchStream(client *grpcclient.GrpcClient, stream dto.SubscribeStream, syncFn func(contextName, kubeconfigPath string) error) error {
	for {
		event, err := client.RecvClusterContext(stream)
		if err != nil {
			return err
		}
		if syncErr := syncFn(event.ContextName, event.KubeconfigPath); syncErr != nil {
			fmt.Fprintf(os.Stderr, "error: sync cluster context (%s): %v\n", event.ContextName, syncErr)
		}
	}
}

// WatchClusterContext connects to the host's gRPC server and subscribes to cluster context changes.
// Uses exponential backoff for reconnection with a max interval of 30s.
func (p *DynamicClusterProvider) WatchClusterContext(hostPort, authToken string) {
	addr := fmt.Sprintf("127.0.0.1:%s", hostPort)
	br := NewBackoffReconnector()

	for {
		stream, err := p.client.DialAndSubscribe(addr, string(util.EventTopicClusterContext), authToken)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			time.Sleep(br.OnDialError())
			continue
		}

		br.OnConnected()

		streamErr := p.ProcessWatchStream(p.client, stream, p.SetActiveContext)
		fmt.Fprintf(os.Stderr, "error: cluster context watch stream: %v\n", streamErr)
		p.client.Close()

		time.Sleep(br.OnStreamError())
	}
}

// ProcessActiveNamespacesWatchStream reads events from stream until Recv() errors,
// invoking syncFn for each event. Mirrors ProcessWatchStream's error handling: a
// syncFn error is logged but does not stop processing.
func (p *DynamicClusterProvider) ProcessActiveNamespacesWatchStream(client *grpcclient.GrpcClient, stream dto.SubscribeStream, syncFn func(namespaces []string) error) error {
	for {
		event, err := client.RecvActiveNamespaces(stream)
		if err != nil {
			return err
		}
		if syncErr := syncFn(event.Namespaces); syncErr != nil {
			fmt.Fprintf(os.Stderr, "error: sync active namespaces: %v\n", syncErr)
		}
	}
}

// WatchActiveNamespaces connects to the host's gRPC server and subscribes to
// active-namespaces changes. Uses exponential backoff for reconnection with a max
// interval of 30s, mirroring WatchClusterContext.
func (p *DynamicClusterProvider) WatchActiveNamespaces(hostPort, authToken string) {
	addr := fmt.Sprintf("127.0.0.1:%s", hostPort)
	br := NewBackoffReconnector()

	for {
		stream, err := p.client.DialAndSubscribe(addr, string(util.EventTopicNamespacesActive), authToken)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			time.Sleep(br.OnDialError())
			continue
		}

		br.OnConnected()

		streamErr := p.ProcessActiveNamespacesWatchStream(p.client, stream, p.SetActiveNamespaces)
		fmt.Fprintf(os.Stderr, "error: active namespaces watch stream: %v\n", streamErr)
		p.client.Close()

		time.Sleep(br.OnStreamError())
	}
}

// BuildClusterProvider creates a ClusterProvider from a kubeconfig path.
// If kubeconfig is empty, attempts in-cluster config.
// If kubeconfig is explicitly provided but fails to load, returns an error immediately.
// client is the gRPC client the returned provider uses for WatchClusterContext and
// WatchActiveNamespaces.
func BuildClusterProvider(kubeconfig string, client *grpcclient.GrpcClient) (port.ClusterProvider, error) {
	dp := NewDynamicClusterProvider(context.Background(), client)

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
