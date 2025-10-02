#!/bin/bash
pkg="$1"
timeout="$2"

grep "func Fuzz" "$pkg"/*.go |
	cut -f2 -d' ' |
	cut -f1 -d'(' |
	while read -r f; do
		go test -fuzztime="$timeout" -fuzz="$f" "$pkg"/...
	done
go test "$pkg"/...
