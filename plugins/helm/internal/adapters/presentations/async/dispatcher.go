package async

import (
	"context"
	"time"

	"github.com/litelensapp/litelens-plugins/plugins/helm/internal/applications/port"
	"github.com/litelensapp/litelens/packages/core/async"
)

// EventDispatcher wires the helm plugin's handler onto the shared core event routes,
// then starts them.
type EventDispatcher struct {
	receiver port.KubeClusterProvider
	routes   []async.EventRoute
}

// NewEventDispatcher wires receiver's routes onto a new dispatcher, but does not start
// them — call StartAll to launch the event loops. hostClient/pluginID are used to ack
// clear-first/synced pushes back to the host (see HostPluginServer.PublishAndAwaitAck,
// host repo) via the shared GrpcClient.Emit helper, which already publishes to
// plugins.<pluginID>.<eventName> fire-and-forget in a goroutine.
func NewEventDispatcher(receiver port.KubeClusterProvider, hostClient *async.GrpcClient, pluginID string) *EventDispatcher {
	ack := func(ctx context.Context, requestID string) {
		hostClient.Emit(ctx, "ack", pluginID, map[string]string{"requestId": requestID})
	}
	h := NewHandler(receiver, ack)

	return &EventDispatcher{
		receiver: receiver,
		routes: []async.EventRoute{
			async.NewRoute(string(async.EventTopicClusterContext), h.handleClusterContext, async.DeserializeClusterContext),
			async.NewRoute(string(async.EventTopicNamespacesActive), h.handleActiveNamespaces, async.DeserializeActiveNamespaces),
		},
	}
}

// Start launches the event loops without waiting for the initial sync — use when
// BuildClusterProvider seeded no kubeconfig-derived guess to race against (the plugin
// launched app-wide, before any cluster is connected), so there's nothing wrong to
// serve in the meantime and no reason to delay opening the HTTP server.
func (ed *EventDispatcher) Start(grpcAddr, authToken string) {
	for _, route := range ed.routes {
		go route.Run(grpcAddr, authToken)
	}
}

// StartAll launches the event loops, then blocks until the host's replayed
// cluster-context/active-namespaces messages land or syncTimeout elapses — so the
// first business call isn't served against BuildClusterProvider's kubeconfig-derived
// guess (wrong cluster, unfiltered namespaces), a result the frontend would then cache
// indefinitely.
func (ed *EventDispatcher) StartAll(grpcAddr, authToken string, syncTimeout time.Duration) {
	ed.Start(grpcAddr, authToken)
	ed.receiver.WaitForInitialSync(syncTimeout)
}
