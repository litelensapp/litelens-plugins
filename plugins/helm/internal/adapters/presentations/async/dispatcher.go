package async

import (
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
// them — call StartAll to launch the event loops.
func NewEventDispatcher(receiver port.KubeClusterProvider) *EventDispatcher {
	h := NewHandler(receiver)

	return &EventDispatcher{
		receiver: receiver,
		routes: []async.EventRoute{
			async.NewRoute(string(async.EventTopicClusterContext), h.handleClusterContext, async.DeserializeClusterContext),
			async.NewRoute(string(async.EventTopicNamespacesActive), h.handleActiveNamespaces, async.DeserializeActiveNamespaces),
		},
	}
}

// StartAll launches the event loops, then blocks until the host's replayed
// cluster-context/active-namespaces messages land or syncTimeout elapses — so the
// first business call isn't served against BuildClusterProvider's kubeconfig-derived
// guess (wrong cluster, unfiltered namespaces), a result the frontend would then cache
// indefinitely.
func (ed *EventDispatcher) StartAll(grpcAddr, authToken string, syncTimeout time.Duration) {
	for _, route := range ed.routes {
		go route.Run(grpcAddr, authToken)
	}
	ed.receiver.WaitForInitialSync(syncTimeout)
}
