#!/usr/bin/env bash
set -e

FILES=$(git diff --cached --name-only --diff-filter=ACMR)

gofumpt -l -w .
golangci-lint run --new --fix

git add $FILES
