package async

import (
	"github.com/litelensapp/litelens-plugins/plugins/helm/internal/applications/port"
	"github.com/litelensapp/litelens/packages/core/util"
)

type EventDispatcher struct {
	routes []EventRoute
}

// NewEventDispatcher wires receiver's routes onto a new dispatcher, but does not start
// them — call StartAll to launch the event loops.
func NewEventDispatcher(receiver port.EventReceiver) *EventDispatcher {
	h := NewHandler(receiver)

	return &EventDispatcher{
		routes: []EventRoute{
			NewRoute(string(util.EventTopicClusterContext), h.handleClusterContext, deserializeClusterContext),
			NewRoute(string(util.EventTopicNamespacesActive), h.handleActiveNamespaces, deserializeActiveNamespaces),
		},
	}
}

func (ed *EventDispatcher) StartAll(grpcAddr, authToken string) {
	for _, route := range ed.routes {
		go route.Run(grpcAddr, authToken)
	}
}
