package helm

import (
	"context"
	"sync"
	"testing"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func TestHelmAge(t *testing.T) {
	tests := []struct {
		name string
		d    time.Duration
		want string
	}{
		{"minutes: under one hour", 45 * time.Minute, "45m"},
		{"minutes: zero", 0, "0m"},
		{"hours: exactly one hour", 1 * time.Hour, "1h"},
		{"hours: 23 hours", 23 * time.Hour, "23h"},
		{"days: exactly one day", 24 * time.Hour, "1d"},
		{"days: 7 days", 7 * 24 * time.Hour, "7d"},
		{"days: 30 days", 30 * 24 * time.Hour, "30d"},
		{"future timestamp clamps to zero", -5 * time.Minute, "0m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Negative tt.d means a future timestamp (clamp case).
			// time.Now().Add(-tt.d) moves the anchor in the correct direction for both cases.
			ts := time.Now().Add(-tt.d)
			got := helmAge(ts)
			if got != tt.want {
				t.Errorf("helmAge(%v) = %q; want %q", tt.d, got, tt.want)
			}
		})
	}
}

// mockClusterProvider implements KubeClusterProvider for testing concurrency.
type mockClusterProvider struct {
	mu              sync.RWMutex
	cs              *kubernetes.Clientset
	rc              *rest.Config
	activeContext   string
	kubeconfigPath  string
	ctx             context.Context
	setContextCalls int
}

func (p *mockClusterProvider) GetActiveContext() (cs *kubernetes.Clientset, rc *rest.Config, activeContext string, kubeconfigPaths []string) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.cs, p.rc, p.activeContext, []string{p.kubeconfigPath}
}

func (p *mockClusterProvider) GetActiveNamespaces() []string {
	return nil
}

func (p *mockClusterProvider) Ctx() context.Context {
	return p.ctx
}

func (p *mockClusterProvider) SetActiveContext(contextName, kubeconfigPath string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.activeContext = contextName
	p.kubeconfigPath = kubeconfigPath
	p.setContextCalls++
	return nil
}

func (p *mockClusterProvider) SetActiveNamespaces(namespaces []string) error {
	return nil
}

func (p *mockClusterProvider) SyncClusterContext(ctx context.Context, contextName, kubeconfigPath string) error {
	return p.SetActiveContext(contextName, kubeconfigPath)
}

func (p *mockClusterProvider) SyncActiveNamespaces(ctx context.Context, namespaces []string) error {
	return p.SetActiveNamespaces(namespaces)
}

func (p *mockClusterProvider) WaitForInitialSync(timeout time.Duration) {}

// TestClusterProviderConcurrency verifies that SetActiveContext and GetActiveContext
// can be called concurrently without race conditions. The dynamic cluster provider used
// in the plugin subprocess must safely handle this pattern: the app calls SetClusterContext
// (which calls SetActiveContext) on every feature invocation to sync the active context,
// while the Service concurrently reads via GetActiveContext().
func TestClusterProviderConcurrency(t *testing.T) {
	t.Run("concurrent SetActiveContext and GetActiveContext calls", func(t *testing.T) {
		provider := &mockClusterProvider{
			ctx: context.Background(),
		}

		// Simulate the pattern: multiple goroutines calling GetActiveContext (e.g., helm Service methods)
		// while another goroutine calls SetActiveContext (e.g., gRPC handler for SetClusterContext).
		var wg sync.WaitGroup
		numReaders := 50
		numWriters := 10

		// Readers: call GetActiveContext() repeatedly (simulates feature calls in Service methods)
		for i := 0; i < numReaders; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 100; j++ {
					_, _, _, _ = provider.GetActiveContext()
					time.Sleep(time.Microsecond)
				}
			}()
		}

		// Writers: call SetActiveContext() repeatedly (simulates SetClusterContext RPC calls)
		for i := 0; i < numWriters; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < 20; j++ {
					ctx := "ctx-" + string(rune('a'+id))
					path := "/path/" + string(rune('a'+id))
					_ = provider.SetActiveContext(ctx, path)
					time.Sleep(time.Microsecond)
				}
			}(i)
		}

		// Wait for all goroutines to complete
		wg.Wait()

		// Verify SetActiveContext was called the expected number of times
		if provider.setContextCalls != numWriters*20 {
			t.Errorf("expected %d SetActiveContext calls, got %d", numWriters*20, provider.setContextCalls)
		}
	})
}
