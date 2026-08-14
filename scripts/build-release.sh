#!/bin/sh
set -eu
output=${1:-dist}; version=${AIC_BUILD_VERSION:-3.2.0}
command_name=agentsync
primary_artifact_prefix=agentsync
legacy_artifact_prefix=aic
case "$version" in [vV]*) version=${version#?} ;; esac
[ -n "$version" ] || { printf '版本号不能为空\n' >&2; exit 2; }
mkdir -p "$output"
build(){ GOOS=$1 GOARCH=$2 go build -trimpath -ldflags "-s -w -X github.com/RokiLai/agent_sync_tool/internal/app.Version=$version" -o "$output/$3" "./cmd/$command_name"; }
build darwin arm64 "${primary_artifact_prefix}_Darwin_arm64"
build darwin amd64 "${primary_artifact_prefix}_Darwin_x86_64"
build linux arm64 "${primary_artifact_prefix}_Linux_arm64"
build linux amd64 "${primary_artifact_prefix}_Linux_x86_64"
for file in "$output"/"${primary_artifact_prefix}"_*; do cp "$file" "$output/${legacy_artifact_prefix}_${file##*${primary_artifact_prefix}_}"; done
(cd "$output"; if command -v shasum >/dev/null 2>&1; then shasum -a 256 "${legacy_artifact_prefix}"_* "${primary_artifact_prefix}"_* > checksums.txt; else sha256sum "${legacy_artifact_prefix}"_* "${primary_artifact_prefix}"_* > checksums.txt; fi)
