#!/bin/sh
set -eu
base_url=${AGENTSYNC_RELEASE_BASE_URL:-${AIC_RELEASE_BASE_URL:-https://github.com/RokiLai/agent_sync_tool/releases}}
version=${AGENTSYNC_VERSION:-${AIC_VERSION:-latest}}
command_name=agentsync
artifact_prefix=agentsync
temp_dir=$(mktemp -d "${TMPDIR:-/tmp}/${command_name}-bootstrap.XXXXXX") || exit 1
trap 'rm -rf "$temp_dir"' 0 HUP INT TERM
case "$(uname -s)" in Darwin) release_os=Darwin ;; Linux) release_os=Linux ;; *) printf '错误：不支持的操作系统\n' >&2; exit 1 ;; esac
case "$(uname -m)" in x86_64|amd64) release_arch=x86_64 ;; arm64|aarch64) release_arch=arm64 ;; *) printf '错误：不支持的架构\n' >&2; exit 1 ;; esac
artifact="${artifact_prefix}_${release_os}_${release_arch}"
if [ "$version" = latest ]; then release_url="$base_url/latest/download"; else release_url="$base_url/download/$version"; fi
curl --fail --silent --show-error --location --proto '=https,http' --proto-redir '=https,http' "$release_url/checksums.txt" -o "$temp_dir/checksums.txt"
curl --fail --silent --show-error --location --proto '=https,http' --proto-redir '=https,http' "$release_url/$artifact" -o "$temp_dir/$artifact"
expected=$(awk -v name="$artifact" '$2 == name || $2 == "*" name { print $1; exit }' "$temp_dir/checksums.txt")
[ -n "$expected" ] || { printf '错误：checksum清单缺少%s\n' "$artifact" >&2; exit 1; }
if command -v shasum >/dev/null 2>&1; then actual=$(shasum -a 256 "$temp_dir/$artifact" | awk '{print $1}'); elif command -v sha256sum >/dev/null 2>&1; then actual=$(sha256sum "$temp_dir/$artifact" | awk '{print $1}'); else printf '错误：未找到SHA-256校验工具\n' >&2; exit 1; fi
[ "$actual" = "$expected" ] || { printf '错误：SHA-256校验失败\n' >&2; exit 1; }
chmod 700 "$temp_dir/$artifact"
"$temp_dir/$artifact" version >/dev/null
exec "$temp_dir/$artifact" install "$@"
