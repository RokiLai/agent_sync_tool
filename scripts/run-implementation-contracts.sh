#!/bin/sh

set -eu

implementation=${1:-all}
case "$implementation" in all|shell|go) ;; *) printf '实现必须是 all、shell 或 go\n' >&2; exit 2 ;; esac

script_dir=$(CDPATH= cd -- "$(dirname "$0")" && pwd)
project_dir=$(CDPATH= cd -- "$script_dir/.." && pwd)
source_project=${AI_INSTRUCTIONS_SOURCE_PROJECT:-"$project_dir/../ai-instructions"}

case "$implementation" in
    shell) sh "$project_dir/scripts/verify-shell-contracts.sh" ;;
    go) go test -race -timeout 2m ./... ;;
    all)
        sh "$project_dir/scripts/verify-shell-contracts.sh"
        go test -race -timeout 2m ./...
        ;;
esac

printf '%s 实现契约验证通过\n' "$implementation"
