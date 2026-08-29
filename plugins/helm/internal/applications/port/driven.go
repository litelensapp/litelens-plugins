package port

import (
	"context"
	"time"

	"github.com/litelensapp/litelens/packages/core/async"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// EventEmitter is a callback function to emit events from the helm service.
type EventEmitter func(ctx context.Context, eventName string, data any)

// KubeClusterProvider provides access to active cluster clients and configuration,
// and allows the active context to be changed after construction — implemented
// by the plugin subprocess's dynamic provider so the app can sync it to whatever
// cluster context is currently active, per call.
type KubeClusterProvider interface {
	async.EventReceiver

	Ctx() context.Context
	// WaitForInitialSync blocks until the host has pushed the first cluster-context and
	// active-namespaces messages, or timeout elapses.
	WaitForInitialSync(timeout time.Duration)

	GetActiveContext() (cs *kubernetes.Clientset, rc *rest.Config, activeContext string, kubeconfigPaths []string)
	SetActiveContext(contextName, kubeconfigPath string) error

	// GetActiveNamespaces returns the host's current namespace filter, synced from the
	// host app over the Subscribe("namespaces.active") gRPC stream. An empty slice means
	// cluster-wide (no filter).
	GetActiveNamespaces() []string
	// SetActiveNamespaces updates the locally-synced namespace filter, pushed from the
	// host over the Subscribe("namespaces.active") gRPC stream.
	SetActiveNamespaces(namespaces []string) error
}

// RESTClientGetterFactory builds a genericclioptions.RESTClientGetter wired to a live
// rest.Config, so Helm SDK actions (action.Configuration.Init) honor the currently
// active cluster context without re-reading kubeconfig from disk.
type RESTClientGetterFactory interface {
	NewRESTClientGetter(rc *rest.Config, rules *clientcmd.ClientConfigLoadingRules, overrides *clientcmd.ConfigOverrides) genericclioptions.RESTClientGetter
}
