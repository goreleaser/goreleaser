#!/bin/bash
set -euo pipefail
./scripts/get-releases.sh
mkdocs build -f www/mkdocs.yml
