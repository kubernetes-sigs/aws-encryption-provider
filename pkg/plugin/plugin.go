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
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	//nolint
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	awsreq "github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/kms/kmsiface"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	pb "k8s.io/apiserver/pkg/storage/value/encrypt/envelope/v1beta1"
	"sigs.k8s.io/aws-encryption-provider/pkg/version"
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

	ev, ok := uerr.(awserr.Error)
	if !ok {
		return KMSErrorTypeOther
	}
	zap.L().Debug("parsed error", zap.String("code", ev.Code()), zap.String("message", ev.Message()))
	if request.IsErrorThrottle(uerr) {
		return KMSErrorTypeThrottled
	}
	switch ev.Code() {
	// CMK is disabled or pending deletion
	case kms.ErrCodeDisabledException,
		kms.ErrCodeInvalidStateException:
		return KMSErrorTypeUserInduced

	// CMK does not exist, or grant is not valid
	case kms.ErrCodeKeyUnavailableException,
		kms.ErrCodeInvalidArnException,
		kms.ErrCodeInvalidGrantIdException,
		kms.ErrCodeInvalidGrantTokenException:
		return KMSErrorTypeUserInduced

	// ref. https://docs.aws.amazon.com/kms/latest/developerguide/requests-per-second.html
	case kms.ErrCodeLimitExceededException:
		return KMSErrorTypeThrottled
	}
	return KMSErrorTypeOther
}

const (
	statusSuccess         = "success"
	statusFailure         = "failure"
	statusFailureThrottle = "failure-throttle"
	operationEncrypt      = "encrypt"
	operationDecrypt      = "decrypt"
)

// StorageVersion is a prefix used for versioning encrypted content
const StorageVersion = "1"

var _ pb.KeyManagementServiceServer = &Plugin{}

// Plugin implements the KeyManagementServiceServer
type Plugin struct {
	svc           kmsiface.KMSAPI
	keyID         string
	encryptionCtx map[string]*string

	lastMu  sync.RWMutex
	lastErr error
	lastTs  time.Time

	healthCheckPeriod         time.Duration
	healthCheckErrc           chan error
	healthCheckStopcCloseOnce *sync.Once
	healthCheckStopc          chan struct{}
	healthCheckClosed         chan struct{}
}

// TODO: make configurable
const (
	defaultHealthCheckPeriod = 30 * time.Second
	defaultErrcBufSize       = 100
)

// New returns a new *Plugin
func New(key string, svc kmsiface.KMSAPI, encryptionCtx map[string]string) *Plugin {
	return newPlugin(
		key,
		svc,
		encryptionCtx,
		defaultHealthCheckPeriod,
		defaultErrcBufSize,
	)
}

func newPlugin(
	key string,
	svc kmsiface.KMSAPI,
	encryptionCtx map[string]string,
	checkPeriod time.Duration,
	errcBuf int,
) *Plugin {
	p := &Plugin{
		svc:                       svc,
		keyID:                     key,
		healthCheckPeriod:         checkPeriod,
		healthCheckErrc:           make(chan error, errcBuf),
		healthCheckStopcCloseOnce: new(sync.Once),
		healthCheckStopc:          make(chan struct{}),
		healthCheckClosed:         make(chan struct{}),
	}
	if len(encryptionCtx) > 0 {
		p.encryptionCtx = make(map[string]*string)
	}
	for k, v := range encryptionCtx {
		p.encryptionCtx[k] = aws.String(v)
	}
	go p.startCheckHealth()
	return p
}

func (p *Plugin) startCheckHealth() {
	zap.L().Info("starting health check routine", zap.String("period", p.healthCheckPeriod.String()))
	for {
		select {
		case <-p.healthCheckStopc:
			zap.L().Warn("exiting health check routine")
			p.healthCheckClosed <- struct{}{}
			return
		case err := <-p.healthCheckErrc:
			p.recordErr(err)
		}
	}
}

func (p *Plugin) stopCheckHealth() {
	p.healthCheckStopcCloseOnce.Do(func() {
		close(p.healthCheckStopc)
		<-p.healthCheckClosed
	})
}

func (p *Plugin) isRecentlyChecked() (bool, error) {
	p.lastMu.RLock()
	err, ts := p.lastErr, p.lastTs
	never, latest := err == nil && ts.IsZero(), time.Since(ts) < p.healthCheckPeriod
	p.lastMu.RUnlock()
	return !never && latest, err
}

func (p *Plugin) recordErr(err error) {
	p.lastMu.Lock()
	p.lastErr, p.lastTs = err, time.Now()
	p.lastMu.Unlock()
}

// Health checks KMS API availability.
//
// The goal is to:
//  1. not incur extra KMS API call if plugin "Encrypt" method has already
//  2. return latest health status (cached KMS status must reflect the current)
//
// The error is sent via channel and consumed by goroutine.
// The error channel may be full and block, when there are too many failures.
// The error channel may be empty and block, when there's no failure.
// To handle those two cases, keep track latest health check timestamps.
//
// Call KMS "Encrypt" API call iff:
//  1. there was never a health check done
//  2. there was no health check done for the last "healthCheckPeriod"
//     (only use the cached error if the error is from recent API call)
func (p *Plugin) Health() error {
	recent, err := p.isRecentlyChecked()
	if !recent {
		_, err = p.Encrypt(context.Background(), &pb.EncryptRequest{Plain: []byte("foo")})
		p.recordErr(err)
		if err != nil {
			zap.L().Warn("health check failed", zap.Error(err))
		}
		return err
	}
	if err != nil {
		zap.L().Warn("health check failed", zap.Error(err))
	} else {
		zap.L().Debug("health check success")
	}
	return err
}

// Live checks the liveness of KMS API.
// If the error is user-induced (e.g., revoke CMK), the function returns NO error.
// If the error is due to KMS availability, the function returns the error.
func (p *Plugin) Live() error {
	if err := p.Health(); err != nil && ParseError(err) != KMSErrorTypeUserInduced {
		return err
	}
	return nil
}

// Version returns the plugin server version
func (p *Plugin) Version(ctx context.Context, request *pb.VersionRequest) (*pb.VersionResponse, error) {
	return &pb.VersionResponse{
		Version:        version.APIVersion,
		RuntimeName:    version.Runtime,
		RuntimeVersion: version.Version,
	}, nil
}

// Encrypt executes the encryption operation using AWS KMS
func (p *Plugin) Encrypt(ctx context.Context, request *pb.EncryptRequest) (*pb.EncryptResponse, error) {
	zap.L().Debug("starting encrypt operation")

	startTime := time.Now()
	input := &kms.EncryptInput{
		Plaintext: request.Plain,
		KeyId:     aws.String(p.keyID),
	}
	if len(p.encryptionCtx) > 0 {
		zap.L().Debug("configuring encryption context", zap.String("ctx", fmt.Sprintf("%v", p.encryptionCtx)))
		input.EncryptionContext = p.encryptionCtx
	}

	result, err := p.svc.Encrypt(input)
	if err != nil {
		select {
		case p.healthCheckErrc <- err:
		default:
		}
		zap.L().Error("request to encrypt failed", zap.String("error-type", ParseError(err).String()), zap.Error(err))
		failLabel := getStatusLabel(err)
		kmsLatencyMetric.WithLabelValues(p.keyID, failLabel, operationEncrypt).Observe(getMillisecondsSince(startTime))
		kmsOperationCounter.WithLabelValues(p.keyID, failLabel, operationEncrypt).Inc()
		return nil, fmt.Errorf("failed to encrypt %w", err)
	}

	zap.L().Debug("encrypt operation successful")
	kmsLatencyMetric.WithLabelValues(p.keyID, statusSuccess, operationEncrypt).Observe(getMillisecondsSince(startTime))
	kmsOperationCounter.WithLabelValues(p.keyID, statusSuccess, operationEncrypt).Inc()
	return &pb.EncryptResponse{Cipher: append([]byte(StorageVersion), result.CiphertextBlob...)}, nil
}

// Decrypt executes the decrypt operation using AWS KMS
func (p *Plugin) Decrypt(ctx context.Context, request *pb.DecryptRequest) (*pb.DecryptResponse, error) {
	zap.L().Debug("starting decrypt operation")

	startTime := time.Now()
	if string(request.Cipher[0]) == StorageVersion {
		request.Cipher = request.Cipher[1:]
	}
	input := &kms.DecryptInput{
		CiphertextBlob: request.Cipher,
	}
	if len(p.encryptionCtx) > 0 {
		zap.L().Debug("configuring encryption context", zap.String("ctx", fmt.Sprintf("%v", p.encryptionCtx)))
		input.EncryptionContext = p.encryptionCtx
	}

	result, err := p.svc.Decrypt(input)
	if err != nil {
		select {
		case p.healthCheckErrc <- err:
		default:
		}
		zap.L().Error("request to decrypt failed", zap.String("error-type", ParseError(err).String()), zap.Error(err))
		failLabel := getStatusLabel(err)
		kmsLatencyMetric.WithLabelValues(p.keyID, failLabel, operationDecrypt).Observe(getMillisecondsSince(startTime))
		kmsOperationCounter.WithLabelValues(p.keyID, failLabel, operationDecrypt).Inc()
		return nil, fmt.Errorf("failed to decrypt %w", err)
	}

	zap.L().Debug("decrypt operation successful")
	kmsLatencyMetric.WithLabelValues(p.keyID, statusSuccess, operationDecrypt).Observe(getMillisecondsSince(startTime))
	kmsOperationCounter.WithLabelValues(p.keyID, statusSuccess, operationDecrypt).Inc()
	return &pb.DecryptResponse{Plain: result.Plaintext}, nil
}

// Register registers the plugin with the grpc server
func (p *Plugin) Register(s *grpc.Server) {
	zap.L().Info("registering the kms plugin with grpc server")
	pb.RegisterKeyManagementServiceServer(s, p)
}

// WaitForReady uses a given client to wait until the given duration for the
// server to become ready
func WaitForReady(client pb.KeyManagementServiceClient, duration time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	_, err := client.Version(ctx, &pb.VersionRequest{}, grpc.WaitForReady(true))
	if err != nil {
		return err
	}

	return nil
}

// NewClient returns a KeyManagementServiceClient for a given grpc connection
func NewClient(conn *grpc.ClientConn) pb.KeyManagementServiceClient {
	return pb.NewKeyManagementServiceClient(conn)
}

func getMillisecondsSince(startTime time.Time) float64 {
	return time.Since(startTime).Seconds() * 1000
}

func getStatusLabel(err error) string {
	switch {
	case err == nil:
		return statusSuccess
	case awsreq.IsErrorThrottle(err):
		return statusFailureThrottle
	default:
		return statusFailure
	}
}
