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

package plugin

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	kmstypes "github.com/aws/aws-sdk-go-v2/service/kms/types"
	smithy "github.com/aws/smithy-go"
	"go.uber.org/zap"
	pb "k8s.io/kms/apis/v1beta1"
	"sigs.k8s.io/aws-encryption-provider/pkg/cloud"
	"sigs.k8s.io/aws-encryption-provider/pkg/kmsplugin"
)

var (
	key                = "fakekey"
	encryptedMessage   = "aGVsbG8gd29ybGQ="
	encryptedMessageV2 = "1aGVsbG8gd29ybGQ="
	plainMessage       = "hello world"
	errorMessage       = fmt.Errorf("oops")
)

func TestEncrypt(t *testing.T) {
	tt := []struct {
		input     string
		ctx       map[string]string
		output    string
		err       error
		errType   kmsplugin.KMSErrorType
		healthErr bool
		checkErr  bool
	}{
		{
			input:     plainMessage,
			ctx:       nil,
			output:    encryptedMessage,
			err:       nil,
			errType:   kmsplugin.KMSErrorTypeNil,
			healthErr: false,
			checkErr:  false,
		},
		{
			input:     plainMessage,
			ctx:       nil,
			output:    "",
			err:       errorMessage,
			errType:   kmsplugin.KMSErrorTypeOther,
			healthErr: true,
			checkErr:  true,
		},
		{
			input:  plainMessage,
			ctx:    nil,
			output: "",
			err: &smithy.GenericAPIError{
				Code:    "RequestLimitExceeded",
				Message: "test",
				Fault:   0,
			},
			errType:   kmsplugin.KMSErrorTypeThrottled,
			healthErr: true,
			checkErr:  false,
		},
		{
			input:     plainMessage,
			ctx:       nil,
			output:    "",
			err:       &kmstypes.KMSInternalException{Message: aws.String("test")},
			errType:   kmsplugin.KMSErrorTypeOther,
			healthErr: true,
			checkErr:  true,
		},
		{
			input:     plainMessage,
			ctx:       nil,
			output:    "",
			err:       &kmstypes.LimitExceededException{Message: aws.String("test")},
			errType:   kmsplugin.KMSErrorTypeThrottled,
			healthErr: true,
			checkErr:  false,
		},
		{
			input:  plainMessage,
			ctx:    nil,
			output: "",
			err: &smithy.GenericAPIError{
				Code:    "AccessDeniedException",
				Message: "The ciphertext refers to a customer master key that does not exist, does not exist in this region, or you are not allowed to access",
				Fault:   0,
			},
			errType:   kmsplugin.KMSErrorTypeUserInduced,
			healthErr: true,
			checkErr:  false,
		},
		{
			input:  plainMessage,
			ctx:    nil,
			output: "",
			err: &smithy.GenericAPIError{
				Code:    "AccessDeniedException",
				Message: "Some other error message",
				Fault:   0,
			},
			errType:   kmsplugin.KMSErrorTypeOther,
			healthErr: true,
			checkErr:  true,
		},
		{
			input:     plainMessage,
			ctx:       nil,
			output:    "",
			err:       &kmstypes.DisabledException{Message: aws.String("test")},
			errType:   kmsplugin.KMSErrorTypeUserInduced,
			healthErr: true,
			checkErr:  false,
		},
		{
			input:     plainMessage,
			ctx:       nil,
			output:    "",
			err:       &kmstypes.KMSInvalidStateException{Message: aws.String("test")},
			errType:   kmsplugin.KMSErrorTypeUserInduced,
			healthErr: true,
			checkErr:  false,
		},
		{
			input:     plainMessage,
			ctx:       nil,
			output:    "",
			err:       &kmstypes.InvalidGrantIdException{Message: aws.String("test")},
			errType:   kmsplugin.KMSErrorTypeUserInduced,
			healthErr: true,
			checkErr:  false,
		},
		{
			input:     plainMessage,
			ctx:       nil,
			output:    "",
			err:       &kmstypes.InvalidGrantTokenException{Message: aws.String("test")},
			errType:   kmsplugin.KMSErrorTypeUserInduced,
			healthErr: true,
			checkErr:  false,
		},
		{
			input:     plainMessage,
			ctx:       make(map[string]string),
			output:    encryptedMessage,
			err:       nil,
			errType:   kmsplugin.KMSErrorTypeNil,
			healthErr: false,
			checkErr:  false,
		},
		{
			input:     encryptedMessage,
			ctx:       map[string]string{"a": "b"},
			output:    "",
			err:       errors.New("invalid context"),
			errType:   kmsplugin.KMSErrorTypeOther,
			healthErr: true,
			checkErr:  true,
		},
	}

	c := &cloud.KMSMock{}
	ctx := context.Background()

	for idx, tc := range tt {
		func() {
			c.SetEncryptResp(tc.output, tc.err)
			sharedHealthCheck := NewSharedHealthCheck(DefaultHealthCheckPeriod, DefaultErrcBufSize)
			go sharedHealthCheck.Start()
			p := New(key, c, nil, sharedHealthCheck)
			defer func() {
				sharedHealthCheck.Stop()
			}()

			//nolint:staticcheck
			eReq := &pb.EncryptRequest{Plain: []byte(tc.input)}
			eRes, err := p.Encrypt(ctx, eReq)

			if tc.err != nil && err == nil {
				t.Fatalf("#%d: failed to return expected error %v", idx, tc.err)
			}

			if tc.err == nil && err != nil {
				t.Fatalf("#%d: returned unexpected error: %v", idx, err)
			}

			//nolint:staticcheck
			if tc.err == nil && string(eRes.Cipher) != kmsplugin.StorageVersion+tc.output {
				//nolint:staticcheck
				t.Fatalf("#%d: expected %s, but got %s", idx, kmsplugin.StorageVersion+tc.output, string(eRes.Cipher))
			}

			et := kmsplugin.ParseError(tc.err)
			if !reflect.DeepEqual(tc.errType, et) {
				t.Fatalf("#%d: expected error type %s, got %s", idx, tc.errType, et)
			}

			herr := p.Health()
			if tc.healthErr && herr == nil {
				t.Fatalf("#%d: expected health error, but got nil", idx)
			}
			if !tc.healthErr && herr != nil {
				t.Fatalf("#%d: unexpected health error, got %v", idx, herr)
			}

			cerr := p.Live()
			if tc.checkErr && cerr == nil {
				t.Fatalf("#%d: expected check error, but got nil", idx)
			}
			if !tc.checkErr && cerr != nil {
				t.Fatalf("#%d: unexpected check error, got %v", idx, cerr)
			}
		}()
	}
}
func TestDecrypt(t *testing.T) {
	tt := []struct {
		input  string
		ctx    map[string]string
		output string
		err    error
	}{
		{
			input:  encryptedMessage,
			ctx:    nil,
			output: plainMessage,
			err:    nil,
		},
		{
			input:  encryptedMessage,
			ctx:    nil,
			output: "",
			err:    errorMessage,
		},
		{
			input:  encryptedMessage,
			ctx:    map[string]string{"a": "b"},
			output: "",
			err:    errors.New("invalid context"),
		},
	}

	c := &cloud.KMSMock{}
	ctx := context.Background()

	for _, tc := range tt {
		func() {
			c.SetDecryptResp(tc.output, tc.err)
			sharedHealthCheck := NewSharedHealthCheck(DefaultHealthCheckPeriod, DefaultErrcBufSize)
			go sharedHealthCheck.Start()
			p := New(key, c, tc.ctx, sharedHealthCheck)
			defer func() {
				sharedHealthCheck.Stop()
			}()

			//nolint:staticcheck
			dReq := &pb.DecryptRequest{Cipher: []byte(tc.input)}
			dRes, err := p.Decrypt(ctx, dReq)

			if tc.err != nil && err == nil {
				t.Fatalf("Failed to return expected error %v", tc.err)
			}

			if tc.err == nil && err != nil {
				t.Fatalf("Returned unexpected error: %v", err)
			}

			//nolint:staticcheck
			if tc.err == nil && string(dRes.Plain) != tc.output {
				//nolint:staticcheck
				t.Fatalf("Expected %s, but got %s", tc.output, string(dRes.Plain))
			}
		}()
	}
}

func TestHealth(t *testing.T) {
	zap.ReplaceGlobals(zap.NewExample())

	tt := []struct {
		encryptErr error
		decryptErr error
	}{
		{
			encryptErr: nil,
			decryptErr: nil,
		},
		{
			encryptErr: errors.New("encrypt fail"),
			decryptErr: errors.New("decrypt fail"),
		},
	}
	for idx, entry := range tt {
		c := &cloud.KMSMock{}
		sharedHealthCheck := NewSharedHealthCheck(DefaultHealthCheckPeriod, DefaultErrcBufSize)
		go sharedHealthCheck.Start()
		p := New(key, c, nil, sharedHealthCheck)
		defer func() {
			sharedHealthCheck.Stop()
		}()

		c.SetEncryptResp("foo", entry.encryptErr)
		c.SetDecryptResp("foo", entry.decryptErr)

		//nolint:staticcheck
		_, encErr := p.Encrypt(context.Background(), &pb.EncryptRequest{Plain: []byte("foo")})
		if entry.encryptErr == nil && encErr != nil {
			t.Fatalf("#%d: unexpected error from Encrypt %v", idx, encErr)
		}
		herr1 := p.Health()
		if entry.encryptErr == nil {
			if herr1 != nil {
				t.Fatalf("#%d: unexpected error from Health %v", idx, herr1)
			}
		} else if !strings.HasSuffix(encErr.Error(), entry.encryptErr.Error()) {
			t.Fatalf("#%d: unexpected error from Health %v", idx, herr1)
		}

		//nolint:staticcheck
		_, decErr := p.Decrypt(context.Background(), &pb.DecryptRequest{Cipher: []byte("foo")})
		if entry.decryptErr == nil && decErr != nil {
			t.Fatalf("#%d: unexpected error from Encrypt %v", idx, decErr)
		}
		herr2 := p.Health()
		if entry.decryptErr == nil {
			if herr2 != nil {
				t.Fatalf("#%d: unexpected error from Health %v", idx, herr2)
			}
		} else if !strings.HasSuffix(decErr.Error(), entry.decryptErr.Error()) {
			t.Fatalf("#%d: unexpected error from Health %v", idx, herr2)
		}
	}
}

// TestHealthManyRequests sends many requests to fill up the error channel,
// and ensures following encrypt/decrypt operation do not block.
func TestHealthManyRequests(t *testing.T) {
	zap.ReplaceGlobals(zap.NewExample())

	c := &cloud.KMSMock{}
	sharedHealthCheck := NewSharedHealthCheck(DefaultHealthCheckPeriod, DefaultErrcBufSize)
	go sharedHealthCheck.Start()
	p := newPlugin(key, c, nil, sharedHealthCheck)
	defer func() {
		sharedHealthCheck.Stop()
	}()

	c.SetEncryptResp("foo", errors.New("fail"))
	for i := 0; i < 10; i++ {
		errc := make(chan error)
		go func() {
			//nolint:staticcheck
			_, err := p.Encrypt(
				context.Background(),
				&pb.EncryptRequest{Plain: []byte("foo")},
			)
			errc <- err
		}()
		select {
		case <-time.After(time.Second):
			t.Fatalf("#%d: Encrypt took longer than it should", i)
		case err := <-errc:
			if !strings.HasSuffix(err.Error(), "fail") {
				t.Fatalf("#%d: unexpected errro %v", i, err)
			}
		}
	}
}
