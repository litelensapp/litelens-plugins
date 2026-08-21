package grpc

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/litelensapp/litelens/packages/core/pb"
	grpclib "google.golang.org/grpc"
)

type HostEventEmitter struct {
	client pb.PluginClient
}

func NewHostEventEmitter(conn grpclib.ClientConnInterface) *HostEventEmitter {
	return &HostEventEmitter{client: pb.NewPluginClient(conn)}
}

func (e *HostEventEmitter) Emit(ctx context.Context, eventName string, pluginID string, payload interface{}) {
	var payloadJSON string
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			fmt.Printf("emit event: marshal payload: %v\n", err)
			return
		}
		payloadJSON = string(b)
	}
	go func() {
		emitCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, err := e.client.EmitEvent(emitCtx, &pb.PluginEventRequest{
			PluginId:    pluginID,
			EventName:   eventName,
			PayloadJson: payloadJSON,
		})
		if err != nil {
			fmt.Printf("emit event %q: %v\n", eventName, err)
		}
	}()
}
