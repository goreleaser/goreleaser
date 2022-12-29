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

	for i in $(seq -w 1 "$last_page"); do
		echo "page: $i"
		curl -H "Authorization: Bearer $GITHUB_TOKEN" -sSf "$url?page=$i" | jq 'map({tag_name: .tag_name})' >"$tmp/$i.json"
	done

	jq -s 'add' "$tmp"/*.json >"$file"
	du -hs "$file"
}

latest() {
	local url="$1"
	local file="$2"
	curl -sfL "$url/latest" | jq -r ".tag_name" >"$file"
	du -hs "$file"
}

latest "https://api.github.com/repos/goreleaser/goreleaser/releases" "www/docs/static/latest"
latest "https://api.github.com/repos/goreleaser/goreleaser-pro/releases" "www/docs/static/latest-pro"
generate "https://api.github.com/repos/goreleaser/goreleaser/releases" "www/docs/static/releases.json"
generate "https://api.github.com/repos/goreleaser/goreleaser-pro/releases" "www/docs/static/releases-pro.json"
