#!/bin/bash
set -euo pipefail
./scripts/get-releases.sh
mkdocs build -f www/mkdocs.yml
version=$(git describe --abbrev=0 --tags)
sed -s'' -i "s/__VERSION__/$version/g" www/docs/install.md
