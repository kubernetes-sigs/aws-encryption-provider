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
	l, err := net.Listen("unix", addr)
	if err != nil {
		return fmt.Errorf("failed to create listener: %v", err)
	}

	return s.Serve(l)
}
