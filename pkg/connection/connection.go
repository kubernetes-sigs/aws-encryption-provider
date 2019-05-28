package connection

import (
	"fmt"
	"net"
	"time"

	"google.golang.org/grpc"
)

// New returns a grpc client connection for a given unix socket file path
func New(addr string) (*grpc.ClientConn, error) {
	dialer := func(addr string, timeout time.Duration) (net.Conn, error) {
		return net.DialTimeout("unix", addr, timeout)
	}

	conn, err := grpc.Dial(addr, grpc.WithDialer(dialer), grpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("failed to create connection to socket: %v", err)
	}

	return conn, nil
}
