package probe

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/RokiLai/agent_sync_tool/internal/config"
)

func TestDetectTools(t *testing.T) {
	tempDir := t.TempDir()
	homeDir := filepath.Join(tempDir, "home")
	codexHome := filepath.Join(homeDir, ".codex")
	_ = os.MkdirAll(homeDir, 0755)
	_ = os.MkdirAll(codexHome, 0755)

	mockLookPath := func(cmd string) (string, error) {
		if cmd == "claude" {
			return "/usr/local/bin/claude", nil
		}
		return "", errors.New("not found")
	}

	env := DetectTools(mockLookPath, homeDir, codexHome)
	// Should detect: codex (by dir) and claude (by binary)
	if len(env.DetectedTools) != 2 {
		t.Fatalf("expected 2 detected tools, got %d (%v)", len(env.DetectedTools), env.DetectedKeys)
	}

	foundCodex, foundClaude := false, false
	for _, k := range env.DetectedKeys {
		if k == "codex" {
			foundCodex = true
		}
		if k == "claude" {
			foundClaude = true
		}
	}
	if !foundCodex || !foundClaude {
		t.Fatalf("expected codex and claude detected, got %v", env.DetectedKeys)
	}
}

func TestDetectShell(t *testing.T) {
	home := "/Users/test"
	envMap := map[string]string{
		"SHELL": "/bin/zsh",
	}
	lookup := func(k string) (string, bool) {
		v, ok := envMap[k]
		return v, ok
	}

	shell, rc := DetectShell(lookup, home)
	if shell != "zsh" || rc != filepath.Join(home, ".zshrc") {
		t.Fatalf("expected zsh and .zshrc, got %s and %s", shell, rc)
	}

	// Test fallback to ZSH_VERSION
	delete(envMap, "SHELL")
	envMap["ZSH_VERSION"] = "5.9"
	shell, rc = DetectShell(lookup, home)
	if shell != "zsh" || rc != filepath.Join(home, ".zshrc") {
		t.Fatalf("expected zsh and .zshrc via ZSH_VERSION, got %s and %s", shell, rc)
	}

	// Test unknown
	delete(envMap, "ZSH_VERSION")
	shell, rc = DetectShell(lookup, home)
	if shell != "none" || rc != "" {
		t.Fatalf("expected none, got %s", shell)
	}
}

func TestDetectHistoricalEnabledTools(t *testing.T) {
	tempDir := t.TempDir()
	homeDir := filepath.Join(tempDir, "home")
	runtimeDir := filepath.Join(tempDir, "runtime")
	codexHome := filepath.Join(tempDir, "codex")
	_ = os.MkdirAll(filepath.Join(homeDir, ".claude"), 0755)
	_ = os.MkdirAll(runtimeDir, 0755)

	runtimeFile := filepath.Join(runtimeDir, "AGENTS.md")
	claudeTarget := filepath.Join(homeDir, ".claude/CLAUDE.md")
	_ = os.Symlink(runtimeFile, claudeTarget)

	cfg := config.Config{
		Paths: config.Paths{
			HomeDir:    homeDir,
			RuntimeDir: runtimeDir,
			CodexHome:  codexHome,
		},
	}

	enabled := DetectHistoricalEnabledTools(cfg)
	if len(enabled) != 1 || enabled[0] != "claude" {
		t.Fatalf("expected [claude], got %v", enabled)
	}
}
