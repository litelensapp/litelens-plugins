package kube

import (
	"context"
	"os"
	"sync"
	"sync/atomic"
	"testing"
)

// TestDynamicClusterProvider_Idempotency documents that setting the same context twice is safe.
func TestDynamicClusterProvider_Idempotency(t *testing.T) {
	// The DynamicClusterProvider.SetActiveContext method is idempotent:
	// - If the context name and kubeconfig path are the same as the previous call,
	//   the method returns early without reconstructing the Kubernetes client
	// - This is safe even if called concurrently with ActiveClients() reads
	// - The early-return check is guarded by mu.RLock()

	provider := NewDynamicClusterProvider(context.Background())

	// Both calls should succeed (or both fail on kubeconfig errors, which is fine)
	err1 := provider.SetActiveContext("test-context", "")
	err2 := provider.SetActiveContext("test-context", "")

	// If one fails and the other succeeds, that indicates improper idempotency
	// Both should behave consistently
	if (err1 == nil) != (err2 == nil) {
		t.Fatalf("idempotency violation: first call err=%v, second call err=%v", err1, err2)
	}
}

// TestDynamicClusterProvider_ConcurrentContextAccess tests concurrent SetActiveContext
// and ActiveClients access.
func TestDynamicClusterProvider_ConcurrentContextAccess(t *testing.T) {
	provider := NewDynamicClusterProvider(context.Background())

	var wg sync.WaitGroup
	var contextChangeCount atomic.Int32

	// Simulate concurrent SetActiveContext calls (from watchClusterContext)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			contextName := string(rune(idx))
			_ = provider.SetActiveContext(contextName, "")
			contextChangeCount.Add(1)
		}(i)
	}

	// Simulate concurrent ActiveClients calls (from business logic)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, _, _ = provider.ActiveClients()
		}()
	}

	wg.Wait()

	// All context changes should have completed
	if contextChangeCount.Load() != 10 {
		t.Fatalf("expected 10 context changes, got %d", contextChangeCount.Load())
	}
}

// TestDynamicClusterProvider_SyncClusterContext verifies that SyncClusterContext
// (the syncer port) correctly updates the active context.
func TestDynamicClusterProvider_SyncClusterContext(t *testing.T) {
	provider := NewDynamicClusterProvider(context.Background())
	kubeconfigPath := writeFakeKubeconfig(t)

	// Call SyncClusterContext
	err := provider.SyncClusterContext(context.Background(), "test-context", kubeconfigPath)
	if err != nil {
		t.Fatalf("SyncClusterContext failed: %v", err)
	}

	// Verify the active context was set
	_, _, activeContext, _ := provider.ActiveClients()
	if activeContext != "test-context" {
		t.Fatalf("expected active context 'test-context', got %q", activeContext)
	}
}

// TestDynamicClusterProvider_SyncActiveNamespaces verifies that SyncActiveNamespaces
// (the syncer port) correctly updates the active namespaces.
func TestDynamicClusterProvider_SyncActiveNamespaces(t *testing.T) {
	provider := NewDynamicClusterProvider(context.Background())

	namespaces := []string{"default", "kube-system"}
	err := provider.SyncActiveNamespaces(context.Background(), namespaces)
	if err != nil {
		t.Fatalf("SyncActiveNamespaces failed: %v", err)
	}

	// Verify the active namespaces were set
	activeNamespaces := provider.ActiveNamespaces()
	if len(activeNamespaces) != len(namespaces) || activeNamespaces[0] != "default" || activeNamespaces[1] != "kube-system" {
		t.Fatalf("expected namespaces %v, got %v", namespaces, activeNamespaces)
	}
}

// TestSetActiveContext_LoadsCorrectContext verifies that SetActiveContext loads the
// rest.Config for the specified context name, not the kubeconfig's current-context.
// This is a regression test for the bug where clientcmd.BuildConfigFromFlags was used,
// which ignores the contextName parameter entirely.
func TestSetActiveContext_LoadsCorrectContext(t *testing.T) {
	// Create a kubeconfig with two contexts with distinct, identifiable server URLs
	f, err := os.CreateTemp(t.TempDir(), "kubeconfig-*.yaml")
	if err != nil {
		t.Fatalf("create temp kubeconfig: %v", err)
	}
	defer f.Close()

	kubeconfig := `apiVersion: v1
kind: Config
clusters:
- name: context-a-cluster
  cluster:
    server: https://context-a.example:6443
- name: context-b-cluster
  cluster:
    server: https://context-b.example:6443
contexts:
- name: context-a
  context:
    cluster: context-a-cluster
    user: default
- name: context-b
  context:
    cluster: context-b-cluster
    user: default
current-context: context-a
users:
- name: default
  user:
    token: fake-token
`
	if _, err := f.WriteString(kubeconfig); err != nil {
		t.Fatalf("write temp kubeconfig: %v", err)
	}
	kubeconfigPath := f.Name()

	// Construct a DynamicClusterProvider
	provider := NewDynamicClusterProvider(context.Background())

	// Call SetActiveContext with context-b (while kubeconfig's current-context is context-a)
	err = provider.SetActiveContext("context-b", kubeconfigPath)
	if err != nil {
		t.Fatalf("SetActiveContext failed: %v", err)
	}

	// Verify the resulting rest.Config's Host is context-b's server URL, not context-a's
	_, cfg, _, _ := provider.ActiveClients()
	if cfg == nil {
		t.Fatal("ActiveClients returned nil rest.Config")
	}

	expectedHost := "https://context-b.example:6443"
	if cfg.Host != expectedHost {
		t.Fatalf("expected Host=%q, got %q (bug: likely using kubeconfig's current-context instead of the requested contextName)", expectedHost, cfg.Host)
	}
}

// writeFakeKubeconfig writes a syntactically valid kubeconfig to a temp file and
// returns its path. clientcmd.BuildConfigFromFlags only parses the file and does not
// dial the cluster, so a fake server URL is sufficient for SetActiveContext to succeed.
func writeFakeKubeconfig(t *testing.T) string {
	t.Helper()
	const kubeconfig = `apiVersion: v1
kind: Config
clusters:
- name: test-context-cluster
  cluster:
    server: https://127.0.0.1:6443
contexts:
- name: test-context
  context:
    cluster: test-context-cluster
    user: test-context
current-context: test-context
users:
- name: test-context
  user:
    token: fake-token
`
	f, err := os.CreateTemp(t.TempDir(), "kubeconfig-*.yaml")
	if err != nil {
		t.Fatalf("create temp kubeconfig: %v", err)
	}
	defer f.Close()
	if _, err := f.WriteString(kubeconfig); err != nil {
		t.Fatalf("write temp kubeconfig: %v", err)
	}
	return f.Name()
}
