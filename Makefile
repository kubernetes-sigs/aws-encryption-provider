REPO?=gcr.io/must-override
IMAGE?=aws-encryption-provider
TAG?=0.0.1

.PHONY: lint test build-docker build-server build-client

lint:
	echo "Verifying go mod tidy"
	hack/verify-mod-tidy.sh
	echo "Verifying gofmt"
	hack/verify-gofmt.sh
	echo "Verifying linting"
	hack/verify-golint.sh

test:
	go test -mod mod -v -cover -race ./...

build-docker:
	docker build \
		-t ${REPO}/${IMAGE}:latest \
		-t ${REPO}/${IMAGE}:${TAG} \
		--build-arg TAG=${TAG} .

build-server:
	go build -mod mod -ldflags \
			"-w -s -X sigs.k8s.io/aws-encryption-provider/pkg/version.Version=${TAG}" \
			-o bin/aws-encryption-provider cmd/server/main.go

build-client:
	go build -mod mod -ldflags "-w -s" -o bin/grpcclient cmd/client/main.go

