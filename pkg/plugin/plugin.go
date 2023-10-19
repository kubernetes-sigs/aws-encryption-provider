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
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/kms/kmsiface"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	pb "k8s.io/kms/apis/v1beta1"
	"sigs.k8s.io/aws-encryption-provider/pkg/kmsplugin"
	"sigs.k8s.io/aws-encryption-provider/pkg/version"
)

var _ pb.KeyManagementServiceServer = &V1Plugin{}

const (
	GRPC_V1 = "v1"
)

// Plugin implements the KeyManagementServiceServer
type V1Plugin struct {
	svc           kmsiface.KMSAPI
	keyID         string
	encryptionCtx map[string]*string
	healthCheck   *SharedHealthCheck
}

// New returns a new *V1Plugin
func New(key string, svc kmsiface.KMSAPI, encryptionCtx map[string]string, healthCheck *SharedHealthCheck) *V1Plugin {
	return newPlugin(
		key,
		svc,
		encryptionCtx,
		healthCheck,
	)
}

func newPlugin(
	key string,
	svc kmsiface.KMSAPI,
	encryptionCtx map[string]string,
	sharedHealthCheck *SharedHealthCheck,
) *V1Plugin {
	p := &V1Plugin{
		svc:         svc,
		keyID:       key,
		healthCheck: sharedHealthCheck,
	}
	if len(encryptionCtx) > 0 {
		p.encryptionCtx = make(map[string]*string)
	}
	for k, v := range encryptionCtx {
		p.encryptionCtx[k] = aws.String(v)
	}
	return p
}

// Health checks KMS API availability.
//
// The goal is to:
//  1. not incur extra KMS API call if V1Plugin "Encrypt" method has already
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
func (p *V1Plugin) Health() error {
	recent, err := p.healthCheck.isRecentlyChecked()
	if !recent {
		_, err = p.Encrypt(context.Background(), &pb.EncryptRequest{Plain: []byte("foo")})
		p.healthCheck.recordErr(err)
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
func (p *V1Plugin) Live() error {
	if err := p.Health(); err != nil && kmsplugin.ParseError(err) != kmsplugin.KMSErrorTypeUserInduced {
		return err
	}
	return nil
}

// Version returns the V1Plugin server version
func (p *V1Plugin) Version(ctx context.Context, request *pb.VersionRequest) (*pb.VersionResponse, error) {
	return &pb.VersionResponse{
		Version:        version.APIVersion,
		RuntimeName:    version.Runtime,
		RuntimeVersion: version.Version,
	}, nil
}

// Encrypt executes the encryption operation using AWS KMS
func (p *V1Plugin) Encrypt(ctx context.Context, request *pb.EncryptRequest) (*pb.EncryptResponse, error) {
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
		case p.healthCheck.healthCheckErrc <- err:
		default:
		}
		zap.L().Error("request to encrypt failed", zap.String("error-type", kmsplugin.ParseError(err).String()), zap.Error(err))
		failLabel := kmsplugin.GetStatusLabel(err)
		kmsLatencyMetric.WithLabelValues(p.keyID, failLabel, kmsplugin.OperationEncrypt, GRPC_V1).Observe(kmsplugin.GetMillisecondsSince(startTime))
		kmsOperationCounter.WithLabelValues(p.keyID, failLabel, kmsplugin.OperationEncrypt, GRPC_V1).Inc()
		return nil, fmt.Errorf("failed to encrypt %w", err)
	}

	zap.L().Debug("encrypt operation successful")
	kmsLatencyMetric.WithLabelValues(p.keyID, kmsplugin.StatusSuccess, kmsplugin.OperationEncrypt, GRPC_V1).Observe(kmsplugin.GetMillisecondsSince(startTime))
	kmsOperationCounter.WithLabelValues(p.keyID, kmsplugin.StatusSuccess, kmsplugin.OperationEncrypt, GRPC_V1).Inc()
	return &pb.EncryptResponse{Cipher: append([]byte(kmsplugin.StorageVersion), result.CiphertextBlob...)}, nil
}

// Decrypt executes the decrypt operation using AWS KMS
func (p *V1Plugin) Decrypt(ctx context.Context, request *pb.DecryptRequest) (*pb.DecryptResponse, error) {
	zap.L().Debug("starting decrypt operation")

	startTime := time.Now()
	if string(request.Cipher[0]) == kmsplugin.StorageVersion {
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
		case p.healthCheck.healthCheckErrc <- err:
		default:
		}
		zap.L().Error("request to decrypt failed", zap.String("error-type", kmsplugin.ParseError(err).String()), zap.Error(err))
		failLabel := kmsplugin.GetStatusLabel(err)
		kmsLatencyMetric.WithLabelValues(p.keyID, failLabel, kmsplugin.OperationDecrypt, GRPC_V1).Observe(kmsplugin.GetMillisecondsSince(startTime))
		kmsOperationCounter.WithLabelValues(p.keyID, failLabel, kmsplugin.OperationDecrypt, GRPC_V1).Inc()
		return nil, fmt.Errorf("failed to decrypt %w", err)
	}

	zap.L().Debug("decrypt operation successful")
	kmsLatencyMetric.WithLabelValues(p.keyID, kmsplugin.StatusSuccess, kmsplugin.OperationDecrypt, GRPC_V1).Observe(kmsplugin.GetMillisecondsSince(startTime))
	kmsOperationCounter.WithLabelValues(p.keyID, kmsplugin.StatusSuccess, kmsplugin.OperationDecrypt, GRPC_V1).Inc()
	return &pb.DecryptResponse{Plain: result.Plaintext}, nil
}

// Register registers the V1Plugin with the grpc server
func (p *V1Plugin) Register(s *grpc.Server) {
	zap.L().Info("registering the kmsplugin plugin with grpc server")
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
