/*
Copyright 2020 The Kubernetes Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package server

import (
	"fmt"
	"net"
	"os"

	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type Server struct {
	*grpc.Server
}

func New() *Server {
	return &Server{
		grpc.NewServer(),
	}
}

func (s *Server) ListenAndServe(addr string) error {
	// Server should remove the socket file prior to binding it in case the socket isn't cleaned up gracefully.
	// This can happen if the application is killed by SIGKILL or SIGSTOP, i.e. kill -9 or docker kill by default.
	if _, err := os.Stat(addr); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to os.Stat socket: %v", err)
		}
	} else {
		// the socket file exists, it should be removed
		zap.L().Info("Removing existing socket", zap.String("address", addr))
		if err = os.Remove(addr); err != nil {
			if !os.IsNotExist(err) {
				return fmt.Errorf("failed to os.Remove existing socket: %v", err)
			}
		}
	}
	l, err := net.Listen("unix", addr)
	if err != nil {
		return fmt.Errorf("failed to create listener: %v", err)
	}

	return s.Server.Serve(l)
}
