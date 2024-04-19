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
ARG BUILDER=public.ecr.aws/eks-distro-build-tooling/golang:1.22.2-gcc
ARG BASE_IMAGE=public.ecr.aws/eks-distro/kubernetes/go-runner:v0.16.4-eks-1-30-latest

FROM --platform=$BUILDPLATFORM ${BUILDER} AS build
WORKDIR /go/src/sigs.k8s.io/aws-encryption-provider
ARG TAG
COPY . ./
ENV GO111MODULE=on
ARG TARGETOS TARGETARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags \
    "-w -s -X sigs.k8s.io/aws-encryption-provider/pkg/version.Version=$TAG" \
    -o bin/aws-encryption-provider cmd/server/main.go

FROM --platform=$TARGETPLATFORM ${BASE_IMAGE}
COPY --from=build /go/src/sigs.k8s.io/aws-encryption-provider/bin/aws-encryption-provider /aws-encryption-provider
ENTRYPOINT ["/go-runner"]
CMD ["/aws-encryption-provider"]
