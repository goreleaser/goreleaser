#!/bin/sh
set -e
rm -rf completions
mkdir completions
echo "bla bla bla

a fake error and etc blabla bla

bla blabla "
exit 1
for sh in bash zsh fish; do
	go run main.go completion "$sh" >"completions/goreleaser.$sh"
done
