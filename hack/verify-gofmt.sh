#!/usr/bin/env bash
set -euo pipefail

source hack/setup-go.sh

go version

test -z "$(gofmt -s -d cmd pkg)"
