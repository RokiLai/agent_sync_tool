package diagnose

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RokiLai/agent_sync_tool/internal/config"
	"github.com/RokiLai/agent_sync_tool/internal/runtime"
)

func TestStatusReadsManagedState(t *testing.T) {
	home := t.TempDir()
	repo := filepath.Join(home, "repo")
	cfg := filepath.Join(home, "config")
	run := filepath.Join(home, "runtime")
	for _, dir := range []string{filepath.Join(repo, ".git"), cfg, run} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	data := []byte("rules\n")
	rev := runtime.Revision(data)
	if err := os.WriteFile(filepath.Join(cfg, "agents-url"), []byte(config.AgentsURLMarker+"\nhttps://example.test/rules\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(run, "AGENTS.md"), data, 0444); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(run, "REVISION"), []byte(rev+"\n"), 0444); err != nil {
		t.Fatal(err)
	}
	c := config.Config{Paths: config.Paths{HomeDir: home, RuntimeDir: run, ConfigDir: cfg, CodexHome: filepath.Join(home, "codex"), RepositoryDir: repo}, RepositorySource: "environment"}
	var out bytes.Buffer
	Status(&out, c, Dependencies{GitRev: func(string) string { return "deadbeef" }})
	for _, expected := range []string{"[OK] 仓库：", "仓库路径来源：environment", "deadbeef", "https://example.test/rules", "已部署版本：" + rev} {
		if !strings.Contains(out.String(), expected) {
			t.Errorf("missing %q in %s", expected, out.String())
		}
	}
}

func TestDoctorFailureExit(t *testing.T) {
	home := t.TempDir()
	c := config.Config{Paths: config.Paths{HomeDir: home, RuntimeDir: filepath.Join(home, "runtime"), ConfigDir: filepath.Join(home, "config"), BinDir: filepath.Join(home, "bin"), CodexHome: filepath.Join(home, "codex"), RepositoryDir: filepath.Join(home, "repo")}}
	var out bytes.Buffer
	ok := Doctor(&out, c, Dependencies{LookPath: func(string) (string, error) { return "", errors.New("missing") }}, "/bin/zsh")
	if ok || !strings.Contains(out.String(), "个失败") {
		t.Fatalf("ok=%v output=%s", ok, out.String())
	}
}

func TestDoctorHealthyManagedState(t *testing.T) {
	home := t.TempDir()
	repo := filepath.Join(home, "repo")
	cfg := filepath.Join(home, "config")
	run := filepath.Join(home, "runtime")
	bin := filepath.Join(home, "bin")
	codex := filepath.Join(home, "codex")
	for _, dir := range []string{filepath.Join(repo, ".git"), filepath.Join(cfg, "bin"), run, bin, codex, filepath.Join(home, ".claude"), filepath.Join(home, ".gemini")} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	data := []byte("rules\n")
	rev := runtime.Revision(data)
	files := map[string][]byte{
		filepath.Join(cfg, "repo-path"):            []byte(config.RepoPathMarker + "\n" + repo + "\n"),
		filepath.Join(cfg, "agents-url"):           []byte(config.AgentsURLMarker + "\nhttps://example.test/rules\n"),
		filepath.Join(cfg, "shell-integration.sh"): []byte(config.ManagedMarker + "\n"),
		filepath.Join(run, "AGENTS.md"):            data,
		filepath.Join(run, "REVISION"):             []byte(rev + "\n"),
		filepath.Join(home, ".zshrc"):              []byte(config.BlockBegin + "\n" + config.BlockEnd + "\n"),
		filepath.Join(cfg, "bin/agentsync"):        []byte("#!/bin/sh\n"),
	}
	for path, content := range files {
		mode := os.FileMode(0600)
		if path == filepath.Join(cfg, "bin/agentsync") {
			mode = 0700
		}
		if err := os.WriteFile(path, content, mode); err != nil {
			t.Fatal(err)
		}
	}
	runtimeFile := filepath.Join(run, "AGENTS.md")
	installed := filepath.Join(cfg, "bin/agentsync")
	for path, target := range map[string]string{filepath.Join(codex, "AGENTS.md"): runtimeFile, filepath.Join(home, ".claude/CLAUDE.md"): runtimeFile, filepath.Join(home, ".gemini/GEMINI.md"): runtimeFile, filepath.Join(bin, "agentsync"): installed} {
		if err := os.Symlink(target, path); err != nil {
			t.Fatal(err)
		}
	}
	c := config.Config{Paths: config.Paths{HomeDir: home, RuntimeDir: run, ConfigDir: cfg, BinDir: bin, CodexHome: codex, RepositoryDir: repo}}
	var out bytes.Buffer
	ok := Doctor(&out, c, Dependencies{LookPath: func(name string) (string, error) { return "/bin/" + name, nil }}, "/bin/zsh")
	if !ok || !strings.Contains(out.String(), "0 个失败") {
		t.Fatalf("ok=%v output=%s", ok, out.String())
	}
}

func TestStatusAndDoctorSuggestInstallForUnmanagedDetectedTools(t *testing.T) {
	home := t.TempDir()
	repo := filepath.Join(home, "repo")
	cfg := filepath.Join(home, "config")
	run := filepath.Join(home, "runtime")
	bin := filepath.Join(home, "bin")
	codex := filepath.Join(home, "codex")
	claudeDir := filepath.Join(home, ".claude")
	for _, dir := range []string{filepath.Join(repo, ".git"), filepath.Join(cfg, "bin"), run, bin, codex, claudeDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	data := []byte("rules\n")
	rev := runtime.Revision(data)
	if err := os.WriteFile(filepath.Join(cfg, "agents-url"), []byte(config.AgentsURLMarker+"\nhttps://example.test/rules\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg, "repo-path"), []byte(config.RepoPathMarker+"\n"+repo+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg, "shell-integration.sh"), []byte(config.ManagedMarker+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(run, "AGENTS.md"), data, 0444); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(run, "REVISION"), []byte(rev+"\n"), 0444); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".zshrc"), []byte(config.BlockBegin+"\n"+config.BlockEnd+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg, "bin/agentsync"), []byte("#!/bin/sh\n"), 0700); err != nil {
		t.Fatal(err)
	}
	runtimeFile := filepath.Join(run, "AGENTS.md")
	installed := filepath.Join(cfg, "bin/agentsync")
	if err := os.Symlink(runtimeFile, filepath.Join(codex, "AGENTS.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(installed, filepath.Join(bin, "agentsync")); err != nil {
		t.Fatal(err)
	}

	c := config.Config{
		Paths:        config.Paths{HomeDir: home, RuntimeDir: run, ConfigDir: cfg, BinDir: bin, CodexHome: codex, RepositoryDir: repo},
		EnabledTools: []string{"codex"},
	}

	deps := Dependencies{
		LookPath: func(name string) (string, error) {
			if name == "claude" || name == "git" || name == "curl" || name == "codex" {
				return "/bin/" + name, nil
			}
			return "", errors.New("missing")
		},
		GitRev: func(string) string { return "deadbeef" },
	}

	var statusOut bytes.Buffer
	Status(&statusOut, c, deps)
	if !strings.Contains(statusOut.String(), "检测到 Claude 已安装但未接入规则；运行 agentsync install 可自动接入") {
		t.Fatalf("expected status to suggest install for claude, got:\n%s", statusOut.String())
	}

	var doctorOut bytes.Buffer
	Doctor(&doctorOut, c, deps, "/bin/zsh")
	if !strings.Contains(doctorOut.String(), "检测到未接入规则的 AI 工具（Claude）；运行 agentsync install 可自动接入") {
		t.Fatalf("expected doctor to suggest install for claude, got:\n%s", doctorOut.String())
	}
}
