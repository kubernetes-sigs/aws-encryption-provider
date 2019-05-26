REPO?=gcr.io/must-override
IMAGE?=aws-encryption-provider
TAG?=0.0.1

.PHONY: lint test build-docker build-server build-client

lint:
	hack/verify-golint.sh

test:
	go test ./...

build-docker:
	docker build \
		-t ${REPO}/${IMAGE}:latest \
		-t ${REPO}/${IMAGE}:${TAG} \
		--build-arg TAG=${TAG} .

build-server:
	go build -ldflags \
			"-X github.com/kubernetes-sigs/aws-encryption-provider/pkg/version.Version=${TAG}" \
			-o bin/grpcserver cmd/server/main.go

build-client:
	go build -o bin/grpcclient cmd/client/main.go
