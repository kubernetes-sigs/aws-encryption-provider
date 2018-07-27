/*
Copyright 2018 The Kubernetes Authors.
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

package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/golang/glog"
	"github.com/kubernetes-sigs/aws-encryption-provider/pkg/cloud"
	"github.com/kubernetes-sigs/aws-encryption-provider/pkg/connection"
	"github.com/kubernetes-sigs/aws-encryption-provider/pkg/plugin"
	"github.com/kubernetes-sigs/aws-encryption-provider/pkg/server"
)

func main() {
	var (
		healthzPath = flag.String("healthz-path", "/healthz", "healthcheck path")
		healthzPort = flag.String("health-port", ":8080", "healthcheck port")
		addr        = flag.String("listen", "/var/run/kmsplugin/socket.sock", "GRPC listen address")
		key         = flag.String("key", "", "AWS KMS Key")
		region      = flag.String("region", "us-east-1", "AWS Region")
	)

	flag.Set("logtostderr", "true")

	flag.Parse()

	c, err := cloud.New(*region)
	if err != nil {
		glog.Fatalf("Failed to create new KMS service: %v", err)
	}

	s := server.New()
	p := plugin.New(*key, c)

	p.Register(s.Server)

	conn, err := connection.New(*addr)
	if err != nil {
		glog.Fatalf("Failed to create connection: %v", err)
	}

	client := p.NewClient(conn)

	go func() {
		http.HandleFunc(*healthzPath, func(w http.ResponseWriter, r *http.Request) {
			res, err := p.Check(client)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprint(w, http.StatusText(http.StatusInternalServerError))
				glog.Errorf("Failed healthceck: %v", err)
			} else {
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, http.StatusText(http.StatusOK))
				glog.Infof("Passed healthceck: %v", res)
			}
		})

		if err := http.ListenAndServe(*healthzPort, nil); err != nil {
			glog.Fatalf("Failed to start healthcheck server: %v", err)
		}
	}()

	go func() {
		if err := s.ListenAndServe(*addr); err != nil {
			glog.Fatalf("Failed to start server: %v", err)
		}
	}()

	glog.Infof("Healthchecks listening on %s", *healthzPort)
	glog.Infof("Server listening at %s", *addr)

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	signal := <-signals

	glog.Infof("Captured %v", signal)
	glog.Infof("Closing client connection")
	conn.Close()
	glog.Infof("Shutting down server")
	s.GracefulStop()
	glog.Infof("Exiting...")
	os.Exit(0)
}
