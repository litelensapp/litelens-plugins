package kube

import (
	"context"
	"fmt"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/litelensapp/litelens-plugins/plugins/helm/internal/applications/port"
)

func (p *ClusterProvider) GetActiveContext() (*kubernetes.Clientset, *rest.Config, string, []string) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.cs, p.rc, p.activeContext, []string{p.kubeconfigPath}
}

func (p *ClusterProvider) SetActiveContext(contextName, kubeconfigPath string) error {
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
func (p *ClusterProvider) syncContextFromHost(contextName, kubeconfigPath string) error {
	err := p.SetActiveContext(contextName, kubeconfigPath)
	p.contextSyncOnce.Do(func() {
		close(p.contextSynced)
	})
	return err
}

// SyncClusterContext implements port.KubeClusterProvider by delegating to syncContextFromHost.
func (p *ClusterProvider) SyncClusterContext(ctx context.Context, contextName, kubeconfigPath string) error {
	return p.syncContextFromHost(contextName, kubeconfigPath)
}

// ClearActiveContext implements port.KubeClusterProvider (via async.EventReceiver):
// the host calls this as the first step of every cluster switch, before it has even
// resolved the new context's kubeconfig path, so a business call racing in during the
// switch finds no active context rather than one silently left over from the previous
// cluster. Deliberately leaves cs/rc/kubeconfigPath untouched — only the activeContext
// label is cleared — so nothing risks handing a nil clientset to the cluster-independent
// endpoints that don't gate on activeContext at all.
func (p *ClusterProvider) ClearActiveContext(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.activeContext = ""
	return nil
}

// BuildClusterProvider creates a KubeClusterProvider from a kubeconfig path.
// If kubeconfig is empty, attempts in-cluster config.
// If kubeconfig is explicitly provided but fails to load, returns an error immediately.
func BuildClusterProvider(kubeconfig string) (port.KubeClusterProvider, error) {
	dp := NewClusterProvider(context.Background())

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
