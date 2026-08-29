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
