package completion

import (
	"strings"
	"testing"
)

func TestZshScriptContainsCommandsAndFlags(t *testing.T) {
	script := ZshScript()
	expected := []string{
		"_agentsync_zsh_complete()",
		"install:安装工具",
		"sync:从已保存",
		"source:查看",
		"upgrade:检查并升级",
		"status:显示仓库",
		"doctor:检查依赖",
		"shell-init:输出",
		"uninstall:删除",
		"version:显示版本",
		"help:显示帮助",
		"--shell[选择要配置的 Shell 环境]:shell:(auto zsh bash none)",
		"--tools[选择启用的 AI 工具列表]:tools:(codex claude agy auto)",
		"--dry-run[只显示计划",
		"--auto[后台静默自动同步",
		"show:查看当前",
		"test:验证当前",
		"set:验证并交互式切换",
		"(-V --version)",
		"(-h --help)",
	}
	for _, item := range expected {
		if !strings.Contains(script, item) {
			t.Errorf("ZshScript missing expected snippet: %q", item)
		}
	}
}

func TestBashScriptContainsCommandsAndFlags(t *testing.T) {
	script := BashScript()
	expected := []string{
		"_agentsync_bash_complete()",
		"local commands=\"install sync source upgrade status doctor shell-init uninstall version help\"",
		"compgen -W \"auto zsh bash none\"",
		"compgen -W \"codex claude agy auto\"",
		"compgen -W \"--shell --tools --dry-run\"",
		"compgen -W \"--auto\"",
		"compgen -W \"show test set\"",
		"compgen -W \"-V --version\"",
		"compgen -W \"-h --help\"",
	}
	for _, item := range expected {
		if !strings.Contains(script, item) {
			t.Errorf("BashScript missing expected snippet: %q", item)
		}
	}
}

func TestShellInitScript(t *testing.T) {
	combined := ShellInitScript()
	if !strings.Contains(combined, "_agentsync_zsh_complete") {
		t.Error("ShellInitScript missing Zsh completion")
	}
	if !strings.Contains(combined, "_agentsync_bash_complete") {
		t.Error("ShellInitScript missing Bash completion")
	}
	if !strings.Contains(combined, "compdef _agentsync_zsh_complete agentsync") {
		t.Error("ShellInitScript missing Zsh compdef registration")
	}
	if !strings.Contains(combined, "complete -F _agentsync_bash_complete agentsync") {
		t.Error("ShellInitScript missing Bash complete registration")
	}
}
