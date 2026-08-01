#!/usr/bin/env bash
set -euo pipefail

latest() {
	local repo="$1"
	local file="$2"
	gh api "repos/$repo/releases/latest" --jq ".tag_name" >"$file"
	wc -c "$file"
}

generate() {
	local repo="$1"
	local file="$2"
	gh api --paginate "repos/$repo/releases" \
		--jq '.[] | select(.tag_name | contains("nightly") | not) | {tag_name}' |
		jq -s '.' >"$file"
	wc -c "$file"
}

latest "goreleaser/goreleaser" "www/static/latest"
latest "goreleaser/goreleaser-pro" "www/static/latest-pro"
generate "goreleaser/goreleaser" "www/static/releases.json"
generate "goreleaser/goreleaser-pro" "www/static/releases-pro.json"
