#!/bin/sh
set -e
rm -rf www/docs/cmd/*.md
go run . docs
if which gsed; then
	gsed -i'' -e 's/SEE ALSO/See also/g' -e 's/^## /# /g' -e 's/^### /## /g' -e 's/^#### /### /g' -e 's/^##### /#### /g' ./www/docs/cmd/*.md
else
	sed -i'' 's/SEE ALSO/See also/g' ./www/docs/cmd/*.md
fi
