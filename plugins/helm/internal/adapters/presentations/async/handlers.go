package async

import (
	"context"

	"github.com/litelensapp/litelens/packages/core/async"
)

// Handler dispatches deserialized events to the receiver ports. Takes port interfaces, not
// the concrete kube.DynamicClusterProvider, so this presentation-layer file stays decoupled
// from infrastructure per this repo's hexagonal-architecture convention.
type Handler struct {
	receiver async.EventReceiver
}

func NewHandler(receiver async.EventReceiver) *Handler {
	return &Handler{receiver: receiver}
}

func (h *Handler) handleClusterContext(event *async.ClusterContextEvent) error {
	return h.receiver.SyncClusterContext(context.Background(), event.ContextName, event.KubeconfigPath)
}

func (h *Handler) handleActiveNamespaces(event *async.ActiveNamespacesEvent) error {
	return h.receiver.SyncActiveNamespaces(context.Background(), event.Namespaces)
}
