package app

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/RokiLai/agent_sync_tool/internal/config"
	"github.com/RokiLai/agent_sync_tool/internal/core"
	"github.com/RokiLai/agent_sync_tool/internal/runtime"
)

func testDeps(t *testing.T) (Dependencies, *bytes.Buffer, *bytes.Buffer, string) {
	t.Helper()
	home := t.TempDir()
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	env := map[string]string{"HOME": home, "AI_INSTRUCTIONS_CONFIG_DIR": filepath.Join(home, "config"), "AI_INSTRUCTIONS_RUNTIME_DIR": filepath.Join(home, "runtime"), "AI_INSTRUCTIONS_BIN_DIR": filepath.Join(home, "bin"), "CODEX_HOME": filepath.Join(home, "codex")}
	return Dependencies{Stdout: stdout, Stderr: stderr, Executable: "/missing/aic", LookupEnv: func(key string) (string, bool) { value, ok := env[key]; return value, ok }}, stdout, stderr, home
}

func TestUpgradeChecksConfirmsAndRendersProgress(t *testing.T) {
	origVersion := Version
	Version = "3.2.2"
	defer func() { Version = origVersion }()

	deps, stdout, stderr, home := testDeps(t)
	installed := filepath.Join(home, "config/bin/ai-instructions")
	if err := os.MkdirAll(filepath.Dir(installed), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(installed, []byte("#!/bin/sh\nprintf 'ai-instructions "+Version+"\\n'\n"), 0700); err != nil {
		t.Fatal(err)
	}
	candidate := []byte("#!/bin/sh\nprintf 'ai-instructions 3.4.0\\n'\n")
	sum := sha256.Sum256(candidate)
	artifact, err := core.CurrentArtifact()
	if err != nil {
		t.Fatal(err)
	}
	var artifactRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/latest/download/checksums.txt" {
			http.Redirect(w, r, "/download/v3.4.0/checksums.txt", http.StatusFound)
			return
		}
		if filepath.Base(r.URL.Path) == "checksums.txt" {
			fmt.Fprintf(w, "%x  %s\n", sum, artifact)
			return
		}
		artifactRequests.Add(1)
		w.Header().Set("Content-Length", fmt.Sprint(len(candidate)))
		_, _ = w.Write(candidate)
	}))
	defer server.Close()
	oldLookup := deps.LookupEnv
	deps.LookupEnv = func(key string) (string, bool) {
		switch key {
		case "AIC_RELEASE_BASE_URL":
			return server.URL, true
		case "AIC_VERSION":
			return "latest", true
		}
		return oldLookup(key)
	}
	deps.HTTPClient = server.Client()
	deps.Stdin = strings.NewReader("y\n")
	deps.IsTerminal = func() bool { return true }
	deps.IsOutputTerminal = func() bool { return true }
	if code := Main(context.Background(), []string{"upgrade"}, deps); code != 0 {
		t.Fatalf("code=%d out=%q err=%q", code, stdout.String(), stderr.String())
	}
	got, _ := os.ReadFile(installed)
	if string(got) != string(candidate) || artifactRequests.Load() != 1 {
		t.Fatalf("artifactRequests=%d installed=%q", artifactRequests.Load(), got)
	}
	for _, want := range []string{"当前版本：v" + Version, "最新版本：v3.4.0", "100%", "升级成功：v" + Version + " → v3.4.0"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("out=%q missing=%q", stdout.String(), want)
		}
	}
}

func TestUpgradeCancelDoesNotDownloadArtifact(t *testing.T) {
	origVersion := Version
	Version = "3.2.2"
	defer func() { Version = origVersion }()

	deps, stdout, stderr, home := testDeps(t)
	installed := filepath.Join(home, "config/bin/ai-instructions")
	if err := os.MkdirAll(filepath.Dir(installed), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(installed, []byte("old"), 0700); err != nil {
		t.Fatal(err)
	}
	var artifactRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/latest/download/checksums.txt" {
			http.Redirect(w, r, "/download/v3.3.0/checksums.txt", http.StatusFound)
			return
		}
		if filepath.Base(r.URL.Path) == "checksums.txt" {
			fmt.Fprintln(w, "abc  ignored")
			return
		}
		artifactRequests.Add(1)
	}))
	defer server.Close()
	oldLookup := deps.LookupEnv
	deps.LookupEnv = func(key string) (string, bool) {
		if key == "AIC_RELEASE_BASE_URL" {
			return server.URL, true
		}
		return oldLookup(key)
	}
	deps.HTTPClient = server.Client()
	deps.Stdin = strings.NewReader("n\n")
	deps.IsTerminal = func() bool { return true }
	if code := Main(context.Background(), []string{"upgrade"}, deps); code != 0 || artifactRequests.Load() != 0 || !strings.Contains(stdout.String(), "已取消升级") {
		t.Fatalf("code=%d requests=%d out=%q err=%q", code, artifactRequests.Load(), stdout.String(), stderr.String())
	}
}

func TestUpgradeCurrentVersionSkipsArtifact(t *testing.T) {
	origVersion := Version
	Version = "3.2.2"
	defer func() { Version = origVersion }()

	deps, stdout, stderr, home := testDeps(t)
	installed := filepath.Join(home, "config/bin/ai-instructions")
	if err := os.MkdirAll(filepath.Dir(installed), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(installed, []byte("old"), 0700); err != nil {
		t.Fatal(err)
	}
	var artifactRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if filepath.Base(r.URL.Path) == "checksums.txt" {
			fmt.Fprintln(w, "abc  ignored")
			return
		}
		artifactRequests.Add(1)
	}))
	defer server.Close()
	oldLookup := deps.LookupEnv
	deps.LookupEnv = func(key string) (string, bool) {
		switch key {
		case "AIC_RELEASE_BASE_URL":
			return server.URL, true
		case "AIC_VERSION":
			return "v" + Version, true
		}
		return oldLookup(key)
	}
	deps.HTTPClient = server.Client()
	if code := Main(context.Background(), []string{"upgrade"}, deps); code != 0 || artifactRequests.Load() != 0 || !strings.Contains(stdout.String(), "当前已是最新版本") {
		t.Fatalf("code=%d requests=%d out=%q err=%q", code, artifactRequests.Load(), stdout.String(), stderr.String())
	}
}

func TestHelpAndVersion(t *testing.T) {
	for _, test := range []struct {
		args     []string
		expected string
	}{{nil, "用法：agentsync"}, {[]string{"--version"}, "ai-instructions " + Version + "\n"}, {[]string{"-V"}, "ai-instructions " + Version + "\n"}} {
		deps, stdout, _, _ := testDeps(t)
		if code := Main(context.Background(), test.args, deps); code != 0 || !strings.Contains(stdout.String(), test.expected) {
			t.Fatalf("args=%v code=%d output=%q", test.args, code, stdout.String())
		}
	}
}

func TestLegacyCommandWarnsAboutRename(t *testing.T) {
	for _, name := range []string{"aic", "ai-instructions"} {
		deps, _, stderr, _ := testDeps(t)
		deps.ProgramName = name
		if code := Main(context.Background(), []string{"version"}, deps); code != 0 {
			t.Fatalf("name=%s code=%d", name, code)
		}
		if want := "[WARN] " + name + " 已更名为 agentsync"; !strings.Contains(stderr.String(), want) {
			t.Fatalf("name=%s stderr=%q missing=%q", name, stderr.String(), want)
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

func TestSourceTestNormalizesGitHubBlobURL(t *testing.T) {
	deps, stdout, stderr, _ := testDeps(t)
	var requested string
	deps.HTTPClient = &http.Client{Transport: appRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		requested = req.URL.String()
		return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Header: http.Header{"Content-Type": []string{"text/plain"}}, Body: io.NopCloser(strings.NewReader("rules\n"))}, nil
	})}
	input := "https://github.com/RokiLai/agents/blob/main/AGENTS.md"
	if code := Main(context.Background(), []string{"source", "test", input}, deps); code != 0 {
		t.Fatalf("code=%d out=%q err=%q", code, stdout.String(), stderr.String())
	}
	want := "https://raw.githubusercontent.com/RokiLai/agents/main/AGENTS.md"
	if requested != want || !strings.Contains(stdout.String(), "已转换为原始文件地址") || !strings.Contains(stdout.String(), want) {
		t.Fatalf("requested=%q out=%q", requested, stdout.String())
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

func TestSourceSetPersistsNormalizedGitHubURL(t *testing.T) {
	deps, stdout, stderr, home := testDeps(t)
	deps.Stdin = strings.NewReader("y\n")
	deps.IsTerminal = func() bool { return true }
	deps.HTTPClient = &http.Client{Transport: appRoundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Header: http.Header{"Content-Type": []string{"text/plain"}}, Body: io.NopCloser(strings.NewReader("rules\n"))}, nil
	})}
	if err := os.MkdirAll(filepath.Join(home, "config"), 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(home, "config/agents-url")
	if err := os.WriteFile(path, []byte(config.AgentsURLMarker+"\nhttps://old.test/AGENTS.md\n"), 0600); err != nil {
		t.Fatal(err)
	}
	input := "https://github.com/RokiLai/agents/blob/main/AGENTS.md"
	if code := Main(context.Background(), []string{"source", "set", input}, deps); code != 0 {
		t.Fatalf("code=%d out=%q err=%q", code, stdout.String(), stderr.String())
	}
	want := "https://raw.githubusercontent.com/RokiLai/agents/main/AGENTS.md"
	got, err := config.ReadManagedValue(path, config.AgentsURLMarker)
	if err != nil || got != want || !strings.Contains(stdout.String(), "已转换为原始文件地址") {
		t.Fatalf("got=%q out=%q err=%v", got, stdout.String(), err)
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

type appRoundTripFunc func(*http.Request) (*http.Response, error)

func (f appRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

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

type safeTestBuffer struct {
	b  bytes.Buffer
	mu sync.Mutex
}

func (s *safeTestBuffer) Write(p []byte) (n int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *safeTestBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

func TestUpgradeInteractiveTerminalSpinner(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		if r.URL.Path == "/latest/download/checksums.txt" {
			http.Redirect(w, r, "/download/v"+Version+"/checksums.txt", http.StatusFound)
			return
		}
		if filepath.Base(r.URL.Path) == "checksums.txt" {
			fmt.Fprintln(w, "checksums")
			return
		}
	}))
	defer server.Close()

	deps, _, stderr, home := testDeps(t)
	installed := filepath.Join(home, "config/bin/ai-instructions")
	if err := os.MkdirAll(filepath.Dir(installed), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(installed, []byte("#!/bin/sh\nprintf 'ai-instructions "+Version+"\\n'\n"), 0700); err != nil {
		t.Fatal(err)
	}

	var stdout safeTestBuffer
	deps.Stdout = &stdout
	deps.IsOutputTerminal = func() bool { return true }
	oldLookup := deps.LookupEnv
	deps.LookupEnv = func(key string) (string, bool) {
		if key == "AIC_RELEASE_BASE_URL" {
			return server.URL, true
		}
		return oldLookup(key)
	}
	deps.HTTPClient = server.Client()

	code := Main(context.Background(), []string{"upgrade"}, deps)
	if code != 0 {
		t.Fatalf("expected code 0, got %d out=%s err=%s", code, stdout.String(), stderr.String())
	}
	got := stdout.String()
	if !strings.Contains(got, "当前已是最新版本") {
		t.Fatalf("missing expected output in %q", got)
	}
	if !strings.Contains(got, "\r\033[2K") {
		t.Fatalf("expected ANSI spinner line clear in interactive output: %q", got)
	}
}
