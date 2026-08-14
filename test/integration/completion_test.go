package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestShellInitScriptSyntaxAndCompletions(t *testing.T) {
	binary := buildAIC(t)
	home := t.TempDir()

	cmd := exec.Command(binary, "shell-init")
	cmd.Env = append(os.Environ(), "HOME="+home)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("agentsync shell-init failed: %v", err)
	}

	content := string(out)

	// Check essential completion elements
	for _, expected := range []string{
		"_agentsync_zsh_complete()",
		"_agentsync_bash_complete()",
		"compdef _agentsync_zsh_complete agentsync",
		"complete -F _agentsync_bash_complete agentsync",
		"install sync source upgrade status doctor shell-init uninstall version help",
	} {
		if !strings.Contains(content, expected) {
			t.Errorf("shell-init output missing %q", expected)
		}
	}

	scriptPath := filepath.Join(t.TempDir(), "shell-integration.sh")
	if err := os.WriteFile(scriptPath, out, 0600); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	// Validate with bash -n if bash is available
	if bashPath, err := exec.LookPath("bash"); err == nil {
		bashCmd := exec.Command(bashPath, "-n", scriptPath)
		if bashOut, err := bashCmd.CombinedOutput(); err != nil {
			t.Fatalf("bash -n syntax validation failed: %v\nOutput: %s", err, bashOut)
		}
	}

	// Validate with zsh -n if zsh is available
	if zshPath, err := exec.LookPath("zsh"); err == nil {
		zshCmd := exec.Command(zshPath, "-n", scriptPath)
		if zshOut, err := zshCmd.CombinedOutput(); err != nil {
			t.Fatalf("zsh -n syntax validation failed: %v\nOutput: %s", err, zshOut)
		}
	}
}
