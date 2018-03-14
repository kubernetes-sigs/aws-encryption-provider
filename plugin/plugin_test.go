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
	"testing"

	"github.com/kubernetes-sigs/aws_encryption-provider/cloud"
	pb "k8s.io/apiserver/pkg/storage/value/encrypt/envelope/v1beta1"
)

func TestEncryptDecrypt(t *testing.T) {
	var (
		key     = "fakekey"
		testReq = "hello world"
		testRes = "aGVsbG8gd29ybGQ="
	)

	c := cloud.NewKMSMock(testRes, nil, testReq, nil)

	p := New(key, c)

	eReq := &pb.EncryptRequest{Plain: []byte(testReq)}
	eRes, err := p.Encrypt(context.Background(), eReq)
	if err != nil {
		t.Fatalf("Failed to encrypt data: %v", err)
	}

	if string(eRes.Cipher) != testRes {
		t.Fatalf("Expected %s, but got %s", testRes, string(eRes.Cipher))
	}

	dReq := &pb.DecryptRequest{Cipher: []byte(eRes.Cipher)}
	dRes, err := p.Decrypt(context.Background(), dReq)
	if err != nil {
		t.Fatalf("Failed to decrypt data: %v", err)
	}

	if string(dRes.Plain) != testReq {
		t.Fatalf("Expected %s, but got %s", testReq, string(dRes.Plain))
	}
}
