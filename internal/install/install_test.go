package install

import (
	"context"
	"github.com/RokiLai/agent_sync_tool/internal/config"
	"os"
	"path/filepath"
	"testing"
)

func fixture(t *testing.T) (Installer, Options, config.Config) {
	t.Helper()
	home := t.TempDir()
	repo := filepath.Join(home, "repo")
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(home, "aic")
	if err := os.WriteFile(exe, []byte("ai-instructions binary"), 0700); err != nil {
		t.Fatal(err)
	}
	c := config.Config{Paths: config.Paths{HomeDir: home, RuntimeDir: filepath.Join(home, "runtime"), ConfigDir: filepath.Join(home, "config"), BinDir: filepath.Join(home, "bin"), CodexHome: filepath.Join(home, "codex"), RepositoryDir: repo}}
	i := Installer{Config: c, Download: func(context.Context, string) ([]byte, error) { return []byte("rules\n"), nil }}
	return i, Options{URL: "https://example.test/rules", Shell: "zsh", Tools: []string{"codex", "claude", "agy"}, Executable: exe}, c
}
func TestParse(t *testing.T) {
	o, err := Parse([]string{"https://x.test/a", "--shell", "auto", "--tools", "codex,agy", "--dry-run"}, "/bin/zsh")
	if err != nil || o.Shell != "zsh" || !o.DryRun || len(o.Tools) != 2 {
		t.Fatalf("o=%#v err=%v", o, err)
	}
	if _, err := Parse([]string{"--tools", "bad"}, ""); err == nil {
		t.Fatal("expected error")
	}
}
func TestPrepareExecuteIdempotent(t *testing.T) {
	i, o, c := fixture(t)
	p, err := i.Prepare(context.Background(), o)
	if err != nil {
		t.Fatal(err)
	}
	if err := Execute(p, c); err != nil {
		t.Fatal(err)
	}
	p, err = i.Prepare(context.Background(), o)
	if err != nil {
		t.Fatal(err)
	}
	if err := Execute(p, c); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Readlink(filepath.Join(c.BinDir, "aic")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Readlink(filepath.Join(c.CodexHome, "AGENTS.md")); err != nil {
		t.Fatal(err)
	}
}
func TestPreflightConflictZeroWrite(t *testing.T) {
	i, o, c := fixture(t)
	if err := os.MkdirAll(filepath.Join(c.HomeDir, ".claude"), 0755); err != nil {
		t.Fatal(err)
	}
	foreign := filepath.Join(c.HomeDir, ".claude/CLAUDE.md")
	if err := os.WriteFile(foreign, []byte("user"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := i.Prepare(context.Background(), o); err == nil {
		t.Fatal("expected conflict")
	}
	if _, err := os.Stat(c.ConfigDir); !os.IsNotExist(err) {
		t.Fatalf("config created: %v", err)
	}
	data, _ := os.ReadFile(foreign)
	if string(data) != "user" {
		t.Fatal("foreign changed")
	}
}

func TestPrepareAcceptsReleaseModeAndRejectsIncompleteRuntime(t *testing.T) {
	i, o, c := fixture(t)
	if err := os.RemoveAll(filepath.Join(c.RepositoryDir, ".git")); err != nil {
		t.Fatal(err)
	}
	if _, err := i.Prepare(context.Background(), o); err != nil {
		t.Fatalf("release mode rejected: %v", err)
	}
	if err := os.MkdirAll(c.RuntimeDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(c.RuntimeDir, "AGENTS.md"), []byte("rules"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := i.Prepare(context.Background(), o); err == nil {
		t.Fatal("incomplete runtime accepted")
	}
}
