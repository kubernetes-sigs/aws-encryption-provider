REPO?=gcr.io/must-override
IMAGE?=aws-encryption-provider
TAG?=0.0.1

.PHONY: lint test build-docker build-server build-client

lint:
	echo "Verifying vendored dependencies"
	hack/verify-vendor.sh
	echo "Verifying linting"
	hack/verify-golint.sh

test:
	go test -v -cover -race ./...

build-docker:
	docker build \
		-t ${REPO}/${IMAGE}:latest \
		-t ${REPO}/${IMAGE}:${TAG} \
		--build-arg TAG=${TAG} .

build-server:
	go build -ldflags \
			"-X sigs.k8s.io/aws-encryption-provider/pkg/version.Version=${TAG}" \
			-o bin/grpcserver cmd/server/main.go

build-client:
	go build -o bin/grpcclient cmd/client/main.go
