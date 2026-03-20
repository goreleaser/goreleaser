#!/bin/sh
set -e
rm -rf completions manpages
mkdir completions manpages
go build
for sh in bash zsh fish; do
	./goreleaser completion "$sh" >"completions/goreleaser.$sh"
done
./goreleaser man | gzip -c -9 >manpages/goreleaser.1.gz
