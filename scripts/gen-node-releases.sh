#!/usr/bin/env bash
# Refreshes internal/nodedist/releases.json from nodejs.org/dist.
#
# Output schema:
#   { "vX.Y.Z": { "<name>": "<sha256>" } }
#
# Only releases satisfying the SEA min-version constraint are kept,
# and only file names matching internal/builders/node/targets.txt are
# included for each release.
set -euo pipefail

readonly TARGETS_FILE="internal/builders/node/targets.txt"
readonly OUT="internal/nodedist/releases.json"

targets=()
while IFS= read -r line; do
	[[ -z "$line" ]] && continue
	targets+=("$line")
done < "$TARGETS_FILE"
targets_json=$(printf '%s\n' "${targets[@]}" | jq -R . | jq -s .)

# Pull the upstream index, keep only releases the SEA builder will
# accept (mirrors minTargetConstraint in internal/nodesea).
versions=$(curl -fsSL https://nodejs.org/dist/index.json | jq -r '
  map(select(
    (.version | ltrimstr("v") | split(".") | map(tonumber)) as $v |
      ($v[0] == 22 and $v[1] >= 20)
      or ($v[0] == 24 and $v[1] >= 6)
      or ($v[0] >= 25)
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
