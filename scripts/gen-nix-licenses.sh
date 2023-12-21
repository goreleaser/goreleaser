#!/usr/bin/env bash
set -euo pipefail

{
	echo "// AUTO-GENERATED. DO NOT EDIT."
	echo
	echo "package nix"
	echo "var validLicenses = []string {"
	curl -s https://raw.githubusercontent.com/NixOS/nixpkgs/master/lib/licenses.nix |
		grep -E '.* = \{' |
		grep -v default |
		cut -f1 -d= |
		awk '{print "\"" $1 "\","}'
	echo -e "}"
} >./internal/pipe/nix/licenses.go

gofumpt -w ./internal/pipe/nix/licenses.go
