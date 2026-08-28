package async

import (
	"context"

	"github.com/litelensapp/litelens-plugins/plugins/helm/internal/applications/dto"
	"github.com/litelensapp/litelens-plugins/plugins/helm/internal/applications/port"
)

// Handler dispatches deserialized events to the receiver ports. Takes port interfaces, not
// the concrete kube.DynamicClusterProvider, so this presentation-layer file stays decoupled
// from infrastructure per this repo's hexagonal-architecture convention.
type Handler struct {
	receiver port.EventReceiver
}

func NewHandler(receiver port.EventReceiver) *Handler {
	return &Handler{receiver: receiver}
}

func (h *Handler) handleClusterContext(event *dto.ClusterContextEvent) error {
	return h.receiver.SyncClusterContext(context.Background(), event.ContextName, event.KubeconfigPath)
}

func (h *Handler) handleActiveNamespaces(event *dto.ActiveNamespacesEvent) error {
	return h.receiver.SyncActiveNamespaces(context.Background(), event.Namespaces)
}
