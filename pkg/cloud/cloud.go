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
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/kms/kmsiface"
	"sigs.k8s.io/aws-encryption-provider/pkg/httputil"
)

type AWSKMS struct {
	kmsiface.KMSAPI
}

func New(region, kmsEndpoint string, qps, burst int) (*AWSKMS, error) {
	cfg := &aws.Config{
		Region:                        aws.String(region),
		CredentialsChainVerboseErrors: aws.Bool(true),
		Endpoint:                      aws.String(kmsEndpoint),
	}

	sess, err := session.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create new session: %v", err)
	}

	if qps > 0 {
		var err error
		sess.Config.HTTPClient, err = httputil.NewRateLimitedClient(
			qps,
			burst,
		)
		if err != nil {
			return nil, err
		}
	}

	return &AWSKMS{
		kms.New(sess, cfg),
	}, nil
}
