package completion

import "strings"

// ZshScript returns the Zsh completion function and registration for agentsync.
func ZshScript() string {
	return `_agentsync_zsh_complete() {
    local -a commands
    commands=(
        'install:安装工具、同步规则、创建 AI 入口并配置 Shell'
        'sync:从已保存的 HTTP(S) 链接原子部署最新 AGENTS.md'
        'source:查看、验证或更换 AGENTS.md 来源链接'
        'upgrade:检查并升级到最新正式版本'
        'status:显示仓库、runtime 和入口状态'
        'doctor:检查依赖、版本一致性、入口和 Shell 配置'
        'shell-init:输出 Zsh/Bash wrapper，可供 source/eval 使用'
        'uninstall:删除本工具明确管理的入口和 Shell 配置'
        'version:显示版本'
        'help:显示帮助信息'
    )

    _arguments -C \
        '1: :->command' \
        '*:: :->args'

    case "$state" in
        command)
            _describe -t commands 'agentsync 命令' commands
            ;;
        args)
            case "$words[1]" in
                install)
                    _arguments \
                        '--shell[选择要配置的 Shell 环境]:shell:(auto zsh bash none)' \
                        '--tools[选择启用的 AI 工具列表]:tools:(codex claude agy auto)' \
                        '--dry-run[只显示计划，不修改任何文件]' \
                        '*:AGENTS.md 来源 URL:_urls'
                    ;;
                sync)
                    _arguments \
                        '--auto[后台静默自动同步模式]'
                    ;;
                source)
                    local -a source_actions
                    source_actions=(
                        'show:查看当前 AGENTS.md 来源'
                        'test:验证当前来源或候选 URL，不作修改'
                        'set:验证并交互式切换 AGENTS.md 来源'
                    )
                    if (( CURRENT == 2 )); then
                        _describe -t source_actions 'source 操作' source_actions
                    else
                        case "$words[2]" in
                            test|set)
                                _arguments '*:AGENTS.md 来源 URL:_urls'
                                ;;
                        esac
                    fi
                    ;;
                version)
                    _arguments \
                        '(-V --version)'{-V,--version}'[显示版本]'
                    ;;
                help)
                    _arguments \
                        '(-h --help)'{-h,--help}'[显示帮助]'
                    ;;
            esac
            ;;
    esac
}`
}

// BashScript returns the Bash completion function and registration for agentsync.
func BashScript() string {
	return `_agentsync_bash_complete() {
    local cur prev words cword
    if type _init_completion >/dev/null 2>&1; then
        _init_completion -n = 2>/dev/null
    else
        cur="${COMP_WORDS[COMP_CWORD]}"
        prev="${COMP_WORDS[COMP_CWORD-1]}"
        words=("${COMP_WORDS[@]}")
        cword=$COMP_CWORD
    fi

    local commands="install sync source upgrade status doctor shell-init uninstall version help"

    if [[ $cword -eq 1 ]]; then
        COMPREPLY=($(compgen -W "$commands" -- "$cur"))
        return 0
    fi

    local command="${words[1]}"
    case "$command" in
        install)
            case "$prev" in
                --shell)
                    COMPREPLY=($(compgen -W "auto zsh bash none" -- "$cur"))
                    return 0
                    ;;
                --tools)
                    COMPREPLY=($(compgen -W "codex claude agy auto" -- "$cur"))
                    return 0
                    ;;
            esac
            if [[ "$cur" == -* ]]; then
                COMPREPLY=($(compgen -W "--shell --tools --dry-run" -- "$cur"))
                return 0
            fi
            ;;
        sync)
            if [[ "$cur" == -* ]]; then
                COMPREPLY=($(compgen -W "--auto" -- "$cur"))
                return 0
            fi
            ;;
        source)
            if [[ $cword -eq 2 ]]; then
                COMPREPLY=($(compgen -W "show test set" -- "$cur"))
                return 0
            fi
            ;;
        version)
            if [[ "$cur" == -* ]]; then
                COMPREPLY=($(compgen -W "-V --version" -- "$cur"))
                return 0
            fi
            ;;
        help)
            if [[ "$cur" == -* ]]; then
                COMPREPLY=($(compgen -W "-h --help" -- "$cur"))
                return 0
            fi
            ;;
    esac
}`
}

// ShellInitScript returns the combined completion block suitable for shell-init.
func ShellInitScript() string {
	var sb strings.Builder
	sb.WriteString(ZshScript())
	sb.WriteString("\n\n")
	sb.WriteString(BashScript())
	sb.WriteString(`

if [ -n "$ZSH_VERSION" ]; then
    if ! type compdef >/dev/null 2>&1; then
        autoload -Uz compinit 2>/dev/null && compinit -C 2>/dev/null || true
    fi
    if type compdef >/dev/null 2>&1; then
        compdef _agentsync_zsh_complete agentsync 2>/dev/null || true
    fi
elif [ -n "$BASH_VERSION" ]; then
    complete -F _agentsync_bash_complete agentsync 2>/dev/null || true
fi
`)
	return sb.String()
}
