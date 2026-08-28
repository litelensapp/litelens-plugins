package async

import (
	"github.com/litelensapp/litelens/packages/core/async"
)

// EventDispatcher wires the helm plugin's handler onto the shared core event routes,
// then starts them.
type EventDispatcher struct {
	routes []async.EventRoute
}

// NewEventDispatcher wires receiver's routes onto a new dispatcher, but does not start
// them — call StartAll to launch the event loops.
func NewEventDispatcher(receiver async.EventReceiver) *EventDispatcher {
	h := NewHandler(receiver)

	return &EventDispatcher{
		routes: []async.EventRoute{
			async.NewRoute(string(async.EventTopicClusterContext), h.handleClusterContext, async.DeserializeClusterContext),
			async.NewRoute(string(async.EventTopicNamespacesActive), h.handleActiveNamespaces, async.DeserializeActiveNamespaces),
		},
	}
}

func (ed *EventDispatcher) StartAll(grpcAddr, authToken string) {
	for _, route := range ed.routes {
		go route.Run(grpcAddr, authToken)
	}
}
