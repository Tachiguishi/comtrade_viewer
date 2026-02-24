#!/usr/bin/env bash
set -euo pipefail

# Usage:
#   ./test.sh                    # run all tests with coverage
#   ./test.sh TestName           # run a single test by name, no coverage

if [[ $# -gt 0 ]]; then
	test_name="$1"
	go test -v ./test -run "^${test_name}$"
	exit 0
fi

go test -v ./test \
	-cover \
	-coverpkg=./... \
	-coverprofile=coverage.out \
	&& go tool cover -html=coverage.out -o coverage.html \
	&& rm -f coverage.out

