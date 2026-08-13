#!/bin/sh

set -eu

[ "$#" -eq 2 ] || { printf '用法：%s SHELL_TREE GO_TREE\n' "$0" >&2; exit 2; }

snapshot() {
    snapshot_root=$1
    snapshot_output=$2
    (
        cd "$snapshot_root"
        find . -mindepth 1 -not -path './repo/.git*' -not -name '*.backup.*' -print | LC_ALL=C sort | while IFS= read -r path; do
            if [ -L "$path" ]; then
                printf 'L\t%s\t%s\n' "$path" "$(readlink "$path")"
            elif [ -f "$path" ]; then
                mode=$(stat -f '%Lp' "$path" 2>/dev/null || stat -c '%a' "$path")
                hash=$(git hash-object "$path")
                printf 'F\t%s\t%s\t%s\n' "$path" "$mode" "$hash"
            elif [ -d "$path" ]; then
                printf 'D\t%s\n' "$path"
            fi
        done
    ) > "$snapshot_output"
}

temp_dir=$(mktemp -d "${TMPDIR:-/tmp}/aic-tree-compare.XXXXXX") || exit 1
trap 'rm -rf "$temp_dir"' 0 HUP INT TERM
snapshot "$1" "$temp_dir/shell.tree"
snapshot "$2" "$temp_dir/go.tree"
diff -u "$temp_dir/shell.tree" "$temp_dir/go.tree"
