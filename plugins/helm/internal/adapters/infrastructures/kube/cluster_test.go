package kube

import (
	"context"
	"os"
	"sync"
	"sync/atomic"
	"testing"
)

// TestClusterProvider_Idempotency documents that setting the same context twice is safe.
func TestClusterProvider_Idempotency(t *testing.T) {
	// The ClusterProvider.SetActiveContext method is idempotent:
	// - If the context name and kubeconfig path are the same as the previous call,
	//   the method returns early without reconstructing the Kubernetes client
	// - This is safe even if called concurrently with GetActiveContext() reads
	// - The early-return check is guarded by mu.RLock()

	provider := NewClusterProvider(context.Background())

	// Both calls should succeed (or both fail on kubeconfig errors, which is fine)
	err1 := provider.SetActiveContext("test-context", "")
	err2 := provider.SetActiveContext("test-context", "")

	// If one fails and the other succeeds, that indicates improper idempotency
	// Both should behave consistently
	if (err1 == nil) != (err2 == nil) {
		t.Fatalf("idempotency violation: first call err=%v, second call err=%v", err1, err2)
	}
}

// TestClusterProvider_ConcurrentContextAccess tests concurrent SetActiveContext
// and GetActiveContext access.
func TestClusterProvider_ConcurrentContextAccess(t *testing.T) {
	provider := NewClusterProvider(context.Background())

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

	// Simulate concurrent GetActiveContext calls (from business logic)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, _, _ = provider.GetActiveContext()
		}()
	}

	wg.Wait()

	// All context changes should have completed
	if contextChangeCount.Load() != 10 {
		t.Fatalf("expected 10 context changes, got %d", contextChangeCount.Load())
	}
}

// TestClusterProvider_SyncClusterContext verifies that SyncClusterContext
// (the syncer port) correctly updates the active context.
func TestClusterProvider_SyncClusterContext(t *testing.T) {
	provider := NewClusterProvider(context.Background())
	kubeconfigPath := writeFakeKubeconfig(t)

	// Call SyncClusterContext
	err := provider.SyncClusterContext(context.Background(), "test-context", kubeconfigPath)
	if err != nil {
		t.Fatalf("SyncClusterContext failed: %v", err)
	}

	// Verify the active context was set
	_, _, activeContext, _ := provider.GetActiveContext()
	if activeContext != "test-context" {
		t.Fatalf("expected active context 'test-context', got %q", activeContext)
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

	// Construct a ClusterProvider
	provider := NewClusterProvider(context.Background())

	// Call SetActiveContext with context-b (while kubeconfig's current-context is context-a)
	err = provider.SetActiveContext("context-b", kubeconfigPath)
	if err != nil {
		t.Fatalf("SetActiveContext failed: %v", err)
	}

	// Verify the resulting rest.Config's Host is context-b's server URL, not context-a's
	_, cfg, _, _ := provider.GetActiveContext()
	if cfg == nil {
		t.Fatal("GetActiveContext returned nil rest.Config")
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
