#!/bin/bash
set -euo pipefail
yum install -y jq
pip install mkdocs-material mkdocs-redirects mkdocs-minify-plugin lunr
