#!/usr/bin/env bash
set -euo pipefail

test -z "$(gofmt -s -d cmd pkg)"
