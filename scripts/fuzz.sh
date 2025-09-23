#!/bin/bash
pkg="$1"
timeout="$1"

grep "func Fuzz" "$pkg" |
	cut -f1 -d'(' |
	cut -f2 -d' ' |
	while read -r f; do
		go test -fuzztime="$timeout" -fuzz="$f" "$pkg"/...
	done
