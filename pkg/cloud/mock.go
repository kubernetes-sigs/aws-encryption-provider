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

package cloud

import (
	"sync"

	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/kms/kmsiface"
)

type KMSMock struct {
	kmsiface.KMSAPI

	mutex sync.Mutex

	encOut *kms.EncryptOutput
	encErr error
	decOut *kms.DecryptOutput
	decErr error
}

func (m *KMSMock) SetEncryptResp(enc string, encErr error) *KMSMock {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.encOut = &kms.EncryptOutput{CiphertextBlob: []byte(enc)}
	m.encErr = encErr
	return m
}

func (m *KMSMock) SetDecryptResp(dec string, decErr error) *KMSMock {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.decOut = &kms.DecryptOutput{Plaintext: []byte(dec)}
	m.decErr = decErr
	return m
}

func (m *KMSMock) Encrypt(input *kms.EncryptInput) (*kms.EncryptOutput, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.encOut, m.encErr
}

func (m *KMSMock) Decrypt(input *kms.DecryptInput) (*kms.DecryptOutput, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.decOut, m.decErr
}
