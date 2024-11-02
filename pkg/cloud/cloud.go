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
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/ratelimit"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/kms"
)

type AWSKMSv2 interface {
	Encrypt(ctx context.Context, params *kms.EncryptInput, optFns ...func(*kms.Options)) (*kms.EncryptOutput, error)
	Decrypt(ctx context.Context, params *kms.DecryptInput, optFns ...func(*kms.Options)) (*kms.DecryptOutput, error)
}

func New(region, kmsEndpoint string, qps, burst int) (AWSKMSv2, error) {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to create new session: %w", err)
	}

	if qps > 0 {
		cfg, err = config.LoadDefaultConfig(context.Background(), config.WithRetryer(func() aws.Retryer {
			return retry.NewStandard(func(o *retry.StandardOptions) {
				o.RateLimiter = ratelimit.NewTokenRateLimit(uint(qps) * uint(burst))
			})
		}))
		if err != nil {
			return nil, fmt.Errorf("failed to create new session: %w", err)
		}
	}

	client := kms.NewFromConfig(cfg, func(o *kms.Options) {
		o.Region = region
		o.BaseEndpoint = aws.String(kmsEndpoint)
	})
	return client, nil
}
