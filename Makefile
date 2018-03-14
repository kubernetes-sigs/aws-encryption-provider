REPO?=gcr.io/must-override
IMAGE?=aws-encryption-provider
TAG?=0.0.1

.PHONY: test
test:
	go test ./...

.PHONY: build
build:
	docker build \
		-t ${REPO}/${IMAGE}:latest \
		-t ${REPO}/${IMAGE}:${TAG} \
		--build-arg TAG=${TAG} .
