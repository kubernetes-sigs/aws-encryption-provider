package kmsplugin

import (
	"context"
	"errors"
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
