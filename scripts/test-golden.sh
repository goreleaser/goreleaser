#!/bin/bash
set -euo pipefail
grep -Rl 'internal/golden"' . |
	grep '_test.go' |
	grep -v 'main' |
	sort |
	uniq |
	while read -r file; do
		go test --failfast "$(dirname "$file")/..." -update
	done
