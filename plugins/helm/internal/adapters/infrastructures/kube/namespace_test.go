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

// TestClearActiveNamespaces_FallsBackToClusterWide is a regression test for the
// clear-first cluster-switch design: clearing must reset to nil, matching
// GetActiveNamespaces' existing "empty means cluster-wide/unfiltered" semantics
// rather than introducing a new distinct error state.
func TestClearActiveNamespaces_FallsBackToClusterWide(t *testing.T) {
	provider := NewClusterProvider(context.Background())

	if err := provider.SyncActiveNamespaces(context.Background(), []string{"default"}); err != nil {
		t.Fatalf("SyncActiveNamespaces failed: %v", err)
	}

	if err := provider.ClearActiveNamespaces(context.Background()); err != nil {
		t.Fatalf("ClearActiveNamespaces failed: %v", err)
	}

	if got := provider.GetActiveNamespaces(); got != nil {
		t.Fatalf("expected ClearActiveNamespaces to reset to nil (cluster-wide), got %v", got)
	}
}
