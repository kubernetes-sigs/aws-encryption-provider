# Copyright 2018 The Kubernetes Authors.
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
# http://www.apache.org/licenses/LICENSE-2.0
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM golang:1.12-alpine AS build
RUN apk --no-cache update && \
    apk --no-cache add ca-certificates git && \
    rm -rf /var/cache/apk/*
WORKDIR /go/src/github.com/kubernetes-sigs/aws-encryption-provider
ARG TAG
COPY . ./
RUN	CGO_ENABLED=0 GOOS=linux go build -installsuffix cgo -ldflags \
    "-X github.com/kubernetes-sigs/aws-encryption-provider/pkg/version.Version=$TAG" \
    -o bin/aws-encryption-provider cmd/server/main.go

FROM scratch AS aws-encryption-provider
COPY --from=build /etc/ssl/certs/ /etc/ssl/certs/
COPY --from=build /go/src/github.com/kubernetes-sigs/aws-encryption-provider/bin/aws-encryption-provider /aws-encryption-provider
ENTRYPOINT ["/aws-encryption-provider"]
