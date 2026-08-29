package kube

import (
	"context"
	"testing"
)

// TestClusterProvider_SyncActiveNamespaces verifies that SyncActiveNamespaces
// (the syncer port) correctly updates the active namespaces.
func TestClusterProvider_SyncActiveNamespaces(t *testing.T) {
	provider := NewClusterProvider(context.Background())

	namespaces := []string{"default", "kube-system"}
	err := provider.SyncActiveNamespaces(context.Background(), namespaces)
	if err != nil {
		t.Fatalf("SyncActiveNamespaces failed: %v", err)
	}

	// Verify the active namespaces were set
	activeNamespaces := provider.GetActiveNamespaces()
	if len(activeNamespaces) != len(namespaces) || activeNamespaces[0] != "default" || activeNamespaces[1] != "kube-system" {
		t.Fatalf("expected namespaces %v, got %v", namespaces, activeNamespaces)
	}
}
