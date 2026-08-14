package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/RokiLai/agent_sync_tool/internal/identity"
)

const (
	ManagedMarker      = "# ai-instructions managed file v1"
	RepoPathMarker     = "# ai-instructions repository path v1"
	AgentsURLMarker    = "# ai-instructions AGENTS URL v1"
	EnabledToolsMarker = "# ai-instructions enabled tools v1"
	BlockBegin         = "# >>> ai-instructions managed block >>>"
	BlockEnd           = "# <<< ai-instructions managed block <<<"
)

type Paths struct {
	HomeDir, RuntimeDir, ConfigDir, BinDir, CodexHome, RepositoryDir string
}

type Config struct {
	Paths
	RepositorySource string
	EnabledTools     []string
}

type LookupEnv func(string) (string, bool)

func Load(lookup LookupEnv, executable, command string) (Config, error) {
	home, ok := lookup("HOME")
	if !ok || home == "" {
		return Config{}, errors.New("HOME 未设置")
	}
	defaultRepository := filepath.Join(home, ".local/share/ai-instructions")
	c := Config{Paths: Paths{
		HomeDir:    home,
		RuntimeDir: valueOr(lookup, "AI_INSTRUCTIONS_RUNTIME_DIR", filepath.Join(home, ".local/share/ai-instructions-runtime")),
		ConfigDir:  valueOr(lookup, "AI_INSTRUCTIONS_CONFIG_DIR", filepath.Join(home, ".config/ai-instructions")),
		BinDir:     valueOr(lookup, "AI_INSTRUCTIONS_BIN_DIR", filepath.Join(home, ".local/bin")),
		CodexHome:  valueOr(lookup, "CODEX_HOME", filepath.Join(home, ".codex")),
	}}
	if repo, ok := lookup("AI_INSTRUCTIONS_REPO"); ok && repo != "" {
		c.RepositoryDir, c.RepositorySource = repo, "environment"
	} else if command == "install" {
		if repo, ok := DetectRepository(executable); ok {
			c.RepositoryDir, c.RepositorySource = repo, "script"
		}
	}
	if c.RepositoryDir == "" {
		if repo, err := ReadManagedValue(filepath.Join(c.ConfigDir, "repo-path"), RepoPathMarker); err == nil {
			c.RepositoryDir, c.RepositorySource = repo, "saved"
		} else if repo, ok := DetectRepository(executable); ok {
			c.RepositoryDir, c.RepositorySource = repo, "script"
		} else {
			c.RepositoryDir, c.RepositorySource = defaultRepository, "default"
		}
	}
	if (c.RepositorySource == "default" || c.RepositorySource == "saved") && c.RepositoryDir == defaultRepository && !isDir(filepath.Join(c.RepositoryDir, ".git")) {
		if info, err := os.Stat(filepath.Join(c.ConfigDir, "bin/ai-instructions")); err == nil && info.Mode().IsRegular() && info.Mode()&0111 != 0 {
			c.RepositorySource = "release"
		}
	}
	if info, err := os.Stat(c.RepositoryDir); err == nil && info.IsDir() {
		if resolved, err := filepath.EvalSymlinks(c.RepositoryDir); err == nil {
			c.RepositoryDir = resolved
		}
	}

	// 加载 EnabledTools
	if raw, err := ReadManagedValue(filepath.Join(c.ConfigDir, "enabled-tools"), EnabledToolsMarker); err == nil && raw != "" {
		var tools []string
		for _, part := range strings.Split(raw, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				tools = append(tools, part)
			}
		}
		c.EnabledTools = tools
	} else {
		// 尝试向下兼容探测已有受管符号链接
		c.EnabledTools = detectHistoricalTools(c)
		if len(c.EnabledTools) == 0 {
			c.EnabledTools = identity.DefaultToolKeys()
		}
	}

	return c, nil
}

func detectHistoricalTools(c Config) []string {
	runtimeFile := filepath.Join(c.RuntimeDir, "AGENTS.md")
	var enabled []string
	for _, tool := range identity.SupportedTools() {
		targetPath := tool.TargetPath(c.HomeDir, c.CodexHome)
		if got, err := os.Readlink(targetPath); err == nil && (got == runtimeFile || strings.HasSuffix(got, "AGENTS.md")) {
			enabled = append(enabled, tool.Key)
		}
	}
	return enabled
}

func valueOr(lookup LookupEnv, key, fallback string) string {
	if value, ok := lookup(key); ok && value != "" {
		return value
	}
	return fallback
}

func DetectRepository(executable string) (string, bool) {
	abs, err := filepath.Abs(executable)
	if err != nil {
		return "", false
	}
	dir := filepath.Dir(abs)
	for _, candidate := range []string{dir, filepath.Dir(dir)} {
		candidate, err = filepath.EvalSymlinks(candidate)
		if err != nil {
			continue
		}
		if isDir(filepath.Join(candidate, ".git")) && isRegular(filepath.Join(candidate, "bin/ai-instructions")) {
			return candidate, true
		}
	}
	return "", false
}

func ReadManagedValue(path, marker string) (string, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return "", errors.New("配置不存在或不是普通文件")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	if len(lines) < 2 || lines[0] != marker || lines[1] == "" {
		return "", errors.New("配置 marker 或值无效")
	}
	return lines[1], nil
}

func isDir(path string) bool { info, err := os.Stat(path); return err == nil && info.IsDir() }
func isRegular(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}
