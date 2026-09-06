#!/usr/bin/env bash
set -e

gofumpt -l -w .
golangci-lint run --new --fix
