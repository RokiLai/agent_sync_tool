package app

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RokiLai/agent_sync_tool/internal/config"
	"github.com/RokiLai/agent_sync_tool/internal/runtime"
)

func testDeps(t *testing.T) (Dependencies, *bytes.Buffer, *bytes.Buffer, string) {
	t.Helper()
	home := t.TempDir()
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	env := map[string]string{"HOME": home, "AI_INSTRUCTIONS_CONFIG_DIR": filepath.Join(home, "config"), "AI_INSTRUCTIONS_RUNTIME_DIR": filepath.Join(home, "runtime"), "AI_INSTRUCTIONS_BIN_DIR": filepath.Join(home, "bin"), "CODEX_HOME": filepath.Join(home, "codex")}
	return Dependencies{Stdout: stdout, Stderr: stderr, Executable: "/missing/aic", LookupEnv: func(key string) (string, bool) { value, ok := env[key]; return value, ok }}, stdout, stderr, home
}

func TestHelpAndVersion(t *testing.T) {
	for _, test := range []struct {
		args     []string
		expected string
	}{{nil, "用法：aic"}, {[]string{"--version"}, "ai-instructions 3.0.1\n"}, {[]string{"-V"}, "ai-instructions 3.0.1\n"}} {
		deps, stdout, _, _ := testDeps(t)
		if code := Main(context.Background(), test.args, deps); code != 0 || !strings.Contains(stdout.String(), test.expected) {
			t.Fatalf("args=%v code=%d output=%q", test.args, code, stdout.String())
		}
	}
}

func TestUnknownAndUnimplementedCommands(t *testing.T) {
	for _, command := range []string{"wat", "sync", "install", "upgrade", "uninstall"} {
		deps, _, stderr, _ := testDeps(t)
		if code := Main(context.Background(), []string{command}, deps); code == 0 || stderr.Len() == 0 {
			t.Fatalf("command=%s code=%d stderr=%q", command, code, stderr.String())
		}
	}
}

func TestSourceShowAndTest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("rules\n")) }))
	defer server.Close()
	deps, stdout, stderr, home := testDeps(t)
	deps.HTTPClient = server.Client()
	if err := os.MkdirAll(filepath.Join(home, "config"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "config/agents-url"), []byte(config.AgentsURLMarker+"\n"+server.URL+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if code := Main(context.Background(), []string{"source"}, deps); code != 0 || !strings.Contains(stdout.String(), server.URL) {
		t.Fatalf("show code=%d out=%q err=%q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := Main(context.Background(), []string{"source", "test"}, deps); code != 0 || !strings.Contains(stdout.String(), "内容版本：") {
		t.Fatalf("test code=%d out=%q err=%q", code, stdout.String(), stderr.String())
	}
}

func TestSourceTestFailureStaysOnStderr(t *testing.T) {
	deps, stdout, stderr, _ := testDeps(t)
	if code := Main(context.Background(), []string{"source", "test", "file:///tmp/a"}, deps); code == 0 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "HTTP(S)") {
		t.Fatalf("code=%d out=%q err=%q", code, stdout.String(), stderr.String())
	}
}

func TestArgumentAndSourceErrors(t *testing.T) {
	tests := [][]string{{"help", "extra"}, {"version", "extra"}, {"status", "extra"}, {"doctor", "extra"}, {"shell-init", "extra"}, {"source", "show", "extra"}, {"source", "test", "a", "b"}, {"source", "unknown"}, {"source", "set", "https://example.test/a"}}
	for _, args := range tests {
		deps, _, stderr, _ := testDeps(t)
		if code := Main(context.Background(), args, deps); code == 0 || stderr.Len() == 0 {
			t.Fatalf("args=%v code=%d stderr=%q", args, code, stderr.String())
		}
	}
}

func TestMissingHome(t *testing.T) {
	var stderr bytes.Buffer
	code := Main(context.Background(), []string{"help"}, Dependencies{Stderr: &stderr, LookupEnv: func(string) (string, bool) { return "", false }, Executable: "/missing/aic"})
	if code == 0 || !strings.Contains(stderr.String(), "HOME 未设置") {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
}

func TestSyncOutputAndCache(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("rules\n")) }))
	deps, stdout, stderr, home := testDeps(t)
	deps.HTTPClient = server.Client()
	if err := os.MkdirAll(filepath.Join(home, "config"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "config/agents-url"), []byte(config.AgentsURLMarker+"\n"+server.URL+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if code := Main(context.Background(), []string{"sync"}, deps); code != 0 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "AI 指令已同步") {
		t.Fatalf("code=%d out=%q err=%q", code, stdout.String(), stderr.String())
	}
	server.Close()
	stderr.Reset()
	if code := Main(context.Background(), []string{"sync"}, deps); code != 0 || !strings.Contains(stderr.String(), "最后一次成功部署") {
		t.Fatalf("cache code=%d err=%q", code, stderr.String())
	}
}

func TestSourceSetConfirmAndCancel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("new rules\n")) }))
	defer server.Close()
	for _, input := range []string{"\n", "y\n"} {
		deps, stdout, stderr, home := testDeps(t)
		deps.HTTPClient = server.Client()
		deps.Stdin = strings.NewReader(input)
		deps.IsTerminal = func() bool { return true }
		if err := os.MkdirAll(filepath.Join(home, "config"), 0755); err != nil {
			t.Fatal(err)
		}
		oldURL := "https://old.test/rules"
		path := filepath.Join(home, "config/agents-url")
		if err := os.WriteFile(path, []byte(config.AgentsURLMarker+"\n"+oldURL+"\n"), 0600); err != nil {
			t.Fatal(err)
		}
		code := Main(context.Background(), []string{"source", "set", server.URL}, deps)
		if code != 0 {
			t.Fatalf("input=%q code=%d out=%q err=%q", input, code, stdout.String(), stderr.String())
		}
		value, err := config.ReadManagedValue(path, config.AgentsURLMarker)
		if err != nil {
			t.Fatal(err)
		}
		if input == "\n" && value != oldURL {
			t.Fatalf("cancel changed source: %s", value)
		}
		if input == "y\n" && value != server.URL {
			t.Fatalf("confirm source: %s", value)
		}
	}
}

func TestCommitSourceRollsBackConfig(t *testing.T) {
	home := t.TempDir()
	cfg := filepath.Join(home, "config")
	run := filepath.Join(home, "runtime")
	if err := os.MkdirAll(cfg, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(run, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(cfg, "agents-url")
	old := config.AgentsURLMarker + "\nhttps://old.test/rules\n"
	if err := os.WriteFile(path, []byte(old), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(run, "AGENTS.md"), []byte("foreign"), 0600); err != nil {
		t.Fatal(err)
	}
	candidate, err := runtime.NewCandidate([]byte("new rules\n"))
	if err != nil {
		t.Fatal(err)
	}
	c := config.Config{Paths: config.Paths{ConfigDir: cfg, RuntimeDir: run}}
	if err := commitSource(context.Background(), c, "https://new.test/rules", candidate); err == nil {
		t.Fatal("expected conflict")
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != old {
		t.Fatalf("data=%q err=%v", data, err)
	}
}

func TestInstallDryRunAndUninstallNonTTY(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("rules\n")) }))
	defer server.Close()
	deps, stdout, stderr, home := testDeps(t)
	deps.HTTPClient = server.Client()
	repo := filepath.Join(home, "repo")
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	oldLookup := deps.LookupEnv
	deps.LookupEnv = func(key string) (string, bool) {
		if key == "AI_INSTRUCTIONS_REPO" {
			return repo, true
		}
		return oldLookup(key)
	}
	exe := filepath.Join(home, "aic")
	if err := os.WriteFile(exe, []byte("ai-instructions"), 0700); err != nil {
		t.Fatal(err)
	}
	deps.Executable = exe
	if code := Main(context.Background(), []string{"install", server.URL, "--shell", "none", "--dry-run"}, deps); code != 0 || !strings.Contains(stdout.String(), "dry-run 完成") {
		t.Fatalf("code=%d out=%q err=%q", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(home, "config")); !os.IsNotExist(err) {
		t.Fatalf("dry run wrote config: %v", err)
	}
	stderr.Reset()
	if code := Main(context.Background(), []string{"uninstall"}, deps); code == 0 || !strings.Contains(stderr.String(), "interactive terminal") {
		t.Fatalf("code=%d err=%q", code, stderr.String())
	}
}

func TestUninstallCancelAndExecute(t *testing.T) {
	for _, input := range []string{"\n", "y\nn\n"} {
		deps, stdout, stderr, home := testDeps(t)
		deps.IsTerminal = func() bool { return true }
		deps.Stdin = strings.NewReader(input)
		cfg := filepath.Join(home, "config")
		bin := filepath.Join(home, "bin")
		if err := os.MkdirAll(filepath.Join(cfg, "bin"), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(bin, 0755); err != nil {
			t.Fatal(err)
		}
		installed := filepath.Join(cfg, "bin/ai-instructions")
		if err := os.WriteFile(installed, []byte("ai-instructions"), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(installed, filepath.Join(bin, "aic")); err != nil {
			t.Fatal(err)
		}
		code := Main(context.Background(), []string{"uninstall"}, deps)
		if code != 0 {
			t.Fatalf("input=%q code=%d out=%q err=%q", input, code, stdout.String(), stderr.String())
		}
		_, err := os.Lstat(filepath.Join(bin, "aic"))
		if input == "\n" && err != nil {
			t.Fatal("cancel removed link")
		}
		if input != "\n" && !os.IsNotExist(err) {
			t.Fatalf("execute kept link: %v", err)
		}
	}
}
