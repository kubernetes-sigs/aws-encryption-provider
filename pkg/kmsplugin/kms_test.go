package kmsplugin

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/kms/types"
	"github.com/aws/smithy-go"
	"github.com/stretchr/testify/assert"
)

// mockAPIError implements smithy.APIError interface for testing
type mockAPIError struct {
	code    string
	message string
}

func (e *mockAPIError) Error() string {
	return e.message
}

func (e *mockAPIError) ErrorCode() string {
	return e.code
}

func (e *mockAPIError) ErrorMessage() string {
	return e.message
}

func (e *mockAPIError) ErrorFault() smithy.ErrorFault {
	return smithy.FaultUnknown
}

func TestParseError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected KMSErrorType
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: KMSErrorTypeNil,
		},
		{
			name:     "non-API error",
			err:      errors.New("generic error"),
			expected: KMSErrorTypeOther,
		},
		{
			name:     "DisabledException",
			err:      &mockAPIError{code: (&types.DisabledException{}).ErrorCode()},
			expected: KMSErrorTypeUserInduced,
		},
		{
			name:     "KMSInvalidStateException",
			err:      &mockAPIError{code: (&types.KMSInvalidStateException{}).ErrorCode()},
			expected: KMSErrorTypeUserInduced,
		},
		{
			name:     "KeyUnavailableException",
			err:      &mockAPIError{code: (&types.KeyUnavailableException{}).ErrorCode()},
			expected: KMSErrorTypeUserInduced,
		},
		{
			name:     "InvalidArnException",
			err:      &mockAPIError{code: (&types.InvalidArnException{}).ErrorCode()},
			expected: KMSErrorTypeUserInduced,
		},
		{
			name:     "InvalidGrantIdException",
			err:      &mockAPIError{code: (&types.InvalidGrantIdException{}).ErrorCode()},
			expected: KMSErrorTypeUserInduced,
		},
		{
			name:     "InvalidGrantTokenException",
			err:      &mockAPIError{code: (&types.InvalidGrantTokenException{}).ErrorCode()},
			expected: KMSErrorTypeUserInduced,
		},
		{
			name:     "LimitExceededException",
			err:      &mockAPIError{code: (&types.LimitExceededException{}).ErrorCode()},
			expected: KMSErrorTypeThrottled,
		},
		{
			name:     "InvalidCiphertextException",
			err:      &mockAPIError{code: (&types.InvalidCiphertextException{}).ErrorCode()},
			expected: KMSErrorTypeCorruption,
		},
		{
			name:     "AccessDeniedException caused by key not existing or missing permissions - 1",
			err:      &mockAPIError{code: "AccessDeniedException", message: "The ciphertext refers to a customer master key that does not exist"},
			expected: KMSErrorTypeUserInduced,
		},
		{
			name:     "AccessDeniedException caused by key not existing or missing permissions - 2",
			err:      &mockAPIError{code: "AccessDeniedException", message: "User dummy is not authorized to perform: kms:Decrypt on this resource because the resource does not exist in this Region, no resource-based policies allow access, or a resource-based policy explicitly denies access"},
			expected: KMSErrorTypeUserInduced,
		},
		{
			name:     "AccessDeniedException caused by explicit deny in resource policy",
			err:      &mockAPIError{code: "AccessDeniedException", message: "User dummy is not authorized to perform: kms:Decrypt on this resource with an explicit deny in a resource control policy"},
			expected: KMSErrorTypeUserInduced,
		},
		{
			name:     "AccessDeniedException caused by inactive resource owner account",
			err:      &mockAPIError{code: "AccessDeniedException", message: "User dummy is not authorized to perform: kms:Decrypt on this resource because the resource owner's account is not active"},
			expected: KMSErrorTypeUserInduced,
		},
		{
			name:     "Other AccessDeniedException",
			err:      &mockAPIError{code: "AccessDeniedException", message: "access denied for some other reason"},
			expected: KMSErrorTypeOther,
		},
		{
			name:     "KMSInternalException with timeout message",
			err:      &mockAPIError{code: (&types.KMSInternalException{}).ErrorCode(), message: "AWS KMS rejected the request because the external key store proxy did not respond in time. Retry the request. If you see this error repeatedly, report it to your external key store proxy administrator"},
			expected: KMSErrorTypeUserInduced,
		},
		{
			name:     "KMSInternalException with external key store proxy communication failure",
			err:      &mockAPIError{code: (&types.KMSInternalException{}).ErrorCode(), message: "AWS KMS cannot communicate with the external key store proxy"},
			expected: KMSErrorTypeUserInduced,
		},
		{
			name:     "KMSInternalException with other message",
			err:      &mockAPIError{code: (&types.KMSInternalException{}).ErrorCode(), message: "Some other internal error"},
			expected: KMSErrorTypeOther,
		},
		{
			name:     "wrapped other error",
			err:      errors.New("wrapped: " + (&mockAPIError{code: (&types.DisabledException{}).ErrorCode()}).Error()),
			expected: KMSErrorTypeOther,
		},
		{
			name:     "context cancelled",
			err:      context.Canceled,
			expected: KMSErrorTypeUserInduced,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseError(tt.err)
			assert.Equal(t, tt.expected, result, "ParseError returned incorrect error type")
		})
	}
}

func TestIsKeyRevoked(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "non-API error",
			err:      errors.New("generic error"),
			expected: false,
		},
		{
			name:     "DisabledException",
			err:      &mockAPIError{code: (&types.DisabledException{}).ErrorCode()},
			expected: true,
		},
		{
			name:     "NotFoundException",
			err:      &mockAPIError{code: (&types.NotFoundException{}).ErrorCode()},
			expected: true,
		},
		{
			name:     "KMSInvalidStateException",
			err:      &mockAPIError{code: (&types.KMSInvalidStateException{}).ErrorCode()},
			expected: false,
		},
		{
			name:     "KMSInvalidStateException with message",
			err:      &mockAPIError{code: (&types.KMSInvalidStateException{}).ErrorCode(), message: "test"},
			expected: false,
		},
		{
			name:     "KMSInvalidStateException with custom key store message",
			err:      &mockAPIError{code: (&types.KMSInvalidStateException{}).ErrorCode(), message: "The custom key store associated with this KMS key is not connected"},
			expected: false,
		},
		{
			name:     "KMSInvalidStateException with external key store proxy message",
			err:      &mockAPIError{code: (&types.KMSInvalidStateException{}).ErrorCode(), message: "AWS KMS cannot communicate with the external key store proxy"},
			expected: false,
		},
		{
			name:     "AccessDeniedException",
			err:      &mockAPIError{code: "AccessDeniedException"},
			expected: false,
		},
		{
			name:     "AccessDeniedException with not authorized message",
			err:      &mockAPIError{code: "AccessDeniedException", message: "User xxx is not authorized to perform: kms:Encrypt on this resource"},
			expected: false,
		},
		{
			name:     "AccessDeniedException with resource control policy message",
			err:      &mockAPIError{code: "AccessDeniedException", message: "User xxx is not authorized to perform: kms:Encrypt on this resource with an explicit deny in a resource control policy"},
			expected: false,
		},
		{
			name:     "AccessDeniedException with account not active message",
			err:      &mockAPIError{code: "AccessDeniedException", message: "User xxx is not authorized to perform: kms:Encrypt because the resource owner's account is not active"},
			expected: false,
		},
		{
			name:     "AccessDeniedException with identity-based policy message",
			err:      &mockAPIError{code: "AccessDeniedException", message: "User xxx is not authorized to perform: kms:Encrypt on this resource with an explicit deny in an identity-based policy"},
			expected: false,
		},
		{
			name:     "AccessDeniedException with does not exist message",
			err:      &mockAPIError{code: "AccessDeniedException", message: "The ciphertext refers to a customer master key that does not exist, does not exist in this region, or you are not allowed to access."},
			expected: false,
		},
		{
			name:     "AccessDeniedException with no resource-based policies allow access message",
			err:      &mockAPIError{code: "AccessDeniedException", message: "User xxx is not authorized to perform: kms:Encrypt on this resource because the resource does not exist in this Region, no resource-based policies allow access, or a resource-based policy explicitly denies access"},
			expected: false,
		},
		{
			name:     "KeyUnavailableException",
			err:      &mockAPIError{code: (&types.KeyUnavailableException{}).ErrorCode()},
			expected: false,
		},
		{
			name:     "KMSInternalException",
			err:      &mockAPIError{code: (&types.KMSInternalException{}).ErrorCode()},
			expected: false,
		},
		{
			name:     "LimitExceededException",
			err:      &mockAPIError{code: (&types.LimitExceededException{}).ErrorCode()},
			expected: false,
		},
		{
			name:     "InvalidCiphertextException",
			err:      &mockAPIError{code: (&types.InvalidCiphertextException{}).ErrorCode()},
			expected: false,
		},
		{
			name:     "InvalidGrantTokenException",
			err:      &mockAPIError{code: (&types.InvalidGrantTokenException{}).ErrorCode()},
			expected: false,
		},
		{
			name:     "wrapped DisabledException",
			err:      fmt.Errorf("failed to encrypt %w", &mockAPIError{code: (&types.DisabledException{}).ErrorCode()}),
			expected: true,
		},
		{
			name:     "context cancelled",
			err:      context.Canceled,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsKeyRevoked(tt.err), "IsKeyRevoked returned incorrect result")
		})
	}
}
