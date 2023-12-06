#!/usr/bin/env bash
set -euo pipefail

source hack/setup-go.sh

go version
go build -ldflags \
		"-w -s -X sigs.k8s.io/aws-encryption-provider/pkg/version.Version=${TAG}" \
		-o bin/aws-encryption-provider cmd/server/main.go
