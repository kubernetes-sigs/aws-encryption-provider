#!/usr/bin/env bash
set -euo pipefail

source hack/setup-go.sh

go version
go build -ldflags "-w -s" -o bin/grpcclient cmd/client/main.go
go build -ldflags "-w -s" -o bin/grpcclientv2 cmd/clientv2/main.go
