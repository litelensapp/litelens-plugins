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

	"github.com/litelensapp/litelens-plugins/plugins/helm/internal/applications/port"
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

	contextSynced      chan struct{}
	namespacesSynced   chan struct{}
	contextSyncOnce    sync.Once
	namespacesSyncOnce sync.Once
}

// NewDynamicClusterProvider returns a DynamicClusterProvider bound to ctx, with no
// active cluster client yet — call SetActiveContext to seed it.
func NewDynamicClusterProvider(ctx context.Context) *DynamicClusterProvider {
	return &DynamicClusterProvider{
		ctx:              ctx,
		contextSynced:    make(chan struct{}),
		namespacesSynced: make(chan struct{}),
	}
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

// syncContextFromHost applies a cluster-context update received over the host's gRPC
// stream and marks the initial sync complete, unblocking WaitForInitialSync. Distinct
// from SetActiveContext so the kubeconfig-derived guess seeded by BuildClusterProvider
// (which is not authoritative — it reflects kubectl's last-used context, not litelens's
// active cluster) never itself counts as "synced".
func (p *DynamicClusterProvider) syncContextFromHost(contextName, kubeconfigPath string) error {
	err := p.SetActiveContext(contextName, kubeconfigPath)
	p.contextSyncOnce.Do(func() { close(p.contextSynced) })
	return err
}

// syncNamespacesFromHost mirrors syncContextFromHost for the active-namespaces stream.
func (p *DynamicClusterProvider) syncNamespacesFromHost(namespaces []string) error {
	err := p.SetActiveNamespaces(namespaces)
	p.namespacesSyncOnce.Do(func() { close(p.namespacesSynced) })
	return err
}

// SyncClusterContext implements async.EventReceiver by delegating to syncContextFromHost.
func (p *DynamicClusterProvider) SyncClusterContext(ctx context.Context, contextName, kubeconfigPath string) error {
	return p.syncContextFromHost(contextName, kubeconfigPath)
}

// SyncActiveNamespaces implements async.EventReceiver by delegating to syncNamespacesFromHost.
func (p *DynamicClusterProvider) SyncActiveNamespaces(ctx context.Context, namespaces []string) error {
	return p.syncNamespacesFromHost(namespaces)
}

// WaitForInitialSync blocks until the host has pushed the first cluster-context and
// active-namespaces messages, or timeout elapses. Call this before serving business
// HTTP calls: without it, a request racing the watch streams' initial dial+subscribe
// would be served against BuildClusterProvider's kubeconfig-derived guess (the wrong
// cluster/unfiltered namespaces) — a result the frontend then caches indefinitely.
func (p *DynamicClusterProvider) WaitForInitialSync(timeout time.Duration) {
	deadline, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	select {
	case <-p.contextSynced:
	case <-deadline.Done():
		fmt.Fprintf(os.Stderr, "warning: timed out waiting for initial cluster-context sync from host\n")
		return
	}

	select {
	case <-p.namespacesSynced:
	case <-deadline.Done():
		fmt.Fprintf(os.Stderr, "warning: timed out waiting for initial active-namespaces sync from host\n")
	}
}

// BuildClusterProvider creates a ClusterProvider from a kubeconfig path.
// If kubeconfig is empty, attempts in-cluster config.
// If kubeconfig is explicitly provided but fails to load, returns an error immediately.
func BuildClusterProvider(kubeconfig string) (port.ClusterProvider, error) {
	dp := NewDynamicClusterProvider(context.Background())

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
