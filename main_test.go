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
	"context"
	"testing"
	"time"

	"github.com/kubernetes-sigs/aws_encryption-provider/client"
	"github.com/kubernetes-sigs/aws_encryption-provider/cloud"
	"github.com/kubernetes-sigs/aws_encryption-provider/plugin"
	"github.com/kubernetes-sigs/aws_encryption-provider/server"
	pb "k8s.io/apiserver/pkg/storage/value/encrypt/envelope/v1beta1"
)

func TestServer(t *testing.T) {
	var (
		addr    = "test.sock"
		key     = "fakekey"
		testReq = "hello world"
		testRes = "aGVsbG8gd29ybGQ="
	)

	s := server.New()

	defer func() {
		t.Log("Shutting down server")
		s.GracefulStop()
	}()

	c := cloud.NewKMSMock(testRes, nil, testReq, nil)

	plugin.New(key, c).Register(s.Server)

	go func() {
		if err := s.ListenAndServe(addr); err != nil {
			t.Fatalf("Failed to start server: %v", err)
		}
	}()

	time.Sleep(5 * time.Millisecond)

	conn, client, err := client.New(addr)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer conn.Close()

	eReq := &pb.EncryptRequest{Plain: []byte(testReq)}
	eRes, err := client.Encrypt(context.Background(), eReq)
	if err != nil {
		t.Fatalf("Failed to encrypt data: %v", err)
	}

	if string(eRes.Cipher) != testRes {
		t.Fatalf("Expected %s, but got %s", testRes, string(eRes.Cipher))
	}

	dReq := &pb.DecryptRequest{Cipher: []byte(eRes.Cipher)}
	dRes, err := client.Decrypt(context.Background(), dReq)
	if err != nil {
		t.Fatalf("Failed to decrypt data: %v", err)
	}

	if string(dRes.Plain) != testReq {
		t.Fatalf("Expected %s, but got %s", testReq, string(dRes.Plain))
	}
}
