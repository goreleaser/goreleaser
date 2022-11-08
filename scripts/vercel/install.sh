#!/bin/bash
set -euo pipefail
yum install -y jq
pip install --upgrade pip
pip install mkdocs-material mkdocs-redirects mkdocs-minify-plugin lunr
