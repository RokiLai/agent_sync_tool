package diagnose

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/RokiLai/agent_sync_tool/internal/config"
	"github.com/RokiLai/agent_sync_tool/internal/core"
	"github.com/RokiLai/agent_sync_tool/internal/identity"
	"github.com/RokiLai/agent_sync_tool/internal/probe"
)

type Dependencies struct {
	LookPath func(string) (string, error)
	GitRev   func(string) string
}

func DefaultDependencies() Dependencies {
	return Dependencies{
		LookPath: exec.LookPath,
		GitRev: func(repo string) string {
			out, err := exec.Command("git", "-C", repo, "rev-parse", "--verify", "origin/main^{commit}").Output()
			if err != nil {
				return "unknown"
			}
			return strings.TrimSpace(string(out))
		},
	}
}

func Status(out io.Writer, c config.Config, deps Dependencies) {
	if c.RepositorySource == "release" {
		ok(out, "安装模式：Release二进制")
	} else if isDir(filepath.Join(c.RepositoryDir, ".git")) {
		ok(out, "仓库：%s", c.RepositoryDir)
		info(out, "仓库路径来源：%s", c.RepositorySource)
		info(out, "origin/main：%s", deps.GitRev(c.RepositoryDir))
	} else {
		fmt.Fprintf(out, "[FAIL] 仓库不存在：%s\n", c.RepositoryDir)
	}
	if raw, err := ReadAgentsURL(c); err == nil {
		ok(out, "AGENTS.md 来源：%s", raw)
	} else {
		fmt.Fprintf(out, "[FAIL] AGENTS.md 来源未配置或无效：%s\n", filepath.Join(c.ConfigDir, "agents-url"))
	}
	state := core.InspectRuntime(c.RuntimeDir)
	if state.Valid {
		ok(out, "runtime：%s", c.RuntimeDir)
		info(out, "已部署版本：%s", state.Revision)
	} else {
		fmt.Fprintf(out, "[FAIL] runtime 不完整：%s\n", c.RuntimeDir)
	}

	runtimeFile := filepath.Join(c.RuntimeDir, "AGENTS.md")
	enabledMap := map[string]bool{}
	for _, k := range c.EnabledTools {
		enabledMap[k] = true
	}
	detected := probe.DetectTools(deps.LookPath, c.HomeDir, c.CodexHome)
	detectedMap := map[string]bool{}
	for _, k := range detected.DetectedKeys {
		detectedMap[k] = true
	}
	for _, tool := range identity.SupportedTools() {
		targetPath := tool.TargetPath(c.HomeDir, c.CodexHome)
		if enabledMap[tool.Key] {
			entryStatus(out, tool.DisplayName, targetPath, runtimeFile)
		} else {
			if symlinkEquals(targetPath, runtimeFile) {
				entryStatus(out, tool.DisplayName, targetPath, runtimeFile)
			} else if detectedMap[tool.Key] {
				fmt.Fprintf(out, "[INFO] 检测到 %s 已安装但未接入规则；运行 %s install 可自动接入：%s\n", tool.DisplayName, identity.PrimaryCommand, targetPath)
			} else {
				fmt.Fprintf(out, "[INFO] %s 入口未安装：%s\n", tool.DisplayName, targetPath)
			}
		}
	}
}

func Doctor(out io.Writer, c config.Config, deps Dependencies, shell string) bool {
	failures, warnings := 0, 0
	for _, command := range []string{"git", "curl"} {
		if _, err := deps.LookPath(command); err == nil {
			ok(out, "%s 可用", map[string]string{"git": "Git", "curl": "curl"}[command])
		} else {
			fmt.Fprintf(out, "[FAIL] 未找到 %s\n", map[string]string{"git": "Git", "curl": "curl"}[command])
			failures++
		}
	}
	if c.RepositorySource == "release" {
		ok(out, "Release二进制安装")
	} else if isDir(filepath.Join(c.RepositoryDir, ".git")) {
		ok(out, "中央仓库存在")
	} else {
		fmt.Fprintf(out, "[FAIL] 中央仓库不存在：%s\n", c.RepositoryDir)
		failures++
	}
	if _, err := config.ReadManagedValue(filepath.Join(c.ConfigDir, "repo-path"), config.RepoPathMarker); err == nil {
		ok(out, "仓库路径配置有效")
	} else {
		fmt.Fprintf(out, "[WARN] 仓库路径配置缺失或不受管：%s\n", filepath.Join(c.ConfigDir, "repo-path"))
		warnings++
	}
	if _, err := ReadAgentsURL(c); err == nil {
		ok(out, "AGENTS.md 来源配置有效")
	} else {
		fmt.Fprintf(out, "[FAIL] AGENTS.md 来源配置缺失或无效：%s\n", filepath.Join(c.ConfigDir, "agents-url"))
		failures++
	}
	state := core.InspectRuntime(c.RuntimeDir)
	if state.Valid {
		ok(out, "runtime 文件成对有效")
		data, err := os.ReadFile(filepath.Join(c.RuntimeDir, "AGENTS.md"))
		if err == nil && core.Revision(data) == state.Revision {
			ok(out, "runtime 内容与 REVISION 一致")
		} else {
			fmt.Fprintln(out, "[FAIL] runtime 内容与 REVISION 不一致")
			failures++
		}
	} else {
		fmt.Fprintln(out, "[FAIL] runtime 文件缺失或为空")
		failures++
	}
	if nonEmpty(filepath.Join(c.CodexHome, "AGENTS.override.md")) {
		fmt.Fprintln(out, "[WARN] Codex 存在非空 AGENTS.override.md，可能覆盖共享规则")
		warnings++
	}

	runtimeFile := filepath.Join(c.RuntimeDir, "AGENTS.md")
	enabledMap := map[string]bool{}
	for _, k := range c.EnabledTools {
		enabledMap[k] = true
	}
	for _, tool := range identity.SupportedTools() {
		targetPath := tool.TargetPath(c.HomeDir, c.CodexHome)
		if enabledMap[tool.Key] {
			if symlinkEquals(targetPath, runtimeFile) {
				ok(out, "%s 入口正确", tool.DisplayName)
			} else {
				fmt.Fprintf(out, "[WARN] %s 入口未受管或不正确：%s\n", tool.DisplayName, targetPath)
				warnings++
			}
		}
	}

	installed := filepath.Join(c.ConfigDir, "bin", identity.ManagedBinaryName)
	if info, err := os.Stat(installed); err == nil && info.Mode()&0111 != 0 {
		ok(out, "工具本体已安装")
	} else {
		fmt.Fprintf(out, "[WARN] 工具本体未安装或不可执行：%s\n", installed)
		warnings++
	}
	for _, name := range identity.CommandNames() {
		link := filepath.Join(c.BinDir, name)
		if symlinkEquals(link, installed) {
			ok(out, "命令入口正确：%s", link)
		} else {
			fmt.Fprintf(out, "[WARN] 命令入口未受管或不正确：%s\n", link)
			warnings++
		}
	}
	shellFile := filepath.Join(c.ConfigDir, "shell-integration.sh")
	if firstLine(shellFile) == config.ManagedMarker || firstLine(shellFile) == config.LegacyManagedMarker {
		ok(out, "Shell 集成文件正确")
	} else {
		fmt.Fprintf(out, "[WARN] Shell 集成文件未安装或不受管：%s\n", shellFile)
		warnings++
	}
	rc := ""
	switch filepath.Base(shell) {
	case "zsh":
		rc = filepath.Join(c.HomeDir, ".zshrc")
	case "bash":
		rc = filepath.Join(c.HomeDir, ".bashrc")
	}
	if rc != "" && (containsBoth(rc, config.BlockBegin, config.BlockEnd) || containsBoth(rc, config.LegacyBlockBegin, config.LegacyBlockEnd)) {
		ok(out, "Shell 配置块正确：%s", rc)
	} else {
		fmt.Fprintln(out, "[WARN] 当前 Shell 未加载受管配置块")
		warnings++
	}

	for _, tool := range identity.SupportedTools() {
		if enabledMap[tool.Key] {
			if _, err := deps.LookPath(tool.BinaryName); err == nil {
				ok(out, "%s 可执行程序可用", tool.BinaryName)
			} else {
				fmt.Fprintf(out, "[WARN] 未找到 %s 可执行程序\n", tool.BinaryName)
				warnings++
			}
		}
	}

	detected := probe.DetectTools(deps.LookPath, c.HomeDir, c.CodexHome)
	var unmanagedDetected []string
	for _, tool := range detected.DetectedTools {
		targetPath := tool.TargetPath(c.HomeDir, c.CodexHome)
		if !enabledMap[tool.Key] || !symlinkEquals(targetPath, runtimeFile) {
			unmanagedDetected = append(unmanagedDetected, tool.DisplayName)
		}
	}
	if len(unmanagedDetected) > 0 {
		info(out, "检测到未接入规则的 AI 工具（%s）；运行 %s install 可自动接入", strings.Join(unmanagedDetected, "、"), identity.PrimaryCommand)
	}

	info(out, "检查完成：%d 个失败，%d 个警告", failures, warnings)
	return failures == 0
}

func ReadAgentsURL(c config.Config) (string, error) {
	raw, err := config.ReadManagedValue(filepath.Join(c.ConfigDir, "agents-url"), config.AgentsURLMarker)
	if err != nil {
		return "", err
	}
	if err := core.ValidateURL(raw); err != nil {
		return "", err
	}
	return raw, nil
}

func ok(w io.Writer, format string, args ...any)   { fmt.Fprintf(w, "[OK] "+format+"\n", args...) }
func info(w io.Writer, format string, args ...any) { fmt.Fprintf(w, "[INFO] "+format+"\n", args...) }
func isDir(path string) bool                       { info, err := os.Stat(path); return err == nil && info.IsDir() }
func nonEmpty(path string) bool                    { info, err := os.Stat(path); return err == nil && info.Size() > 0 }
func symlinkEquals(path, target string) bool {
	value, err := os.Readlink(path)
	return err == nil && value == target
}
func firstLine(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.SplitN(string(data), "\n", 2)[0]
}
func containsBoth(path, a, b string) bool {
	data, err := os.ReadFile(path)
	return err == nil && strings.Contains(string(data), a) && strings.Contains(string(data), b)
}
func entryStatus(w io.Writer, name, path, target string) {
	if symlinkEquals(path, target) {
		ok(w, "%s 入口：%s", name, path)
	} else if _, err := os.Lstat(path); err == nil {
		fmt.Fprintf(w, "[WARN] %s 入口不是受管链接：%s\n", name, path)
	} else {
		fmt.Fprintf(w, "[INFO] %s 入口未安装：%s\n", name, path)
	}
}
