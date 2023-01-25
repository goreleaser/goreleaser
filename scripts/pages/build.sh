#!/bin/bash
set -euo pipefail

# install
pip install --upgrade pip
pip install -U mkdocs-material mkdocs-redirects mkdocs-minify-plugin lunr

# prepare
version="$(cat ./www/docs/static/latest)"
sed -s'' -i "s/__VERSION__/$version/g" www/docs/install.md

# build
mkdocs build -f www/mkdocs.yml
