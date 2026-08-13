package integration_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestBinaryInstallDryRunAndInstall(t *testing.T) {
	binary := buildAIC(t)
	home := t.TempDir()
	repo := filepath.Join(home, "repo")
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("rules\n")) }))
	defer server.Close()
	env := append(os.Environ(), "HOME="+home, "AI_INSTRUCTIONS_REPO="+repo, "AI_INSTRUCTIONS_CONFIG_DIR="+filepath.Join(home, "config"), "AI_INSTRUCTIONS_RUNTIME_DIR="+filepath.Join(home, "runtime"), "AI_INSTRUCTIONS_BIN_DIR="+filepath.Join(home, "bin"), "CODEX_HOME="+filepath.Join(home, "codex"))
	run := func(extra ...string) (string, error) {
		args := append([]string{"install", server.URL, "--shell", "none", "--tools", "codex"}, extra...)
		cmd := exec.Command(binary, args...)
		cmd.Env = env
		out, err := cmd.CombinedOutput()
		return string(out), err
	}
	out, err := run("--dry-run")
	if err != nil || !strings.Contains(out, "dry-run 完成") {
		t.Fatalf("out=%s err=%v", out, err)
	}
	if _, err := os.Stat(filepath.Join(home, "config")); !os.IsNotExist(err) {
		t.Fatalf("dry run wrote: %v", err)
	}
	out, err = run()
	if err != nil {
		t.Fatalf("out=%s err=%v", out, err)
	}
	for _, path := range []string{filepath.Join(home, "bin/aic"), filepath.Join(home, "codex/AGENTS.md"), filepath.Join(home, "runtime/current"), filepath.Join(home, "config/agents-url")} {
		if _, err := os.Lstat(path); err != nil {
			t.Fatalf("missing %s: %v", path, err)
		}
	}
	out, err = run()
	if err != nil {
		t.Fatalf("reinstall out=%s err=%v", out, err)
	}
}
func TestBinaryInstallConflictZeroWrite(t *testing.T) {
	binary := buildAIC(t)
	home := t.TempDir()
	repo := filepath.Join(home, "repo")
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0755); err != nil {
		t.Fatal(err)
	}
	foreign := filepath.Join(home, ".claude/CLAUDE.md")
	if err := os.WriteFile(foreign, []byte("user"), 0600); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("rules")) }))
	defer server.Close()
	cmd := exec.Command(binary, "install", server.URL, "--shell", "none", "--tools", "claude")
	cmd.Env = append(os.Environ(), "HOME="+home, "AI_INSTRUCTIONS_REPO="+repo)
	if err := cmd.Run(); err == nil {
		t.Fatal("expected conflict")
	}
	if _, err := os.Stat(filepath.Join(home, ".config")); !os.IsNotExist(err) {
		t.Fatalf("config written: %v", err)
	}
	data, _ := os.ReadFile(foreign)
	if string(data) != "user" {
		t.Fatal("foreign changed")
	}
}
