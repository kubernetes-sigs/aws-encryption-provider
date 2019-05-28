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

package plugin

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/kms/kmsiface"
	"github.com/kubernetes-sigs/aws-encryption-provider/pkg/version"
	"google.golang.org/grpc"
	pb "k8s.io/apiserver/pkg/storage/value/encrypt/envelope/v1beta1"
)

// StorageVersion is a prefix used for versioning encrypted content
const StorageVersion = "1"

var _ pb.KeyManagementServiceServer = &Plugin{}

// Plugin implements the KeyManagementServiceServer
type Plugin struct {
	svc   kmsiface.KMSAPI
	keyID string
}

// New returns a new *Plugin
func New(key string, svc kmsiface.KMSAPI) *Plugin {
	return &Plugin{
		svc:   svc,
		keyID: key,
	}
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
	input := &kms.EncryptInput{
		Plaintext: request.Plain,
		KeyId:     aws.String(p.keyID),
	}

	result, err := p.svc.Encrypt(input)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt data: %v", err)
	}

	return &pb.EncryptResponse{Cipher: append([]byte(StorageVersion), result.CiphertextBlob...)}, nil
}

// Decrypt executes the decrypt operation using AWS KMS
func (p *Plugin) Decrypt(ctx context.Context, request *pb.DecryptRequest) (*pb.DecryptResponse, error) {
	if string(request.Cipher[0]) == StorageVersion {
		request.Cipher = request.Cipher[1:]
	}
	input := &kms.DecryptInput{
		CiphertextBlob: request.Cipher,
	}

	result, err := p.svc.Decrypt(input)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt data: %v", err)
	}

	return &pb.DecryptResponse{Plain: result.Plaintext}, nil
}

// Register registers the plugin with the grpc server
func (p *Plugin) Register(s *grpc.Server) {
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

// Check validates the availability of the server using the provided client
func Check(client pb.KeyManagementServiceClient) (string, error) {
	res, err := client.Version(context.Background(), &pb.VersionRequest{})
	if err != nil {
		return "", err
	}

	return res.String(), nil
}

// NewClient returns a KeyManagementServiceClient for a given grpc connection
func NewClient(conn *grpc.ClientConn) pb.KeyManagementServiceClient {
	return pb.NewKeyManagementServiceClient(conn)
}
