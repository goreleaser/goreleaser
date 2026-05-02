#!/usr/bin/env bash
# Refreshes internal/nodedist/releases.json from nodejs.org/dist.
#
# Output schema:
#   { "vX.Y.Z": { "<name>": "<sha256>" } }
#
# Only releases satisfying the SEA min-version constraint are kept,
# and only file names matching the supported targets list are included
# for each release. Keep this list in sync with `supportedTargets` in
# internal/builders/node/targets.go.
set -euo pipefail

readonly OUT="internal/nodedist/releases.json"

targets=(
	darwin-arm64
	darwin-x64
	linux-arm64
	linux-x64
	win-arm64
	win-x64
)
targets_json=$(printf '%s\n' "${targets[@]}" | jq -R . | jq -s .)

# Pull the upstream index, keep only Node releases that ship `--build-sea`
# (v25.5.0+ where LIEF is bundled into the official builds).
versions=$(curl -fsSL https://nodejs.org/dist/index.json | jq -r '
  map(select(
    (.version | ltrimstr("v") | split(".") | map(tonumber)) as $v |
      ($v[0] > 25)
      or ($v[0] == 25 and $v[1] >= 5)
  ))
  | .[] | .version
')

tmp=$(mktemp)
trap 'rm -f "$tmp" "$tmp.sha"' EXIT
echo "{}" > "$tmp"

while IFS= read -r version; do
	[[ -z "$version" ]] && continue
	echo "fetching $version" >&2
	curl -fsSL "https://nodejs.org/dist/$version/SHASUMS256.txt" > "$tmp.sha"
	jq --arg version "$version" \
		--argjson targets "$targets_json" \
		--rawfile shasums "$tmp.sha" \
		'
		  ($shasums
		    | split("\n")
		    | map(split("  ") | select(length == 2))
		    | map({(.[1]): .[0]}) | add) as $shas |
		  ($targets | map(
		    if startswith("win-") then "\(.)/node.exe"
		    else "node-\($version)-\(.).tar.gz" end
		  )) as $names |
		  ($names
		    | map({name: ., sha: $shas[.]})
		    | map(select(.sha))
		    | map({(.name): .sha})
		    | add // {}) as $files |
		  . + { ($version): $files }
		' "$tmp" > "$tmp.new"
	mv "$tmp.new" "$tmp"
done <<< "$versions"

jq -S . "$tmp" > "$OUT"
echo "wrote $OUT" >&2
