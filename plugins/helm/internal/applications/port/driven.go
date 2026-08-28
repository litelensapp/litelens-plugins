package port

import (
	"context"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// EventEmitter is a callback function to emit events from the helm service.
type EventEmitter func(ctx context.Context, eventName string, data any)

// ClusterProvider provides access to active cluster clients and configuration.
type ClusterProvider interface {
	ActiveClients() (cs *kubernetes.Clientset, rc *rest.Config, activeContext string, kubeconfigPaths []string)
	// ActiveNamespaces returns the host's current namespace filter, synced from the
	// host app over the Subscribe("namespaces.active") gRPC stream. An empty slice means
	// cluster-wide (no filter).
	ActiveNamespaces() []string
	Ctx() context.Context
}

// MutableClusterProvider is a ClusterProvider whose active context can be changed
// after construction — implemented by the plugin subprocess's dynamic provider so
// the app can sync it to whatever cluster context is currently active, per call.
type MutableClusterProvider interface {
	ClusterProvider
	SetActiveContext(contextName, kubeconfigPath string) error
}

// RESTClientGetterFactory builds a genericclioptions.RESTClientGetter wired to a live
// rest.Config, so Helm SDK actions (action.Configuration.Init) honor the currently
// active cluster context without re-reading kubeconfig from disk.
type RESTClientGetterFactory interface {
	NewRESTClientGetter(rc *rest.Config, rules *clientcmd.ClientConfigLoadingRules, overrides *clientcmd.ConfigOverrides) genericclioptions.RESTClientGetter
}

// EventReceiver is satisfied by any type implementing both driven ports the plugin's event
// routes dispatch to (e.g. *kube.DynamicClusterProvider).
type EventReceiver interface {
	// SyncClusterContext syncs a cluster context update received from the host.
	SyncClusterContext(ctx context.Context, contextName, kubeconfigPath string) error
	// SyncActiveNamespaces syncs an active-namespaces update received from the host.
	SyncActiveNamespaces(ctx context.Context, namespaces []string) error
}
