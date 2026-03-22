#!/usr/bin/env bash
set -euo pipefail

# unshallow (needed for the rss plugin)
git fetch --prune --tags --unshallow

# prepare
version="$(cat ./www/static/latest)"
sed -i "s/__VERSION__/$version/g" www/content/getting-started/install.md

# build
cd www && hugo --gc --minify
