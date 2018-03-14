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

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/kms/kmsiface"
	"github.com/kubernetes-sigs/aws_encryption-provider/version"
	"google.golang.org/grpc"
	pb "k8s.io/apiserver/pkg/storage/value/encrypt/envelope/v1beta1"
)

type plugin struct {
	svc   kmsiface.KMSAPI
	keyId string
}

func New(key string, svc kmsiface.KMSAPI) *plugin {
	return &plugin{
		svc:   svc,
		keyId: key,
	}
}

func (p *plugin) Version(ctx context.Context, request *pb.VersionRequest) (*pb.VersionResponse, error) {
	return &pb.VersionResponse{
		Version:        version.APIVersion,
		RuntimeName:    version.Runtime,
		RuntimeVersion: version.Version,
	}, nil
}

func (p *plugin) Encrypt(ctx context.Context, request *pb.EncryptRequest) (*pb.EncryptResponse, error) {
	input := &kms.EncryptInput{
		Plaintext: request.Plain,
		KeyId:     aws.String(p.keyId),
	}

	result, err := p.svc.Encrypt(input)
	if err != nil {
		return nil, fmt.Errorf("Failed to encrypt data: %v", err)
	}

	return &pb.EncryptResponse{Cipher: result.CiphertextBlob}, nil
}

func (p *plugin) Decrypt(ctx context.Context, request *pb.DecryptRequest) (*pb.DecryptResponse, error) {
	input := &kms.DecryptInput{
		CiphertextBlob: request.Cipher,
	}

	result, err := p.svc.Decrypt(input)
	if err != nil {
		return nil, fmt.Errorf("Failed to decrypt data: %v", err)
	}

	return &pb.DecryptResponse{Plain: result.Plaintext}, nil
}

func (p *plugin) Register(s *grpc.Server) {
	pb.RegisterKeyManagementServiceServer(s, p)
}
