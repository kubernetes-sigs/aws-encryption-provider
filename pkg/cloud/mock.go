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
	"context"
	"sync"

	"github.com/aws/aws-sdk-go-v2/service/kms"
)

type EncryptAssertion func(params *kms.EncryptInput) bool
type DecryptAssertion func(params *kms.DecryptInput) bool

type EncryptRule struct {
	Assertion EncryptAssertion
	Output    *kms.EncryptOutput
	Error     error
}

type DecryptRule struct {
	Assertion DecryptAssertion
	Output    *kms.DecryptOutput
	Error     error
}

type KMSMock struct {
	AWSKMSv2

	mutex sync.RWMutex

	// Default responses
	defaultEncOut *kms.EncryptOutput
	defaultEncErr error
	defaultDecOut *kms.DecryptOutput
	defaultDecErr error

	// Conditional rules (evaluated in order)
	encryptRules []EncryptRule
	decryptRules []DecryptRule
}

// SetDefaultEncryptResp sets the default encrypt response
func (m *KMSMock) SetDefaultEncryptResp(enc string, encErr error) *KMSMock {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.defaultEncOut = &kms.EncryptOutput{CiphertextBlob: []byte(enc)}
	m.defaultEncErr = encErr
	return m
}

// SetDefaultDecryptResp sets the default decrypt response
func (m *KMSMock) SetDefaultDecryptResp(dec string, decErr error) *KMSMock {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.defaultDecOut = &kms.DecryptOutput{Plaintext: []byte(dec)}
	m.defaultDecErr = decErr
	return m
}

// Legacy methods for backward compatibility
func (m *KMSMock) SetEncryptResp(enc string, encErr error) *KMSMock {
	return m.SetDefaultEncryptResp(enc, encErr)
}

func (m *KMSMock) SetDecryptResp(dec string, decErr error) *KMSMock {
	return m.SetDefaultDecryptResp(dec, decErr)
}

// AddEncryptRule adds a conditional encrypt rule
func (m *KMSMock) AddEncryptRule(assertion EncryptAssertion, enc string, encErr error) *KMSMock {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	rule := EncryptRule{
		Assertion: assertion,
		Output:    &kms.EncryptOutput{CiphertextBlob: []byte(enc)},
		Error:     encErr,
	}
	m.encryptRules = append(m.encryptRules, rule)
	return m
}

// AddDecryptRule adds a conditional decrypt rule
func (m *KMSMock) AddDecryptRule(assertion DecryptAssertion, dec string, decErr error) *KMSMock {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	rule := DecryptRule{
		Assertion: assertion,
		Output:    &kms.DecryptOutput{Plaintext: []byte(dec)},
		Error:     decErr,
	}
	m.decryptRules = append(m.decryptRules, rule)
	return m
}

// ClearRules removes all conditional rules
func (m *KMSMock) ClearRules() *KMSMock {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.encryptRules = nil
	m.decryptRules = nil
	return m
}

func (m *KMSMock) Encrypt(ctx context.Context, params *kms.EncryptInput, optFns ...func(*kms.Options)) (*kms.EncryptOutput, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Check conditional rules first (in order)
	for _, rule := range m.encryptRules {
		if rule.Assertion(params) {
			return rule.Output, rule.Error
		}
	}

	// Fall back to default response
	return m.defaultEncOut, m.defaultEncErr
}

func (m *KMSMock) Decrypt(ctx context.Context, params *kms.DecryptInput, optFns ...func(*kms.Options)) (*kms.DecryptOutput, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Check conditional rules first (in order)
	for _, rule := range m.decryptRules {
		if rule.Assertion(params) {
			return rule.Output, rule.Error
		}
	}

	// Fall back to default response
	return m.defaultDecOut, m.defaultDecErr
}
