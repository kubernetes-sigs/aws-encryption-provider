package kmsplugin

import (
	"errors"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	kmstypes "github.com/aws/aws-sdk-go-v2/service/kms/types"
	"github.com/aws/smithy-go"
	"go.uber.org/zap"
)

type KMSErrorType int

const (
	KMSErrorTypeNil = KMSErrorType(iota)
	KMSErrorTypeUserInduced
	KMSErrorTypeThrottled
	KMSErrorTypeOther
)

func (t KMSErrorType) String() string {
	switch t {
	case KMSErrorTypeNil:
		return ""
	case KMSErrorTypeUserInduced:
		return "user-induced"
	case KMSErrorTypeThrottled:
		return "throttled"
	case KMSErrorTypeOther:
		return "other"
	default:
		return ""
	}
}

// ParseError parses error codes from KMS
// ref. https://docs.aws.amazon.com/kms/latest/developerguide/key-state.html
// ref. https://docs.aws.amazon.com/sdk-for-go/api/service/kms/
func ParseError(err error) (errorType KMSErrorType) {
	if err == nil {
		return KMSErrorTypeNil
	}

	uerr := errors.Unwrap(err)
	if uerr == nil {
		// in case the error was not wrapped,
		// preserve the original error type
		uerr = err
	}

	var ae smithy.APIError
	if !errors.As(uerr, &ae) {
		return KMSErrorTypeOther
	}

	zap.L().Debug("parsed error", zap.String("code", ae.ErrorCode()), zap.String("message", ae.ErrorMessage()))
	var defaultCodes retry.IsErrorThrottles = retry.DefaultThrottles
	if defaultCodes.IsErrorThrottle(uerr) == aws.TrueTernary {
		return KMSErrorTypeThrottled
	}
	switch ae.ErrorCode() {
	// CMK is disabled or pending deletion
	case (&kmstypes.DisabledException{}).ErrorCode(),
		(&kmstypes.KMSInvalidStateException{}).ErrorCode():
		return KMSErrorTypeUserInduced

	// CMK does not exist, or grant is not valid
	case (&kmstypes.KeyUnavailableException{}).ErrorCode(),
		(&kmstypes.InvalidArnException{}).ErrorCode(),
		(&kmstypes.InvalidGrantIdException{}).ErrorCode(),
		(&kmstypes.InvalidGrantTokenException{}).ErrorCode():
		return KMSErrorTypeUserInduced

	// ref. https://docs.aws.amazon.com/kms/latest/developerguide/requests-per-second.html
	case (&kmstypes.LimitExceededException{}).ErrorCode():
		return KMSErrorTypeThrottled

	// AWS SDK Go for KMS does not "yet" define specific error code for a case where a customer specifies the deleted key
	// "AccessDeniedException" error code may be returned when (1) CMK does not exist (not pending delete),
	// or (2) corresponding IAM role is not allowed to access the key.
	// Thus we only want to mark "AccessDeniedException" as user-induced for the case (1).
	// e.g., "AccessDeniedException: The ciphertext refers to a customer master key that does not exist, does not exist in this region, or you are not allowed to access."
	// KMS service may change the error message, so we do the string match.
	case "AccessDeniedException":
		if strings.Contains(ae.ErrorMessage(), "customer master key that does not exist") ||
			strings.Contains(ae.ErrorMessage(), "does not exist in this region") {
			return KMSErrorTypeUserInduced
		}
	// Sometimes this error message is returned as part of KMSInvalidStateException or KMSInternalException
	case (&kmstypes.KMSInternalException{}).ErrorCode():
		if strings.Contains(ae.ErrorMessage(), "AWS KMS rejected the request because the external key store proxy did not respond in time. Retry the request. If you see this error repeatedly, report it to your external key store proxy administrator") {
			return KMSErrorTypeUserInduced
		}
	}

	return KMSErrorTypeOther
}

const (
	StatusSuccess         = "success"
	StatusFailure         = "failure"
	StatusFailureThrottle = "failure-throttle"
	OperationEncrypt      = "encrypt"
	OperationDecrypt      = "decrypt"
)

// StorageVersion is a prefix used for versioning encrypted content
const StorageVersion = "1"

type KMSStorageVersion string

const (
	KMSStorageVersionV2 KMSStorageVersion = "1"
)

// TODO: make configurable
const (
	DefaultHealthCheckPeriod = 30 * time.Second
	DefaultErrcBufSize       = 100
)

func GetMillisecondsSince(startTime time.Time) float64 {
	return float64(time.Since(startTime).Milliseconds())
}

func GetStatusLabel(err error) string {
	var defaultCodes retry.IsErrorThrottles = retry.DefaultThrottles
	switch {
	case err == nil:
		return StatusSuccess
	case defaultCodes.IsErrorThrottle(err) == aws.TrueTernary:
		return StatusFailureThrottle
	default:
		return StatusFailure
	}
}
