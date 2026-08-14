package uninstall

import (
	"bytes"
	"github.com/RokiLai/agent_sync_tool/internal/config"
	"github.com/RokiLai/agent_sync_tool/internal/integration"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildExecuteImmutablePlan(t *testing.T) {
	home := t.TempDir()
	cfg := filepath.Join(home, "config")
	bin := filepath.Join(home, "bin")
	run := filepath.Join(home, "runtime")
	for _, d := range []string{cfg, bin, filepath.Join(home, "codex")} {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatal(err)
		}
	}
	installed := filepath.Join(cfg, "bin/ai-instructions")
	if err := os.MkdirAll(filepath.Dir(installed), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(installed, []byte("ai-instructions"), 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"agentsync", "aic", "ai-instructions"} {
		if err := os.Symlink(installed, filepath.Join(bin, name)); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(cfg, "agents-url"), []byte(config.AgentsURLMarker+"\nhttps://x\n"), 0600); err != nil {
		t.Fatal(err)
	}
	rc := filepath.Join(home, ".zshrc")
	if err := integration.InstallRC(rc, filepath.Join(cfg, "shell-integration.sh")); err != nil {
		t.Fatal(err)
	}
	c := config.Config{Paths: config.Paths{HomeDir: home, RuntimeDir: run, ConfigDir: cfg, BinDir: bin, CodexHome: filepath.Join(home, "codex")}}
	p := Build(c, "zsh")
	var out bytes.Buffer
	Print(&out, p)
	if !strings.Contains(out.String(), "即将执行") {
		t.Fatal(out.String())
	}
	foreign := filepath.Join(bin, "later")
	if err := os.WriteFile(foreign, []byte("user"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := Execute(p, false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(foreign); err != nil {
		t.Fatal("later file deleted")
	}
	for _, name := range []string{"agentsync", "aic", "ai-instructions"} {
		if _, err := os.Lstat(filepath.Join(bin, name)); !os.IsNotExist(err) {
			t.Fatalf("%s remains", name)
		}
	}
}
