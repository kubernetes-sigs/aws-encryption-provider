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
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/aws/ratelimit"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	smithymiddleware "github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"
	"go.uber.org/zap"
)

const (
	headerSourceArn     = "x-amz-source-arn"
	headerSourceAccount = "x-amz-source-account"
)

type AWSKMSv2 interface {
	Encrypt(ctx context.Context, params *kms.EncryptInput, optFns ...func(*kms.Options)) (*kms.EncryptOutput, error)
	Decrypt(ctx context.Context, params *kms.DecryptInput, optFns ...func(*kms.Options)) (*kms.DecryptOutput, error)
}

func New(region, kmsEndpoint string, qps, burst, retryTokenCapacity int, sourceArn string) (AWSKMSv2, error) {
	var optFns []func(*config.LoadOptions) error
	if region != "" {
		optFns = append(optFns, config.WithRegion(region))
	}

	switch {
	// Use --retry-token-capacity's value if set, --qps-limit and --burst-limit are deprecated.
	// https://docs.aws.amazon.com/sdk-for-go/v2/developer-guide/configure-retries-timeouts.html (Client-side rate limiting)
	case retryTokenCapacity > 0:
		optFns = append(optFns, config.WithRetryer(func() aws.Retryer {
			return retry.NewStandard(func(o *retry.StandardOptions) {
				o.RateLimiter = ratelimit.NewTokenRateLimit(uint(retryTokenCapacity))
			})
		}))
	case qps > 0:
		zap.L().Info("--qps-limit and --burst-limit are deprecated, use --retry-token-capacity instead")
		if burst <= 0 {
			return nil, fmt.Errorf("burst expected >0, got %d", burst)
		}
		optFns = append(optFns, config.WithRetryer(func() aws.Retryer {
			return retry.NewStandard(func(o *retry.StandardOptions) {
				// Attempt to set a "reasonable" value from the previous intent of --qps-limit and --burst-limit.
				// In aws-sdk-go-v2, client-side rate limits only apply on retries, with varying token cost depending
				// on the type of retry. However, --qps-limit and --burst-limit used to apply to all requests, so set
				// all retry costs to a flat value of 1 until these flags are fully deprecated
				o.RateLimiter = ratelimit.NewTokenRateLimit(uint(qps) + uint(burst))
				o.RetryCost = 1
				o.RetryTimeoutCost = 1
			})
		}))
	}

	cfg, err := config.LoadDefaultConfig(context.Background(), optFns...)
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS config: %w", err)
	}

	err = addConfusedDeputyHeaders(&cfg, sourceArn)
	if err != nil {
		return nil, err
	}

	if cfg.Region == "" {
		ec2 := imds.NewFromConfig(cfg)
		region, err := ec2.GetRegion(context.Background(), &imds.GetRegionInput{})
		if err != nil {
			return nil, fmt.Errorf("failed to call the metadata server's region API, %v", err)
		}
		cfg.Region = region.Region
	}

	var kmsOptFns []func(*kms.Options)
	if kmsEndpoint != "" {
		kmsOptFns = append(kmsOptFns, func(o *kms.Options) {
			o.BaseEndpoint = aws.String(kmsEndpoint)
		})
	}

	client := kms.NewFromConfig(cfg, kmsOptFns...)
	return client, nil
}

func addConfusedDeputyHeaders(cfg *aws.Config, sourceArn string) error {
	if sourceArn != "" {
		sourceAccount, err := getSourceAccount(sourceArn)
		if err != nil {
			return err
		}

		cfg.APIOptions = append(cfg.APIOptions, func(stack *smithymiddleware.Stack) error {
			return stack.Build.Add(smithymiddleware.BuildMiddlewareFunc("KMSConfusedDeputyHeaders", func(
				ctx context.Context, in smithymiddleware.BuildInput, next smithymiddleware.BuildHandler,
			) (smithymiddleware.BuildOutput, smithymiddleware.Metadata, error) {
				req, ok := in.Request.(*smithyhttp.Request)
				if ok {
					req.Header.Set(headerSourceAccount, sourceAccount)
					req.Header.Set(headerSourceArn, sourceArn)
				}
				return next.HandleBuild(ctx, in)
			}), smithymiddleware.Before)
		})

		zap.L().Info("configuring KMS client with confused deputy headers",
			zap.String("sourceArn", sourceArn),
			zap.String("sourceAccount", sourceAccount),
		)
	}
	return nil
}

// getSourceAccount constructs source account and return them for use
func getSourceAccount(sourceArn string) (string, error) {
	// ARN format (https://docs.aws.amazon.com/IAM/latest/UserGuide/reference-arns.html)
	// arn:partition:service:region:account-id:resource-type/resource-id
	// arn:aws:eks:region:account:cluster/cluster-name
	if !arn.IsARN(sourceArn) {
		return "", fmt.Errorf("incorrect ARN format for source arn: %s", sourceArn)
	}

	parsedArn, err := arn.Parse(sourceArn)
	if err != nil {
		return "", err
	}

	return parsedArn.AccountID, nil
}
