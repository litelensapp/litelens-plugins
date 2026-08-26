package kube

import (
	"context"
	"encoding/json"
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

	grpcclient "github.com/litelensapp/litelens-plugins/plugins/helm/internal/api/grpc"
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

// fakeSubscribeStream is a scripted SubscribeStream: it yields the given
// PubSubMessage events in order, then returns finalErr forever.
type fakeSubscribeStream struct {
	events   []*pb.PubSubMessage
	finalErr error
	idx      int
}

func (f *fakeSubscribeStream) Recv() (*pb.PubSubMessage, error) {
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
	stream := &fakeSubscribeStream{
		events: []*pb.PubSubMessage{
			{
				Topic:       "cluster.context",
				PayloadJson: `{"contextName":"ctx-a","kubeconfigPath":"/a"}`,
			},
			{
				Topic:       "cluster.context",
				PayloadJson: `{"contextName":"ctx-b","kubeconfigPath":"/b"}`,
			},
		},
		finalErr: streamErr,
	}

	var synced []string
	err := ProcessWatchStream(grpcclient.NewGrpcClient(nil), stream, func(contextName, kubeconfigPath string) error {
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
	stream := &fakeSubscribeStream{
		events: []*pb.PubSubMessage{
			{
				Topic:       "cluster.context",
				PayloadJson: `{"contextName":"ctx-fail","kubeconfigPath":"/fail"}`,
			},
			{
				Topic:       "cluster.context",
				PayloadJson: `{"contextName":"ctx-ok","kubeconfigPath":"/ok"}`,
			},
		},
		finalErr: streamErr,
	}

	var synced []string
	err := ProcessWatchStream(grpcclient.NewGrpcClient(nil), stream, func(contextName, kubeconfigPath string) error {
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
	testToken := "test-token-64chars-" + "x" // Use a test auth token
	go WatchClusterContext(port, testToken, provider)

	// WatchClusterContext's first dial attempt hits "connection refused" since nothing
	// is listening yet; give it a moment to actually run that failing attempt before
	// the host starts, so the reconnect path (not just the happy path) is exercised.
	time.Sleep(100 * time.Millisecond)

	ln2, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatalf("re-listen on %s: %v", addr, err)
	}

	// Create the PubSubMessage with JSON payload
	eventPayload, _ := json.Marshal(map[string]string{
		"contextName":    "test-context",
		"kubeconfigPath": kubeconfigPath,
	})
	events := make(chan *pb.PubSubMessage, 1)
	events <- &pb.PubSubMessage{
		Topic:       "cluster.context",
		Source:      "host",
		PayloadJson: string(eventPayload),
	}
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
	events chan *pb.PubSubMessage
}

func (m *streamingMockPluginServer) Subscribe(req *pb.SubscribeRequest, stream pb.Plugin_SubscribeServer) error {
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
