REPO?=gcr.io/must-override
IMAGE?=aws-encryption-provider
TAG?=0.0.1
GOOS?=$(shell go env GOOS)
GOARCH?=$(shell go env GOARCH)

.PHONY: lint test build-docker build-server build-client

lint:
	echo "Verifying go mod tidy"
	hack/verify-mod-tidy.sh
	echo "Verifying gofmt"
	hack/verify-gofmt.sh
	echo "Verifying linting"
	hack/verify-golint.sh

test:
	go test -v -cover -race ./...

build-docker:
	docker buildx build \
		--output=type=docker \
		--platform=linux/$(GOARCH) \
		-t ${REPO}/${IMAGE}:latest \
		-t ${REPO}/${IMAGE}:${TAG} \
		--build-arg TAG=${TAG} .

build-server:
	go build -ldflags \
			"-w -s -X sigs.k8s.io/aws-encryption-provider/pkg/version.Version=${TAG}" \
			-o bin/aws-encryption-provider cmd/server/main.go

build-client:
	go build -ldflags "-w -s" -o bin/grpcclient cmd/client/main.go
	go build -ldflags "-w -s" -o bin/grpcclientv2 cmd/clientv2/main.go

