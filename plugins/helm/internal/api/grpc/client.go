// Package grpc holds the plugin subprocess's outbound gRPC client to the host
// app — dialing the host's gRPC server and subscribing to cluster context
// change events.
package grpc

import (
	"context"
	"fmt"

	grpclib "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/litelensapp/litelens/packages/core/pb"
)

// ClusterContextStream is the subset of pb.Plugin_ClusterContextWatchClient that
// callers need, kept narrow so tests can supply a fake stream.
type ClusterContextStream interface {
	Recv() (*pb.ClusterContextChangedEvent, error)
}

// DialAndSubscribe dials the host's gRPC server and opens the ClusterContextWatch stream.
func DialAndSubscribe(addr string) (*grpclib.ClientConn, ClusterContextStream, error) {
	conn, err := grpclib.NewClient(addr, grpclib.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, fmt.Errorf("dial host grpc server: %w", err)
	}
	stream, err := pb.NewPluginClient(conn).ClusterContextWatch(context.Background(), &pb.Empty{})
	if err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("subscribe to cluster context watch: %w", err)
	}
	return conn, stream, nil
}

// ActiveNamespacesStream is the subset of pb.Plugin_ActiveNamespacesWatchClient that
// callers need, kept narrow so tests can supply a fake stream.
type ActiveNamespacesStream interface {
	Recv() (*pb.ActiveNamespacesChangedEvent, error)
}

// DialAndSubscribeActiveNamespaces dials the host's gRPC server and opens the
// ActiveNamespacesWatch stream.
func DialAndSubscribeActiveNamespaces(addr string) (*grpclib.ClientConn, ActiveNamespacesStream, error) {
	conn, err := grpclib.NewClient(addr, grpclib.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, fmt.Errorf("dial host grpc server: %w", err)
	}
	stream, err := pb.NewPluginClient(conn).ActiveNamespacesWatch(context.Background(), &pb.Empty{})
	if err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("subscribe to active namespaces watch: %w", err)
	}
	return conn, stream, nil
}
