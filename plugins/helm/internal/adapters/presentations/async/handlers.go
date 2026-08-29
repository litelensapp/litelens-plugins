package async

import (
	"context"

	"github.com/litelensapp/litelens-plugins/plugins/helm/internal/applications/port"
	"github.com/litelensapp/litelens/packages/core/async"
)

// ackFunc publishes an ack for requestID back to the host (see
// HostPluginServer.PublishAndAwaitAck, host repo) once an event has been applied.
type ackFunc func(ctx context.Context, requestID string)

// Handler dispatches deserialized events to the receiver ports. Takes port interfaces, not
// the concrete kube.ClusterProvider, so this presentation-layer file stays decoupled
// from infrastructure per this repo's hexagonal-architecture convention.
type Handler struct {
	receiver port.KubeClusterProvider
	ack      ackFunc
}

func NewHandler(receiver port.KubeClusterProvider, ack ackFunc) *Handler {
	return &Handler{receiver: receiver, ack: ack}
}

func (h *Handler) handleClusterContext(event *async.ClusterContextEvent) error {
	var err error
	if event.Clearing {
		err = h.receiver.ClearActiveContext(context.Background())
	} else {
		err = h.receiver.SyncClusterContext(context.Background(), event.ContextName, event.KubeconfigPath)
	}
	h.maybeAck(event.RequestID)
	return err
}

func (h *Handler) handleActiveNamespaces(event *async.ActiveNamespacesEvent) error {
	var err error
	if event.Clearing {
		err = h.receiver.ClearActiveNamespaces(context.Background())
	} else {
		err = h.receiver.SyncActiveNamespaces(context.Background(), event.Namespaces)
	}
	h.maybeAck(event.RequestID)
	return err
}

// maybeAck acks unconditionally (even if the apply above errored) — the ack means
// "delivered and an apply was attempted," letting the host's bounded
// PublishAndAwaitAck wait resolve promptly instead of always burning its full
// timeout on a plugin-side error the host can't otherwise observe synchronously.
func (h *Handler) maybeAck(requestID string) {
	if requestID == "" || h.ack == nil {
		return
	}
	h.ack(context.Background(), requestID)
}
