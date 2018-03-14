package client

import (
	"fmt"
	"net"
	"time"

	"google.golang.org/grpc"
	pb "k8s.io/apiserver/pkg/storage/value/encrypt/envelope/v1beta1"
)

func New(addr string) (*grpc.ClientConn, pb.KeyManagementServiceClient, error) {
	dialer := func(addr string, timeout time.Duration) (net.Conn, error) {
		return net.DialTimeout("unix", addr, timeout)
	}

	conn, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithDialer(dialer))
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to create connection to socket: %v", err)
	}

	return conn, pb.NewKeyManagementServiceClient(conn), nil
}
