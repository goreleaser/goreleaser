#!/bin/sh
set -e

SED="sed"
if which gsed >/dev/null 2>&1; then
	SED="gsed"
fi

rm -rf www/docs/cmd/*.md
go run . docs
"$SED" \
	-i'' \
	-e 's/SEE ALSO/See also/g' \
	-e 's/^## /# /g' \
	-e 's/^### /## /g' \
	-e 's/^#### /### /g' \
	-e 's/^##### /#### /g' \
	./www/docs/cmd/*.md
