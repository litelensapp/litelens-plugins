package kube

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// ClusterProvider is a port.KubeClusterProvider whose active context can be changed
// after construction via SetActiveContext, allowing the app to sync the subprocess's
// live cluster client on every feature call.
type ClusterProvider struct {
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

// NewClusterProvider returns a ClusterProvider bound to ctx, with no
// active cluster client yet — call SetActiveContext to seed it.
func NewClusterProvider(ctx context.Context) *ClusterProvider {
	return &ClusterProvider{
		ctx:              ctx,
		contextSynced:    make(chan struct{}),
		namespacesSynced: make(chan struct{}),
	}
}

func (p *ClusterProvider) Ctx() context.Context {
	return p.ctx
}

// WaitForInitialSync blocks until the host has pushed the first cluster-context and
// active-namespaces messages, or timeout elapses. Call this before serving business
// HTTP calls: without it, a request racing the watch streams' initial dial+subscribe
// would be served against BuildClusterProvider's kubeconfig-derived guess (the wrong
// cluster/unfiltered namespaces) — a result the frontend then caches indefinitely.
func (p *ClusterProvider) WaitForInitialSync(timeout time.Duration) {
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
