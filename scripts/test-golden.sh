#!/bin/bash
set -euo pipefail

pkgs() {
	grep -Rl 'internal/golden"' . |
		grep '_test.go' |
		grep -v 'main' |
		while read -r file; do
			echo "$(dirname "$file")/..."
		done |
		sort |
		uniq |
		tr '\n' ' '
}

# shellcheck disable=SC2046
go test --failfast $(pkgs) -update
