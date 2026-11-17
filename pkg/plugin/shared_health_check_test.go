/*
Copyright 2026 The Kubernetes Authors.
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
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	kmstypes "github.com/aws/aws-sdk-go-v2/service/kms/types"
)

func TestSharedHealthCheckKeyRevoked(t *testing.T) {
	disabled := &kmstypes.DisabledException{Message: aws.String("key is disabled")}
	transient := &kmstypes.KMSInternalException{Message: aws.String("internal error")}

	h := NewSharedHealthCheck(DefaultHealthCheckPeriod, DefaultErrcBufSize)
	h.keyRevokedAfter = 50 * time.Millisecond

	h.RecordErr(disabled)
	if h.isKeyRevoked() {
		t.Fatal("key reported revoked before keyRevokedAfter elapsed")
	}

	time.Sleep(60 * time.Millisecond)
	if !h.isKeyRevoked() {
		t.Fatal("key not reported revoked after keyRevokedAfter elapsed")
	}

	// a transient error does not prove the key is usable, but is not a revocation either
	h.RecordErr(transient)
	if h.isKeyRevoked() {
		t.Fatal("key reported revoked while latest error is transient")
	}
	h.RecordErr(disabled)
	if !h.isKeyRevoked() {
		t.Fatal("revocation window should not restart after a transient error")
	}

	// a successful call resets the window
	h.RecordErr(nil)
	if h.isKeyRevoked() {
		t.Fatal("key reported revoked after a successful call")
	}
	h.RecordErr(disabled)
	if h.isKeyRevoked() {
		t.Fatal("revocation window should restart after a successful call")
	}
}

func TestSharedHealthCheckOtherErrorsNotRevoked(t *testing.T) {
	h := NewSharedHealthCheck(DefaultHealthCheckPeriod, DefaultErrcBufSize)
	h.keyRevokedAfter = 0

	for _, err := range []error{
		errors.New("oops"),
		&kmstypes.KMSInternalException{Message: aws.String("internal error")},
		&kmstypes.KeyUnavailableException{Message: aws.String("unavailable")},
	} {
		h.RecordErr(err)
		if h.isKeyRevoked() {
			t.Fatalf("key reported revoked for %v", err)
		}
	}
}
