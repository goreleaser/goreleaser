#!/usr/bin/env bash
set -euo pipefail

# unshallow (needed for the rss plugin)
git fetch --prune --tags --unshallow

# install
pip install --upgrade pip
pip install -r ./www/requirements.txt

# prepare
version="$(cat ./www/docs/static/latest)"
sed -s'' -i "s/__VERSION__/$version/g" www/docs/install.md www/docs/customization/index.md

# build
mkdocs build -f www/mkdocs.yml
