#!/bin/bash
set -euo pipefail

url="https://api.github.com/repos/goreleaser/goreleaser/releases"

get_last_page() {
	curl -sSf -I -H "Authorization: Bearer $GITHUB_TOKEN" \
		"$url" |
		grep -E '^link: ' |
		sed -e 's/^link:.*page=//g' -e 's/>.*$//g'
}

last_page="$(get_last_page)"
tmp="$(mktemp -d)"

for i in $(seq 1 "$last_page"); do
	echo "page: $i"
	curl -H "Authorization: Bearer $GITHUB_TOKEN" -sSf "$url?page=$i" >"$tmp/$i.json"
done

jq '[inputs] | add' "$tmp"/*.json >www/docs/static/releases.json
