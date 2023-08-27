#!/bin/bash
set -euo pipefail

# unshallow
git fetch --prune --tags --unshallow

# install
pip install --upgrade pip
pip install -U mkdocs-material mkdocs-redirects mkdocs-minify-plugin mkdocs-include-markdown-plugin lunr mkdocs-rss-plugin

# prepare
version="$(cat ./www/docs/static/latest)"
sed -s'' -i "s/__VERSION__/$version/g" www/docs/install.md

# build
mkdocs build -f www/mkdocs.yml
