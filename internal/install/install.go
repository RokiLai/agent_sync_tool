package install

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/RokiLai/agent_sync_tool/internal/config"
	"github.com/RokiLai/agent_sync_tool/internal/core"
	"github.com/RokiLai/agent_sync_tool/internal/managedfs"
)

type Options struct {
	URL, Shell string
	Tools      []string
	DryRun     bool
	Executable string
}
type Operation struct {
	Kind, Path, Target string
	Data               []byte
	Mode               os.FileMode
}
type Plan struct {
	Operations         []Operation
	ShellRC, ShellFile string
	Candidate          core.Candidate
	URL                string
}
type Installer struct {
	Config   config.Config
	Download func(context.Context, string) ([]byte, error)
}

func Parse(args []string, shellEnv string) (Options, error) {
	o := Options{Shell: "auto", Tools: []string{"codex", "claude", "agy"}}
	for len(args) > 0 {
		switch args[0] {
		case "--shell":
			if len(args) < 2 {
				return o, errors.New("--shell 缺少参数")
			}
			o.Shell = args[1]
			args = args[2:]
		case "--tools":
			if len(args) < 2 {
				return o, errors.New("--tools 缺少参数")
			}
			o.Tools = strings.Split(args[1], ",")
			args = args[2:]
		case "--dry-run":
			o.DryRun = true
			args = args[1:]
		default:
			if strings.HasPrefix(args[0], "http://") || strings.HasPrefix(args[0], "https://") {
				if o.URL != "" {
					return o, errors.New("install 只能提供一个 AGENTS.md 链接")
				}
				o.URL = args[0]
				args = args[1:]
			} else {
				return o, fmt.Errorf("install 不支持的参数：%s", args[0])
			}
		}
	}
	if o.Shell == "auto" {
		base := filepath.Base(shellEnv)
		if base == "zsh" || base == "bash" {
			o.Shell = base
		} else {
			o.Shell = "none"
		}
	}
	if o.Shell != "zsh" && o.Shell != "bash" && o.Shell != "none" {
		return o, fmt.Errorf("不支持的 Shell：%s", o.Shell)
	}
	seen := map[string]bool{}
	for _, tool := range o.Tools {
		if tool != "codex" && tool != "claude" && tool != "agy" {
			return o, fmt.Errorf("不支持的工具：%s", tool)
		}
		seen[tool] = true
	}
	if len(seen) != len(o.Tools) {
		return o, errors.New("工具列表包含重复项")
	}
	return o, nil
}

func (i Installer) Prepare(ctx context.Context, o Options) (Plan, error) {
	if o.URL == "" {
		return Plan{}, errors.New("非交互环境必须在安装命令后提供 AGENTS.md 链接")
	}
	normalizedURL, _, err := core.NormalizeURL(o.URL)
	if err != nil {
		return Plan{}, err
	}
	o.URL = normalizedURL
	data, err := i.Download(ctx, o.URL)
	if err != nil {
		return Plan{}, errors.New("首次同步失败")
	}
	candidate, err := core.NewCandidate(data)
	if err != nil {
		return Plan{}, err
	}
	c := i.Config
	installed := filepath.Join(c.ConfigDir, "bin/ai-instructions")
	runtimeFile := filepath.Join(c.RuntimeDir, "AGENTS.md")
	p := Plan{Candidate: candidate, URL: o.URL, ShellRC: core.RCPath(c.HomeDir, o.Shell), ShellFile: filepath.Join(c.ConfigDir, "shell-integration.sh")}
	exe, err := os.ReadFile(o.Executable)
	if err != nil {
		return Plan{}, err
	}
	p.Operations = append(p.Operations, Operation{"file", installed, "", exe, 0700}, Operation{"link", filepath.Join(c.BinDir, "ai-instructions"), installed, nil, 0}, Operation{"link", filepath.Join(c.BinDir, "aic"), installed, nil, 0}, Operation{"file", filepath.Join(c.ConfigDir, "repo-path"), "", []byte(config.RepoPathMarker + "\n" + c.RepositoryDir + "\n"), 0600}, Operation{"file", filepath.Join(c.ConfigDir, "agents-url"), "", []byte(config.AgentsURLMarker + "\n" + o.URL + "\n"), 0600})
	for _, tool := range o.Tools {
		path := map[string]string{"codex": filepath.Join(c.CodexHome, "AGENTS.md"), "claude": filepath.Join(c.HomeDir, ".claude/CLAUDE.md"), "agy": filepath.Join(c.HomeDir, ".gemini/GEMINI.md")}[tool]
		p.Operations = append(p.Operations, Operation{"link", path, runtimeFile, nil, 0})
	}
	if o.Shell != "none" {
		p.Operations = append(p.Operations, Operation{"file", p.ShellFile, "", []byte(core.ShellInit(c)), 0600})
	}
	if err := Preflight(p, c); err != nil {
		return Plan{}, err
	}
	return p, nil
}

func Preflight(p Plan, c config.Config) error {
	for _, op := range p.Operations {
		info, err := os.Lstat(op.Path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		switch op.Kind {
		case "link":
			if info.Mode()&os.ModeSymlink == 0 {
				return fmt.Errorf("路径已存在且不是受管符号链接：%s", op.Path)
			}
			target, _ := os.Readlink(op.Path)
			if target != op.Target {
				return fmt.Errorf("符号链接已指向其他位置：%s", op.Path)
			}
		case "file":
			if info.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("配置文件已存在且不受本工具管理：%s", op.Path)
			}
			data, _ := os.ReadFile(op.Path)
			if !managedFile(op.Path, data, c) {
				return fmt.Errorf("配置文件已存在且不受本工具管理：%s", op.Path)
			}
		}
	}
	if err := core.ValidateRC(p.ShellRC); err != nil {
		return err
	}
	return preflightRuntime(c.RuntimeDir)
}
func managedFile(path string, data []byte, c config.Config) bool {
	first := strings.SplitN(string(data), "\n", 2)[0]
	if path == filepath.Join(c.ConfigDir, "repo-path") {
		return first == config.RepoPathMarker
	}
	if path == filepath.Join(c.ConfigDir, "agents-url") {
		return first == config.AgentsURLMarker
	}
	if path == filepath.Join(c.ConfigDir, "shell-integration.sh") {
		return first == config.ManagedMarker
	}
	if path == filepath.Join(c.ConfigDir, "bin/ai-instructions") {
		return strings.Contains(string(data), "ai-instructions")
	}
	return false
}
func preflightRuntime(dir string) error {
	current := filepath.Join(dir, "current")
	if info, err := os.Lstat(current); err == nil {
		if info.Mode()&os.ModeSymlink == 0 {
			return errors.New("runtime current 已存在且不是符号链接")
		}
		target, _ := os.Readlink(current)
		if !strings.HasPrefix(target, "versions/") {
			return errors.New("runtime current 是非预期链接")
		}
		for path, expected := range map[string]string{filepath.Join(dir, "AGENTS.md"): "current/AGENTS.md", filepath.Join(dir, "REVISION"): "current/REVISION"} {
			info, err := os.Lstat(path)
			if err != nil || info.Mode()&os.ModeSymlink == 0 {
				return errors.New("runtime 兼容链接不完整")
			}
			got, _ := os.Readlink(path)
			if got != expected {
				return errors.New("runtime 兼容链接目标冲突")
			}
		}
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	agents, revision := filepath.Join(dir, "AGENTS.md"), filepath.Join(dir, "REVISION")
	aInfo, aErr := os.Lstat(agents)
	rInfo, rErr := os.Lstat(revision)
	aExists := aErr == nil
	rExists := rErr == nil
	if aExists || rExists {
		if !aExists || !rExists || aInfo.Mode()&os.ModeSymlink != 0 || rInfo.Mode()&os.ModeSymlink != 0 || aInfo.Size() == 0 || rInfo.Size() == 0 {
			return errors.New("旧 runtime 布局不完整或包含非预期链接")
		}
	}
	return nil
}
func Execute(p Plan, c config.Config) error {
	if err := (core.Publisher{Dir: c.RuntimeDir}).Publish(p.Candidate); err != nil {
		return err
	}
	for _, op := range p.Operations {
		switch op.Kind {
		case "file":
			if err := managedfs.AtomicWrite(op.Path, op.Data, op.Mode); err != nil {
				return err
			}
		case "link":
			if err := os.MkdirAll(filepath.Dir(op.Path), 0755); err != nil {
				return err
			}
			if err := managedfs.EnsureSymlink(op.Path, op.Target); err != nil {
				return err
			}
		}
	}
	return core.InstallRC(p.ShellRC, p.ShellFile)
}
