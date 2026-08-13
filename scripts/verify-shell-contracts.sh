#!/bin/sh

set -eu

script_dir=$(CDPATH= cd -- "$(dirname "$0")" && pwd) || exit 1
project_dir=$(CDPATH= cd -- "$script_dir/.." && pwd) || exit 1
source_project=${AI_INSTRUCTIONS_SOURCE_PROJECT:-"$project_dir/../ai-instructions"}
source_cli="$source_project/bin/ai-instructions"
contract_dir="$project_dir/test/contract"
temp_dir=$(mktemp -d "${TMPDIR:-/tmp}/aic-contract.XXXXXX") || exit 1

cleanup() {
    case "$temp_dir" in
        "${TMPDIR:-/tmp}"/aic-contract.*) rm -rf "$temp_dir" ;;
        *) printf '拒绝清理非预期目录：%s\n' "$temp_dir" >&2 ;;
    esac
}
trap cleanup 0 HUP INT TERM

[ -f "$source_project/AGENTS.md" ] || { printf '源项目无效：%s\n' "$source_project" >&2; exit 1; }
[ -f "$source_project/changes/ai-instructions-cli.yaml" ] || exit 1
[ -x "$source_cli" ] || { printf '源 CLI 不可执行：%s\n' "$source_cli" >&2; exit 1; }

sh -n "$source_project/ai-instructions"
sh -n "$source_project/install.sh"
sh -n "$source_cli"
sh -n "$source_project/tests/test-ai-instructions.sh"

sh "$source_cli" help > "$temp_dir/help.txt"
sh "$source_cli" version > "$temp_dir/version.txt"
cmp "$contract_dir/golden/help.txt" "$temp_dir/help.txt"
cmp "$contract_dir/golden/version.txt" "$temp_dir/version.txt"

tab=$(printf '\t')
while IFS="$tab" read -r expected escaped; do
    case "$expected" in ''|'#'*) continue ;; esac
    actual=$(printf '%b' "$escaped" | git hash-object --stdin)
    [ "$actual" = "$expected" ] || {
        printf 'Git blob 向量不匹配：expected=%s actual=%s\n' "$expected" "$actual" >&2
        exit 1
    }
done < "$contract_dir/git_blob_vectors.txt"

for marker in \
    '# ai-instructions managed file v1' \
    '# ai-instructions repository path v1' \
    '# ai-instructions AGENTS URL v1' \
    '# >>> ai-instructions managed block >>>' \
    '# <<< ai-instructions managed block <<<'
do
    grep -Fq "$marker" "$source_cli" || { printf '缺少 marker：%s\n' "$marker" >&2; exit 1; }
done

sh "$source_project/tests/test-ai-instructions.sh" > "$temp_dir/regression.out"
grep -Fqx '1..36' "$temp_dir/regression.out"
[ "$(grep -c '^ok [0-9][0-9]* - ' "$temp_dir/regression.out")" -eq 36 ]

printf 'Shell 契约验证通过：36/36\n'
