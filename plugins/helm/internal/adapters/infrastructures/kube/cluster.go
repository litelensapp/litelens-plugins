package kube

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	grpcclient "github.com/litelensapp/litelens-plugins/plugins/helm/internal/adapters/infrastructures/app"
	"github.com/litelensapp/litelens-plugins/plugins/helm/internal/applications/dto"
	"github.com/litelensapp/litelens-plugins/plugins/helm/internal/applications/port"
	"github.com/litelensapp/litelens/packages/core/util"
)

// DynamicClusterProvider is a ClusterProvider whose active context can be changed
// after construction via SetActiveContext, allowing the app to sync the subprocess's
// live cluster client on every feature call.
type DynamicClusterProvider struct {
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

// NewDynamicClusterProvider returns a DynamicClusterProvider bound to ctx, with no
// active cluster client yet — call SetActiveContext to seed it.
func NewDynamicClusterProvider(ctx context.Context) *DynamicClusterProvider {
	return &DynamicClusterProvider{
		ctx:              ctx,
		contextSynced:    make(chan struct{}),
		namespacesSynced: make(chan struct{}),
	}
}

func (p *DynamicClusterProvider) ActiveClients() (*kubernetes.Clientset, *rest.Config, string, []string) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.cs, p.rc, p.activeContext, []string{p.kubeconfigPath}
}

func (p *DynamicClusterProvider) ActiveNamespaces() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.activeNamespaces
}

// SetActiveNamespaces updates the locally-synced namespace filter, pushed from the
// host over the Subscribe("namespaces.active") gRPC stream.
func (p *DynamicClusterProvider) SetActiveNamespaces(namespaces []string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.activeNamespaces = namespaces
	return nil
}

func (p *DynamicClusterProvider) Ctx() context.Context {
	return p.ctx
}

func (p *DynamicClusterProvider) SetActiveContext(contextName, kubeconfigPath string) error {
	p.mu.RLock()
	unchanged := contextName == p.activeContext && kubeconfigPath == p.kubeconfigPath
	p.mu.RUnlock()
	if unchanged {
		return nil
	}

	var cfg *rest.Config
	var err error
	if kubeconfigPath == "" {
		cfg, err = rest.InClusterConfig()
		if err != nil {
			return fmt.Errorf("in-cluster config: %w", err)
		}
	} else {
		loader := &clientcmd.ClientConfigLoadingRules{Precedence: []string{kubeconfigPath}}
		clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			loader,
			&clientcmd.ConfigOverrides{CurrentContext: contextName},
		)
		cfg, err = clientConfig.ClientConfig()
		if err != nil {
			return fmt.Errorf("kubeconfig %q: %w", kubeconfigPath, err)
		}
	}
	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return fmt.Errorf("build clientset: %w", err)
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.cs, p.rc, p.activeContext, p.kubeconfigPath = cs, cfg, contextName, kubeconfigPath
	return nil
}

// syncContextFromHost applies a cluster-context update received over the host's gRPC
// stream and marks the initial sync complete, unblocking WaitForInitialSync. Distinct
// from SetActiveContext so the kubeconfig-derived guess seeded by BuildClusterProvider
// (which is not authoritative — it reflects kubectl's last-used context, not litelens's
// active cluster) never itself counts as "synced".
func (p *DynamicClusterProvider) syncContextFromHost(contextName, kubeconfigPath string) error {
	err := p.SetActiveContext(contextName, kubeconfigPath)
	p.contextSyncOnce.Do(func() { close(p.contextSynced) })
	return err
}

// syncNamespacesFromHost mirrors syncContextFromHost for the active-namespaces stream.
func (p *DynamicClusterProvider) syncNamespacesFromHost(namespaces []string) error {
	err := p.SetActiveNamespaces(namespaces)
	p.namespacesSyncOnce.Do(func() { close(p.namespacesSynced) })
	return err
}

// WaitForInitialSync blocks until the host has pushed the first cluster-context and
// active-namespaces messages, or timeout elapses. Call this before serving business
// HTTP calls: without it, a request racing the watch streams' initial dial+subscribe
// would be served against BuildClusterProvider's kubeconfig-derived guess (the wrong
// cluster/unfiltered namespaces) — a result the frontend then caches indefinitely.
func (p *DynamicClusterProvider) WaitForInitialSync(timeout time.Duration) {
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

// ProcessWatchStream reads events from stream until Recv() errors (connection lost,
// clean server-side stream close, etc.), invoking syncFn for each event. A syncFn
// error is logged but does not stop processing — the stream is only abandoned when
// Recv() itself errors. Always returns a non-nil error (the one that ended the stream).
func (p *DynamicClusterProvider) ProcessWatchStream(client *grpcclient.GrpcClient, stream dto.SubscribeStream, syncFn func(contextName, kubeconfigPath string) error) error {
	for {
		event, err := client.RecvClusterContext(stream)
		if err != nil {
			return err
		}
		if syncErr := syncFn(event.ContextName, event.KubeconfigPath); syncErr != nil {
			fmt.Fprintf(os.Stderr, "error: sync cluster context (%s): %v\n", event.ContextName, syncErr)
		}
	}
}

// WatchClusterContext connects to the host's gRPC server and subscribes to cluster context changes.
// Uses exponential backoff for reconnection with a max interval of 30s. Dials its own
// dedicated GrpcClient rather than sharing one with WatchActiveNamespaces: GrpcClient.Dial
// closes and replaces the instance's connection, so two watch loops sharing one instance
// would tear down each other's live stream every time either one (re)connects.
func (p *DynamicClusterProvider) WatchClusterContext(hostPort, authToken string) {
	addr := fmt.Sprintf("127.0.0.1:%s", hostPort)
	br := NewBackoffReconnector()
	client := &grpcclient.GrpcClient{}

	for {
		stream, err := client.DialAndSubscribe(addr, string(util.EventTopicClusterContext), authToken)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			time.Sleep(br.OnDialError())
			continue
		}

		br.OnConnected()

		streamErr := p.ProcessWatchStream(client, stream, p.syncContextFromHost)
		fmt.Fprintf(os.Stderr, "error: cluster context watch stream: %v\n", streamErr)
		client.Close()

		time.Sleep(br.OnStreamError())
	}
}

// ProcessActiveNamespacesWatchStream reads events from stream until Recv() errors,
// invoking syncFn for each event. Mirrors ProcessWatchStream's error handling: a
// syncFn error is logged but does not stop processing.
func (p *DynamicClusterProvider) ProcessActiveNamespacesWatchStream(client *grpcclient.GrpcClient, stream dto.SubscribeStream, syncFn func(namespaces []string) error) error {
	for {
		event, err := client.RecvActiveNamespaces(stream)
		if err != nil {
			return err
		}
		if syncErr := syncFn(event.Namespaces); syncErr != nil {
			fmt.Fprintf(os.Stderr, "error: sync active namespaces: %v\n", syncErr)
		}
	}
}

// WatchActiveNamespaces connects to the host's gRPC server and subscribes to
// active-namespaces changes. Uses exponential backoff for reconnection with a max
// interval of 30s, mirroring WatchClusterContext. Dials its own dedicated GrpcClient
// for the same reason WatchClusterContext does — see its comment.
func (p *DynamicClusterProvider) WatchActiveNamespaces(hostPort, authToken string) {
	addr := fmt.Sprintf("127.0.0.1:%s", hostPort)
	br := NewBackoffReconnector()
	client := &grpcclient.GrpcClient{}

	for {
		stream, err := client.DialAndSubscribe(addr, string(util.EventTopicNamespacesActive), authToken)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			time.Sleep(br.OnDialError())
			continue
		}

		br.OnConnected()

		streamErr := p.ProcessActiveNamespacesWatchStream(client, stream, p.syncNamespacesFromHost)
		fmt.Fprintf(os.Stderr, "error: active namespaces watch stream: %v\n", streamErr)
		client.Close()

		time.Sleep(br.OnStreamError())
	}
}

// BuildClusterProvider creates a ClusterProvider from a kubeconfig path.
// If kubeconfig is empty, attempts in-cluster config.
// If kubeconfig is explicitly provided but fails to load, returns an error immediately.
func BuildClusterProvider(kubeconfig string) (port.ClusterProvider, error) {
	dp := NewDynamicClusterProvider(context.Background())

	// Resolve the initial context name from kubeconfig
	var contextName string
	if kubeconfig == "" {
		contextName = "in-cluster"
	} else {
		// Get active context name from kubeconfig
		loader := &clientcmd.ClientConfigLoadingRules{Precedence: []string{kubeconfig}}
		config, err := loader.Load()
		if err != nil || config == nil {
			contextName = "default"
		} else {
			contextName = config.CurrentContext
		}
	}

	// Seed the initial cluster configuration
	if err := dp.SetActiveContext(contextName, kubeconfig); err != nil {
		return nil, err
	}

	return dp, nil
}
