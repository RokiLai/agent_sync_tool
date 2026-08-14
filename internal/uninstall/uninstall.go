package uninstall

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/RokiLai/agent_sync_tool/internal/config"
	"github.com/RokiLai/agent_sync_tool/internal/core"
	"github.com/RokiLai/agent_sync_tool/internal/identity"
)

type Plan struct {
	Symlinks, Files, RCBlocks, RuntimeFiles, RuntimeDirs, Warnings []string
	RuntimeDir                                                     string
}

func Build(c config.Config, shell string) Plan {
	p := Plan{RuntimeDir: c.RuntimeDir}
	runtimeFile := filepath.Join(c.RuntimeDir, "AGENTS.md")
	for _, tool := range identity.SupportedTools() {
		path := tool.TargetPath(c.HomeDir, c.CodexHome)
		collectLink(&p, path, runtimeFile)
	}
	installed := filepath.Join(c.ConfigDir, "bin", identity.ManagedBinaryName)
	for _, name := range identity.HistoricalCommandNames() {
		collectLink(&p, filepath.Join(c.BinDir, name), installed)
	}
	for path, marker := range map[string]string{
		filepath.Join(c.ConfigDir, "repo-path"):            config.RepoPathMarker,
		filepath.Join(c.ConfigDir, "agents-url"):           config.AgentsURLMarker,
		filepath.Join(c.ConfigDir, "enabled-tools"):        config.EnabledToolsMarker,
		filepath.Join(c.ConfigDir, "shell-integration.sh"): config.ManagedMarker,
	} {
		if firstLine(path) == marker {
			p.Files = append(p.Files, path)
		} else if exists(path) {
			p.Warnings = append(p.Warnings, "配置不受本工具管理，保留："+path)
		}
	}
	if data, err := os.ReadFile(installed); err == nil && strings.Contains(string(data), identity.VersionOutputName) {
		p.Files = append(p.Files, installed)
	}
	rc := core.RCPath(c.HomeDir, shell)
	if data, err := os.ReadFile(rc); err == nil && strings.Contains(string(data), config.BlockBegin) {
		if strings.Contains(string(data), config.BlockEnd) {
			p.RCBlocks = append(p.RCBlocks, rc)
		} else {
			p.Warnings = append(p.Warnings, "Shell 配置受管块不完整，将保留："+rc)
		}
	}
	if core.InspectRuntime(c.RuntimeDir).Valid {
		p.RuntimeFiles = []string{filepath.Join(c.RuntimeDir, "AGENTS.md"), filepath.Join(c.RuntimeDir, "REVISION"), filepath.Join(c.RuntimeDir, "current"), filepath.Join(c.RuntimeDir, "LAST_CHECKED")}
		p.RuntimeDirs = []string{filepath.Join(c.RuntimeDir, "versions")}
	}
	return p
}
func collectLink(p *Plan, path, target string) {
	if got, err := os.Readlink(path); err == nil && got == target {
		p.Symlinks = append(p.Symlinks, path)
	} else if exists(path) {
		p.Warnings = append(p.Warnings, "不是本工具管理的入口，保留："+path)
	}
}
func Print(w io.Writer, p Plan) {
	fmt.Fprintln(w, "即将执行以下操作：\n删除：")
	for _, items := range [][]string{p.Symlinks, p.Files, p.RCBlocks} {
		for _, path := range items {
			fmt.Fprintf(w, "  %s\n", path)
		}
	}
	fmt.Fprintf(w, "保留：\n  %s\n", p.RuntimeDir)
	for _, warning := range p.Warnings {
		fmt.Fprintf(w, "警告：%s\n", warning)
	}
}
func Execute(p Plan, purge bool) error {
	for _, path := range p.RCBlocks {
		if err := core.RemoveRC(path); err != nil {
			return err
		}
	}
	for _, items := range [][]string{p.Symlinks, p.Files} {
		for _, path := range items {
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				return err
			}
		}
	}
	if purge {
		for _, path := range p.RuntimeFiles {
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				return err
			}
		}
		for _, path := range p.RuntimeDirs {
			if err := os.RemoveAll(path); err != nil {
				return err
			}
		}
		_ = os.Remove(p.RuntimeDir)
	}
	return nil
}
func firstLine(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.SplitN(string(data), "\n", 2)[0]
}
func exists(path string) bool { _, err := os.Lstat(path); return err == nil }
