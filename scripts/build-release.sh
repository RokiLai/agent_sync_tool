#!/bin/sh
set -eu
output=${1:-dist}; version=${AIC_BUILD_VERSION:-3.1.1}
case "$version" in [vV]*) version=${version#?} ;; esac
[ -n "$version" ] || { printf '版本号不能为空\n' >&2; exit 2; }
mkdir -p "$output"
build(){ GOOS=$1 GOARCH=$2 go build -trimpath -ldflags "-s -w -X github.com/RokiLai/agent_sync_tool/internal/app.Version=$version" -o "$output/$3" ./cmd/aic; }
build darwin arm64 aic_Darwin_arm64
build darwin amd64 aic_Darwin_x86_64
build linux arm64 aic_Linux_arm64
build linux amd64 aic_Linux_x86_64
(cd "$output"; if command -v shasum >/dev/null 2>&1; then shasum -a 256 aic_* > checksums.txt; else sha256sum aic_* > checksums.txt; fi)
