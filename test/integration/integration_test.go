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
	"io/ioutil"
	"testing"

	"github.com/kubernetes-sigs/aws-encryption-provider/pkg/cloud"
	"github.com/kubernetes-sigs/aws-encryption-provider/pkg/connection"
	"github.com/kubernetes-sigs/aws-encryption-provider/pkg/plugin"
	"github.com/kubernetes-sigs/aws-encryption-provider/pkg/server"
	pb "k8s.io/apiserver/pkg/storage/value/encrypt/envelope/v1beta1"
)

var (
	key              = "fakekey"
	encryptedMessage = "aGVsbG8gd29ybGQ="
	plainMessage     = "hello world"
	errorMessage     = fmt.Errorf("oops")
)

func setup(t *testing.T) (string, *server.Server, *cloud.KMSMock, pb.KeyManagementServiceClient, func() error) {
	s := server.New()
	c := &cloud.KMSMock{}
	p := plugin.New(key, c)
	p.Register(s.Server)
	dir, err := ioutil.TempDir("", "run")
	if err != nil {
		t.Fatalf("failed to create tmp dir: %v", err)
	}

	addr := dir + "test.sock"

	conn, err := connection.New(addr)
	if err != nil {
		t.Fatalf("Failed to create connection: %v", err)
	}
	return addr, s, c, p.NewClient(conn), conn.Close
}

func TestEncrypt(t *testing.T) {
	addr, server, mock, client, closeConn := setup(t)

	defer func() {
		closeConn()
		server.Stop()
	}()

	go func() {
		if err := server.ListenAndServe(addr); err != nil {
			t.Fatalf("Failed to start server: %v", err)
		}
	}()

	tt := []struct {
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

	for _, tc := range tt {
		mock.SetEncryptResp(tc.output, tc.err)

		eReq := &pb.EncryptRequest{Plain: []byte(tc.input)}
		eRes, err := client.Encrypt(ctx, eReq)

		if tc.err != nil && err == nil {
			t.Fatalf("Failed to return expected error %v", tc.err)
		}

		if tc.err == nil && err != nil {
			t.Fatalf("Returned unexpected error: %v", err)
		}

		if tc.err == nil && string(eRes.Cipher) != plugin.StorageVersion+tc.output {
			t.Fatalf("Expected %s, but got %s", plugin.StorageVersion+tc.output, string(eRes.Cipher))
		}
	}

}

func TestDecrypt(t *testing.T) {
	addr, server, mock, client, closeConn := setup(t)

	defer func() {
		closeConn()
		server.Stop()
	}()

	go func() {
		if err := server.ListenAndServe(addr); err != nil {
			t.Fatalf("Failed to start server: %v", err)
		}
	}()

	tt := []struct {
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

	for _, tc := range tt {
		mock.SetDecryptResp(tc.output, tc.err)

		dReq := &pb.DecryptRequest{Cipher: []byte(tc.input)}
		dRes, err := client.Decrypt(ctx, dReq)

		if tc.err != nil && err == nil {
			t.Fatalf("Failed to return expected error %v", tc.err)
		}

		if tc.err == nil && err != nil {
			t.Fatalf("Returned unexpected error: %v", err)
		}

		if tc.err == nil && string(dRes.Plain) != tc.output {
			t.Fatalf("Expected %s, but got %s", tc.output, string(dRes.Plain))
		}
	}
}
