#!/bin/bash
set -euo pipefail

# install
pip install --upgrade pip
pip install -U mkdocs-material mkdocs-redirects mkdocs-minify-plugin lunr

# prepare
./scripts/get-releases.sh
version="$(curl -sSf -H "Authorization: Bearer $GITHUB_TOKEN" "https://api.github.com/repos/goreleaser/goreleaser/releases/latest" | jq -r '.tag_name')"
sed -s'' -i "s/__VERSION__/$version/g" www/docs/install.md

# build
mkdocs build -f www/mkdocs.yml
