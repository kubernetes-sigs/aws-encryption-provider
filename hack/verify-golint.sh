#!/bin/bash

# Copyright 2018 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -euo pipefail

source hack/setup-go.sh

go version

if ! which golangci-lint > /dev/null; then
    echo "Cannot find golangci-lint. Installing golangci-lint..."
    GO111MODULE=on go install -v github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.0.2
fi

$(go env GOPATH)/bin/golangci-lint run --timeout=10m --config=.golangci.yml

echo "Congratulations! All Go source files have been linted."
