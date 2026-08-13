#!/bin/sh
set -eu
release_dir=${1:-dist}
(cd "$release_dir"; if command -v shasum >/dev/null 2>&1; then shasum -a 256 -c checksums.txt; else sha256sum -c checksums.txt; fi)
for file in "$release_dir"/aic_*; do [ -s "$file" ]; done
