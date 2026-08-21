package kube

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"google.golang.org/grpc"

	"github.com/litelensapp/litelens/packages/core/pb"
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

// fakeClusterContextStream is a scripted clusterContextStream: it yields the given
// events in order, then returns finalErr forever.
type fakeClusterContextStream struct {
	events   []*pb.ClusterContextChangedEvent
	finalErr error
	idx      int
}

func (f *fakeClusterContextStream) Recv() (*pb.ClusterContextChangedEvent, error) {
	if f.idx < len(f.events) {
		e := f.events[f.idx]
		f.idx++
		return e, nil
	}
	return nil, f.finalErr
}

// TestProcessWatchStream_StreamErrorHandling verifies that ProcessWatchStream syncs
// every event in order and returns the error that ended the stream.
func TestProcessWatchStream_StreamErrorHandling(t *testing.T) {
	streamErr := errors.New("stream closed")
	stream := &fakeClusterContextStream{
		events: []*pb.ClusterContextChangedEvent{
			{ContextName: "ctx-a", KubeconfigPath: "/a"},
			{ContextName: "ctx-b", KubeconfigPath: "/b"},
		},
		finalErr: streamErr,
	}

	var synced []string
	err := ProcessWatchStream(stream, func(contextName, kubeconfigPath string) error {
		synced = append(synced, fmt.Sprintf("%s:%s", contextName, kubeconfigPath))
		return nil
	})

	if !errors.Is(err, streamErr) {
		t.Fatalf("expected ProcessWatchStream to return the stream error, got %v", err)
	}
	want := []string{"ctx-a:/a", "ctx-b:/b"}
	if len(synced) != len(want) || synced[0] != want[0] || synced[1] != want[1] {
		t.Fatalf("expected sync calls %v, got %v", want, synced)
	}
}

// TestProcessWatchStream_ConcurrentContextChanges verifies that a sync failure on one
// event does not stop later events in the same stream from being processed.
func TestProcessWatchStream_ConcurrentContextChanges(t *testing.T) {
	streamErr := errors.New("stream closed")
	stream := &fakeClusterContextStream{
		events: []*pb.ClusterContextChangedEvent{
			{ContextName: "ctx-fail", KubeconfigPath: "/fail"},
			{ContextName: "ctx-ok", KubeconfigPath: "/ok"},
		},
		finalErr: streamErr,
	}

	var synced []string
	err := ProcessWatchStream(stream, func(contextName, kubeconfigPath string) error {
		synced = append(synced, contextName)
		if contextName == "ctx-fail" {
			return errors.New("sync failed")
		}
		return nil
	})

	if !errors.Is(err, streamErr) {
		t.Fatalf("expected ProcessWatchStream to return the stream error, got %v", err)
	}
	if len(synced) != 2 || synced[0] != "ctx-fail" || synced[1] != "ctx-ok" {
		t.Fatalf("expected both events processed despite sync error, got %v", synced)
	}
}

// TestWatchClusterContext_ReconnectLoop runs the real WatchClusterContext goroutine
// against a real gRPC server that is not yet listening at goroutine start (forcing at
// least one dial-error/backoff cycle), then starts listening and streams a real event,
// verifying the provider's active context actually gets synced end-to-end.
func TestWatchClusterContext_ReconnectLoop(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve address: %v", err)
	}
	addr := ln.Addr().String()
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("split host port: %v", err)
	}
	ln.Close()

	kubeconfigPath := writeFakeKubeconfig(t)

	provider := NewDynamicClusterProvider(context.Background())
	go WatchClusterContext(port, provider)

	// WatchClusterContext's first dial attempt hits "connection refused" since nothing
	// is listening yet; give it a moment to actually run that failing attempt before
	// the host starts, so the reconnect path (not just the happy path) is exercised.
	time.Sleep(100 * time.Millisecond)

	ln2, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatalf("re-listen on %s: %v", addr, err)
	}
	events := make(chan *pb.ClusterContextChangedEvent, 1)
	events <- &pb.ClusterContextChangedEvent{ContextName: "test-context", KubeconfigPath: kubeconfigPath}
	mock := &streamingMockPluginServer{events: events}
	srv := grpc.NewServer()
	pb.RegisterPluginServer(srv, mock)
	go srv.Serve(ln2)
	defer srv.Stop()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		_, _, activeContext, _ := provider.ActiveClients()
		if activeContext == "test-context" {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("WatchClusterContext did not sync the streamed event within the deadline")
}

// streamingMockPluginServer sends every event on the events channel to the client,
// then blocks until the stream context is cancelled — simulating a live host that
// has one pending cluster-context event to deliver on (re)subscribe.
type streamingMockPluginServer struct {
	pb.UnimplementedPluginServer
	events chan *pb.ClusterContextChangedEvent
}

func (m *streamingMockPluginServer) ClusterContextWatch(req *pb.Empty, stream pb.Plugin_ClusterContextWatchServer) error {
	for {
		select {
		case event, ok := <-m.events:
			if !ok {
				return io.EOF
			}
			if err := stream.Send(event); err != nil {
				return err
			}
		case <-stream.Context().Done():
			return stream.Context().Err()
		}
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
- name: fake
  cluster:
    server: https://127.0.0.1:6443
contexts:
- name: fake
  context:
    cluster: fake
    user: fake
current-context: fake
users:
- name: fake
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
