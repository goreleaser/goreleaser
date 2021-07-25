#!/bin/bash
FILES=$(git diff --staged --diff-filter=AM --no-renames --name-only)

gofumpt -s -l -w $FILES
golangci-lint run --new --fix

git add $FILES
