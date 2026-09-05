#!/bin/bash
set -eo pipefail

pkg="$1"
timeout="$2"

sed -n 's/^func \(Fuzz[^ (]*\)(.*/\1/p' "$pkg"/*.go |
	while read -r f; do
		go test -fuzztime="$timeout" -fuzz="^$f$" "$pkg"/...
	done
go test "$pkg"/...
