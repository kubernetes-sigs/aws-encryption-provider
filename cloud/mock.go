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

package cloud

import (
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/kms/kmsiface"
)

type kmsMock struct {
	kmsiface.KMSAPI
	encryptFunc func() (*kms.EncryptOutput, error)
	decryptFunc func() (*kms.DecryptOutput, error)
}

func NewKMSMock(enc string, encErr error, dec string, decErr error) *kmsMock {
	return &kmsMock{
		encryptFunc: func() (*kms.EncryptOutput, error) {
			return &kms.EncryptOutput{CiphertextBlob: []byte(enc)}, encErr
		},
		decryptFunc: func() (*kms.DecryptOutput, error) {
			return &kms.DecryptOutput{Plaintext: []byte(dec)}, decErr
		},
	}
}

func (m *kmsMock) Encrypt(input *kms.EncryptInput) (*kms.EncryptOutput, error) {
	return m.encryptFunc()
}

func (m *kmsMock) Decrypt(input *kms.DecryptInput) (*kms.DecryptOutput, error) {
	return m.decryptFunc()
}
