package connection

import (
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// New returns a grpc client connection for a given unix socket file path
func New(addr string) (*grpc.ClientConn, error) {
	conn, err := grpc.NewClient(
		"unix://"+addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection to socket: %v", err)
	}

	return conn, nil
}
