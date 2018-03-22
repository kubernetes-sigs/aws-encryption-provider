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

package plugin

import (
	"context"
	"fmt"
	"testing"

	"github.com/kubernetes-sigs/aws_encryption-provider/cloud"
	pb "k8s.io/apiserver/pkg/storage/value/encrypt/envelope/v1beta1"
)

var (
	key              = "fakekey"
	encryptedMessage = "aGVsbG8gd29ybGQ="
	plainMessage     = "hello world"
	errorMessage     = fmt.Errorf("oops")
)

func TestEncrypt(t *testing.T) {
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

	c := &cloud.KMSMock{}
	ctx := context.Background()

	for _, test := range tests {
		c.SetEncryptResp(test.output, test.err)
		p := New(key, c)

		eReq := &pb.EncryptRequest{Plain: []byte(test.input)}
		eRes, err := p.Encrypt(ctx, eReq)

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

	c := &cloud.KMSMock{}
	ctx := context.Background()

	for _, test := range tests {
		c.SetDecryptResp(test.output, test.err)
		p := New(key, c)

		dReq := &pb.DecryptRequest{Cipher: []byte(test.input)}
		dRes, err := p.Decrypt(ctx, dReq)

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
