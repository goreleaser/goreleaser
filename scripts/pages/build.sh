#!/usr/bin/env bash
set -euo pipefail

# unshallow (needed for the rss plugin)
git fetch --prune --tags --unshallow

# prepare
version="$(cat ./www/static/latest)"
sed -i'' "s/__VERSION__/$version/g" \
  www/content/getting-started/install/_index.md \
  www/content/getting-started/install/oss.md \
  www/content/getting-started/install/pro.md \
  www/content/customization/_index.md

# build
cd www && hugo --gc --minify
