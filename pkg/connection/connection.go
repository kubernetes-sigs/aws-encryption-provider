package connection

import (
	"fmt"
	"net"
	"os"
	"syscall"
	"time"

	"github.com/golang/glog"
	"google.golang.org/grpc"
)

// New returns a grpc client connection for a given unix socket file path
func New(addr string) (*grpc.ClientConn, error) {
	dialer := func(addr string, timeout time.Duration) (net.Conn, error) {
		return net.DialTimeout("unix", addr, timeout)
	}

	if _, err := os.Stat(addr); err == nil {
		glog.Infof("socket file %s already exists, removing it", addr)
		err := syscall.Unlink(addr)
		if err != nil {
			glog.Warningf("failed to remove existing socket file %s. %v", addr, err.Error())
		}
	}

	conn, err := grpc.Dial(addr, grpc.WithDialer(dialer), grpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("failed to create connection to socket: %v", err)
	}

	return conn, nil
}
