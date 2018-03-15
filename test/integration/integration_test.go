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

package integration

import (
	"context"
	"fmt"
	"testing"

	"github.com/kubernetes-sigs/aws_encryption-provider/cloud"
	"github.com/kubernetes-sigs/aws_encryption-provider/connection"
	"github.com/kubernetes-sigs/aws_encryption-provider/plugin"
	"github.com/kubernetes-sigs/aws_encryption-provider/server"
	pb "k8s.io/apiserver/pkg/storage/value/encrypt/envelope/v1beta1"
)

var (
	addr             = "test.sock"
	key              = "fakekey"
	encryptedMessage = "aGVsbG8gd29ybGQ="
	plainMessage     = "hello world"
	errorMessage     = fmt.Errorf("oops")
)

func setup(t *testing.T) (*server.Server, *cloud.KMSMock, pb.KeyManagementServiceClient, func() error) {
	s := server.New()
	c := &cloud.KMSMock{}
	p := plugin.New(key, c)
	p.Register(s.Server)
	conn, err := connection.New(addr)
	if err != nil {
		t.Fatalf("Failed to create connection: %v", err)
	}
	return s, c, p.NewClient(conn), conn.Close
}

func TestEncrypt(t *testing.T) {
	server, mock, client, closeConn := setup(t)

	defer func() {
		closeConn()
		server.Stop()
	}()

	go func() {
		if err := server.ListenAndServe(addr); err != nil {
			t.Fatalf("Failed to start server: %v", err)
		}
	}()

	tests := []struct {
		input  string
		output string
		err    error
	}{
		{
			input:  plainMessage,
			output: encryptedMessage,
			err:    nil,
		},
		{
			input:  plainMessage,
			output: encryptedMessage,
			err:    errorMessage,
		},
	}

	ctx := context.Background()

	for _, test := range tests {
		mock.SetEncryptResp(test.output, test.err)

		eReq := &pb.EncryptRequest{Plain: []byte(test.input)}
		eRes, err := client.Encrypt(ctx, eReq)

		if test.err != nil && err == nil {
			t.Fatalf("Failed to return expected error %v", test.err)
		}

		if test.err == nil && err != nil {
			t.Fatalf("Returned unexpected error: %v", err)
		}

		if test.err == nil && string(eRes.Cipher) != test.output {
			t.Fatalf("Expected %s, but got %s", test.output, string(eRes.Cipher))
		}
	}

}

func TestDecrypt(t *testing.T) {
	server, mock, client, closeConn := setup(t)

	defer func() {
		closeConn()
		server.Stop()
	}()

	go func() {
		if err := server.ListenAndServe(addr); err != nil {
			t.Fatalf("Failed to start server: %v", err)
		}
	}()

	tests := []struct {
		input  string
		output string
		err    error
	}{
		{
			input:  encryptedMessage,
			output: plainMessage,
			err:    nil,
		},
		{
			input:  encryptedMessage,
			output: plainMessage,
			err:    errorMessage,
		},
	}

	ctx := context.Background()

	for _, test := range tests {
		mock.SetDecryptResp(test.output, test.err)

		dReq := &pb.DecryptRequest{Cipher: []byte(test.input)}
		dRes, err := client.Decrypt(ctx, dReq)

		if test.err != nil && err == nil {
			t.Fatalf("Failed to return expected error %v", test.err)
		}

		if test.err == nil && err != nil {
			t.Fatalf("Returned unexpected error: %v", err)
		}

		if test.err == nil && string(dRes.Plain) != test.output {
			t.Fatalf("Expected %s, but got %s", test.output, string(dRes.Plain))
		}
	}
}
