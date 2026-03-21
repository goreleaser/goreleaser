#!/usr/bin/env bash
# copy-doc.sh <src> <dst>
# Copies src to dst, preserving any existing front matter in dst.
# If the front matter contains a title:, the first H1 heading in src is stripped
# (since it would be redundant with the title in front matter).
set -euo pipefail

src="$1"
dst="$2"

frontmatter=""
has_title=false

if [[ -f "$dst" ]] && head -1 "$dst" | grep -q '^---$'; then
	frontmatter=$(awk '/^---$/{p++; if(p==2){exit} next} p==1{print}' "$dst")
	if echo "$frontmatter" | grep -q '^title:'; then
		has_title=true
	fi
fi

if [[ -n "$frontmatter" ]]; then
	printf -- '---\n%s\n---\n' "$frontmatter" >"$dst"
	if [[ "$has_title" == "true" ]]; then
		awk 'BEGIN{removed=0} /^# /{if(!removed){removed=1; next}} {print}' "$src" >>"$dst"
	else
		cat "$src" >>"$dst"
	fi
else
	cp "$src" "$dst"
fi
