package kube

import "context"

func (p *ClusterProvider) GetActiveNamespaces() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.activeNamespaces
}

// SetActiveNamespaces updates the locally-synced namespace filter, pushed from the
// host over the Subscribe("namespaces.active") gRPC stream.
func (p *ClusterProvider) SetActiveNamespaces(namespaces []string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.activeNamespaces = namespaces
	return nil
}

// syncNamespacesFromHost mirrors syncContextFromHost for the active-namespaces stream.
func (p *ClusterProvider) syncNamespacesFromHost(namespaces []string) error {
	err := p.SetActiveNamespaces(namespaces)
	p.namespacesSyncOnce.Do(func() {
		close(p.namespacesSynced)
	})
	return err
}

// SyncActiveNamespaces implements port.KubeClusterProvider by delegating to syncNamespacesFromHost.
func (p *ClusterProvider) SyncActiveNamespaces(ctx context.Context, namespaces []string) error {
	return p.syncNamespacesFromHost(namespaces)
}

// ClearActiveNamespaces implements port.KubeClusterProvider (via async.EventReceiver):
// the host calls this as the first step of every cluster switch — see
// ClusterProvider.ClearActiveContext. Clearing to nil falls back to GetActiveNamespaces'
// existing "empty means cluster-wide" semantics rather than a distinct error state: an
// over-broad (unfiltered) read during the brief switch window is a much smaller risk
// than the wrong-cluster case ClearActiveContext guards against.
func (p *ClusterProvider) ClearActiveNamespaces(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.activeNamespaces = nil
	return nil
}
