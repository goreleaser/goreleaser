#!/usr/bin/env bash
set -e

FILES=()
while IFS= read -r -d '' file; do
	FILES+=("$file")
done < <(git diff --cached --name-only -z --diff-filter=ACMR)

gofumpt -l -w .
golangci-lint run --new --fix

if ((${#FILES[@]})); then
	git add -- "${FILES[@]}"
fi
