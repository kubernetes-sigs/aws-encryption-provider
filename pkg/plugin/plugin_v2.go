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
	pb "k8s.io/kms/apis/v2"
	"sigs.k8s.io/aws-encryption-provider/pkg/kmsplugin"
)

var _ pb.KeyManagementServiceServer = &V2Plugin{}

const (
	GRPC_V2 = "v2"
)

// Plugin implements the KeyManagementServiceServer
type V2Plugin struct {
	svc           kmsiface.KMSAPI
	keyID         string
	encryptionCtx map[string]*string
	healthCheck   *SharedHealthCheck
}

// New returns a new *V2Plugin
func NewV2(key string, svc kmsiface.KMSAPI, encryptionCtx map[string]string, healthCheck *SharedHealthCheck) *V2Plugin {
	return newPluginV2(
		key,
		svc,
		encryptionCtx,
		healthCheck,
	)
}

func newPluginV2(
	key string,
	svc kmsiface.KMSAPI,
	encryptionCtx map[string]string,
	healthCheck *SharedHealthCheck,
) *V2Plugin {
	p := &V2Plugin{
		svc:         svc,
		keyID:       key,
		healthCheck: healthCheck,
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
//  1. not incur extra KMS API call if V2Plugin "Encrypt" method has already
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
func (p *V2Plugin) Health() error {
	recent, err := p.healthCheck.isRecentlyChecked()
	if !recent {
		_, err = p.Encrypt(context.Background(), &pb.EncryptRequest{Plaintext: []byte("foo")})
		p.healthCheck.recordErr(err)
		if err != nil {
			zap.L().Warn("health check failed", zap.Error(err))
		}
		return err
	}
	if err != nil {
		zap.L().Warn("cached health check failed", zap.Error(err))
	} else {
		zap.L().Debug("health check success")
	}
	return err
}

func (p *V2Plugin) Live() error {
	if err := p.Health(); err != nil && kmsplugin.ParseError(err) != kmsplugin.KMSErrorTypeUserInduced {
		return err
	}
	return nil
}

// Status returns the V2Plugin server status
func (p *V2Plugin) Status(ctx context.Context, request *pb.StatusRequest) (*pb.StatusResponse, error) {
	status := "ok"
	if p.Health() != nil {
		status = "err"
	}
	return &pb.StatusResponse{
		Version: "v2beta1",
		Healthz: status,
		KeyId:   p.keyID,
	}, nil
}

// Encrypt executes the encryption operation using AWS KMS
func (p *V2Plugin) Encrypt(ctx context.Context, request *pb.EncryptRequest) (*pb.EncryptResponse, error) {
	zap.L().Debug("starting encrypt operation")

	startTime := time.Now()
	input := &kms.EncryptInput{
		Plaintext: request.Plaintext,
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
		kmsLatencyMetric.WithLabelValues(p.keyID, failLabel, kmsplugin.OperationEncrypt, GRPC_V2).Observe(kmsplugin.GetMillisecondsSince(startTime))
		kmsOperationCounter.WithLabelValues(p.keyID, failLabel, kmsplugin.OperationEncrypt, GRPC_V2).Inc()
		return nil, fmt.Errorf("failed to encrypt %w", err)
	}

	zap.L().Debug("encrypt operation successful")
	kmsLatencyMetric.WithLabelValues(p.keyID, kmsplugin.StatusSuccess, kmsplugin.OperationEncrypt, GRPC_V2).Observe(kmsplugin.GetMillisecondsSince(startTime))
	kmsOperationCounter.WithLabelValues(p.keyID, kmsplugin.StatusSuccess, kmsplugin.OperationEncrypt, GRPC_V2).Inc()
	return &pb.EncryptResponse{
		Ciphertext: append([]byte(kmsplugin.KMSStorageVersionV2), result.CiphertextBlob...),
		KeyId:      p.keyID,
	}, nil
}

// Decrypt executes the decrypt operation using AWS KMS
func (p *V2Plugin) Decrypt(ctx context.Context, request *pb.DecryptRequest) (*pb.DecryptResponse, error) {
	zap.L().Debug("starting decrypt operation")

	startTime := time.Now()
	storageVersion := kmsplugin.KMSStorageVersion(request.Ciphertext[0])
	switch storageVersion {
	case kmsplugin.KMSStorageVersionV2:
		request.Ciphertext = request.Ciphertext[1:]
	default:
		// enforce the kmsplugin.StorageVersion in v2
		return nil, fmt.Errorf("version %s in Ciphertext doesn't match kmsplugin", storageVersion)
	}
	input := &kms.DecryptInput{
		CiphertextBlob: request.Ciphertext,
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
		kmsLatencyMetric.WithLabelValues(p.keyID, failLabel, kmsplugin.OperationDecrypt, GRPC_V2).Observe(kmsplugin.GetMillisecondsSince(startTime))
		kmsOperationCounter.WithLabelValues(p.keyID, failLabel, kmsplugin.OperationDecrypt, GRPC_V2).Inc()
		return nil, fmt.Errorf("failed to decrypt %w", err)
	}

	zap.L().Debug("decrypt operation successful")
	kmsLatencyMetric.WithLabelValues(p.keyID, kmsplugin.StatusSuccess, kmsplugin.OperationDecrypt, GRPC_V2).Observe(kmsplugin.GetMillisecondsSince(startTime))
	kmsOperationCounter.WithLabelValues(p.keyID, kmsplugin.StatusSuccess, kmsplugin.OperationDecrypt, GRPC_V2).Inc()
	return &pb.DecryptResponse{Plaintext: result.Plaintext}, nil
}

// Register registers the V2Plugin with the grpc server
func (p *V2Plugin) Register(s *grpc.Server) {
	zap.L().Info("registering the kmsplugin plugin with grpc server")
	pb.RegisterKeyManagementServiceServer(s, p)
}
