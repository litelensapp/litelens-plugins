package grpc

import (
	"encoding/json"
	"fmt"

	"github.com/litelensapp/litelens-plugins/plugins/helm/internal/dto"
)

// ClusterContextEvent represents the payload of a cluster context change event.
type ClusterContextEvent struct {
	ContextName    string `json:"contextName"`
	KubeconfigPath string `json:"kubeconfigPath"`
}

// RecvClusterContext reads the next message from the stream and decodes the
// cluster context event from the JSON payload.
func (c *GrpcClient) RecvClusterContext(stream dto.SubscribeStream) (*ClusterContextEvent, error) {
	msg, err := stream.Recv()
	if err != nil {
		return nil, err
	}

	var event ClusterContextEvent
	if err := json.Unmarshal([]byte(msg.PayloadJson), &event); err != nil {
		return nil, fmt.Errorf("unmarshal cluster context event: %w", err)
	}

	return &event, nil
}

// ActiveNamespacesEvent represents the payload of an active namespaces change event.
type ActiveNamespacesEvent struct {
	Namespaces []string `json:"namespaces"`
}

// RecvActiveNamespaces reads the next message from the stream and decodes the
// active namespaces event from the JSON payload.
func (c *GrpcClient) RecvActiveNamespaces(stream dto.SubscribeStream) (*ActiveNamespacesEvent, error) {
	msg, err := stream.Recv()
	if err != nil {
		return nil, err
	}

	var event ActiveNamespacesEvent
	if err := json.Unmarshal([]byte(msg.PayloadJson), &event); err != nil {
		return nil, fmt.Errorf("unmarshal active namespaces event: %w", err)
	}

	return &event, nil
}
