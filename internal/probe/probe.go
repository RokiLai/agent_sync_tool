package probe

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/RokiLai/agent_sync_tool/internal/config"
	"github.com/RokiLai/agent_sync_tool/internal/core"
	"github.com/RokiLai/agent_sync_tool/internal/identity"
)

type ToolEnvironment struct {
	DetectedTools []identity.ToolSpec
	DetectedKeys  []string
	AllSupported  []identity.ToolSpec
}

func DetectTools(lookPath func(string) (string, error), homeDir, codexHome string) ToolEnvironment {
	all := identity.SupportedTools()
	var detected []identity.ToolSpec
	var keys []string

	for _, tool := range all {
		hasBinary := false
		if lookPath != nil {
			if _, err := lookPath(tool.BinaryName); err == nil {
				hasBinary = true
			}
		}

		hasDir := false
		var dirPath string
		if tool.Key == "codex" {
			dirPath = codexHome
		} else {
			dirPath = filepath.Join(homeDir, tool.HomeSubDir)
		}
		if info, err := os.Stat(dirPath); err == nil && info.IsDir() {
			hasDir = true
		}

		if hasBinary || hasDir {
			detected = append(detected, tool)
			keys = append(keys, tool.Key)
		}
	}

	return ToolEnvironment{
		DetectedTools: detected,
		DetectedKeys:  keys,
		AllSupported:  all,
	}
}

func DetectShell(lookupEnv config.LookupEnv, homeDir string) (string, string) {
	if lookupEnv == nil {
		lookupEnv = os.LookupEnv
	}

	shellEnv, _ := lookupEnv("SHELL")
	base := filepath.Base(shellEnv)

	if base == "zsh" || base == "bash" {
		return base, core.RCPath(homeDir, base)
	}

	if zshVer, ok := lookupEnv("ZSH_VERSION"); ok && zshVer != "" {
		return "zsh", core.RCPath(homeDir, "zsh")
	}
	if bashVer, ok := lookupEnv("BASH_VERSION"); ok && bashVer != "" {
		return "bash", core.RCPath(homeDir, "bash")
	}

	return "none", ""
}

func DetectHistoricalEnabledTools(c config.Config) []string {
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
