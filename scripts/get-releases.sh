#!/bin/bash
set -euo pipefail

get_last_page() {
	local url="$1"
	curl -sSf -I -H "Authorization: Bearer $GITHUB_TOKEN" \
		"$url" |
		grep -E '^link: ' |
		sed -e 's/^link:.*page=//g' -e 's/>.*$//g' || echo "1"
}

generate() {
	local url="$1"
	local file="$2"
	last_page="$(get_last_page "$url")"
	tmp="$(mktemp -d)"

	for i in $(seq 1 "$last_page"); do
		echo "page: $i"
		curl -H "Authorization: Bearer $GITHUB_TOKEN" -sSf "$url?page=$i" >"$tmp/$i.json"
	done

	if test "$last_page" -eq "1"; then
		jq --compact-output 'map({tag_name: .tag_name})' "$tmp"/1.json >"$file"
	else
		jq --compact-output '[inputs] | add | map({tag_name: .tag_name})' "$tmp"/*.json >"$file"
	fi
	du -hs "$file"
}

generate "https://api.github.com/repos/goreleaser/goreleaser/releases" "www/docs/static/releases.json"
generate "https://api.github.com/repos/goreleaser/goreleaser-pro/releases" "www/docs/static/releases-pro.json"
