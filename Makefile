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
	hack/run-test.sh

build-docker:
	docker buildx build \
		--output=type=docker \
		--platform=linux/$(GOARCH) \
		-t ${REPO}/${IMAGE}:latest \
		-t ${REPO}/${IMAGE}:${TAG} \
		--build-arg BUILDER=$(shell hack/setup-go.sh) \
		--build-arg TAG=${TAG} .

build-server:
	TAG=${TAG} hack/build-server.sh

build-client:
	TAG=${TAG} hack/build-client.sh


