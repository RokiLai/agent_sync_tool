#!/bin/sh
set -eu
release_dir=${1:-dist}
primary_artifact_prefix=agentsync
legacy_artifact_prefix=aic
(cd "$release_dir"; if command -v shasum >/dev/null 2>&1; then shasum -a 256 -c checksums.txt; else sha256sum -c checksums.txt; fi)
for file in "$release_dir"/"${primary_artifact_prefix}"_* "$release_dir"/"${legacy_artifact_prefix}"_*; do [ -s "$file" ]; done
