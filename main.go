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
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/kubernetes-sigs/aws_encryption-provider/cloud"
	"github.com/kubernetes-sigs/aws_encryption-provider/plugin"
	"github.com/kubernetes-sigs/aws_encryption-provider/server"
)

var (
	addr   = flag.String("listen", "/tmp/awsencryptionprovider.sock", "GRPC listen address")
	key    = flag.String("key", "", "AWS KMS Key")
	region = flag.String("region", "us-east-1", "AWS Region")
)

func main() {
	flag.Parse()

	sigs := make(chan os.Signal, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	c, err := cloud.New(*region)
	if err != nil {
		log.Fatalf("Failed to create new KMS service: %v", err)
	}

	s := server.New()
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	defer func() {
		log.Println("Shutting down server")
		s.GracefulStop()
	}()

	plugin.New(*key, c).Register(s.Server)

	go func() {
		if err := s.ListenAndServe(*addr); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	log.Printf("Server listening at %s", *addr)

	<-sigs
}
