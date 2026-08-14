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

func sourceProject(t *testing.T) string {
	t.Helper()
	root := projectRoot(t)
	path := os.Getenv("AI_INSTRUCTIONS_SOURCE_PROJECT")
	if path == "" {
		path = filepath.Join(root, "../ai-instructions")
	}
	if _, err := os.Stat(filepath.Join(path, "bin/ai-instructions")); err != nil {
		t.Skip("Shell source project unavailable")
	}
	return path
}

func compatibilityEnv(home, repo string) []string {
	return append(os.Environ(), "HOME="+home, "SHELL=/bin/zsh", "AI_INSTRUCTIONS_REPO="+repo, "AI_INSTRUCTIONS_RUNTIME_DIR="+filepath.Join(home, "runtime"), "AI_INSTRUCTIONS_CONFIG_DIR="+filepath.Join(home, "config"), "AI_INSTRUCTIONS_BIN_DIR="+filepath.Join(home, "bin"), "CODEX_HOME="+filepath.Join(home, "codex"))
}

func fakeCurl(t *testing.T, home, bodyPath string) string {
	t.Helper()
	dir := filepath.Join(home, "fake-bin")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "curl")
	script := "#!/bin/sh\noutput=\nwhile [ \"$#\" -gt 0 ]; do case \"$1\" in --output) output=$2; shift 2 ;; --proto|--proto-redir|--connect-timeout|--max-time) shift 2 ;; --fail|--silent|--show-error|--location) shift ;; *) shift ;; esac; done\ncp \"$AIC_COMPAT_BODY\" \"$output\"\n"
	if err := os.WriteFile(path, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestShellRuntimeReadByGo(t *testing.T) {
	source := sourceProject(t)
	binary := buildAIC(t)
	home := t.TempDir()
	repo := filepath.Join(home, "repo")
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	body := filepath.Join(home, "rules")
	if err := os.WriteFile(body, []byte("shared rules\n"), 0600); err != nil {
		t.Fatal(err)
	}
	fakeBin := fakeCurl(t, home, body)
	env := compatibilityEnv(home, repo)
	env = append(env, "PATH="+fakeBin+":/usr/bin:/bin:/usr/sbin:/sbin", "AIC_COMPAT_BODY="+body)
	shell := exec.Command("sh", filepath.Join(source, "bin/ai-instructions"), "install", "https://rules.example/AGENTS.md", "--shell", "none", "--tools", "codex")
	shell.Env = env
	if out, err := shell.CombinedOutput(); err != nil {
		t.Fatalf("shell install: %v\n%s", err, out)
	}
	cmd := exec.Command(binary, "status")
	cmd.Env = env
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "[OK] runtime") || !strings.Contains(string(out), "https://rules.example/AGENTS.md") {
		t.Fatalf("status=%s", out)
	}
}

func TestGoRuntimeReadByShell(t *testing.T) {
	source := sourceProject(t)
	binary := buildAIC(t)
	home := t.TempDir()
	repo := filepath.Join(home, "repo")
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("shared rules\n")) }))
	defer server.Close()
	env := compatibilityEnv(home, repo)
	goInstall := exec.Command(binary, "install", server.URL, "--shell", "none", "--tools", "codex")
	goInstall.Env = env
	if out, err := goInstall.CombinedOutput(); err != nil {
		t.Fatalf("go install: %v\n%s", err, out)
	}
	body := filepath.Join(home, "rules")
	if err := os.WriteFile(body, []byte("shared rules\n"), 0600); err != nil {
		t.Fatal(err)
	}
	fakeBin := fakeCurl(t, home, body)
	env = append(env, "PATH="+fakeBin+":/usr/bin:/bin:/usr/sbin:/sbin", "AIC_COMPAT_BODY="+body)
	shell := exec.Command("sh", filepath.Join(source, "bin/ai-instructions"), "status")
	shell.Env = env
	out, err := shell.Output()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "[OK] runtime") || !strings.Contains(string(out), server.URL) {
		t.Fatalf("status=%s", out)
	}
}

func TestReadOnlyCommandDifferential(t *testing.T) {
	source := sourceProject(t)
	binary := buildAIC(t)
	home := t.TempDir()
	env := compatibilityEnv(home, filepath.Join(home, "repo"))
	for _, args := range [][]string{{"help"}, {"version"}, {"--version"}, {"-V"}} {
		goCmd := exec.Command(binary, args...)
		goCmd.Env = env
		goOut, goErr := goCmd.CombinedOutput()
		shellCmd := exec.Command("sh", append([]string{filepath.Join(source, "bin/ai-instructions")}, args...)...)
		shellCmd.Env = env
		shellOut, shellErr := shellCmd.CombinedOutput()
		if (goErr == nil) != (shellErr == nil) {
			t.Fatalf("args=%v go=%q/%v shell=%q/%v", args, goOut, goErr, shellOut, shellErr)
		}
		if args[0] == "help" {
			if string(goOut) != string(shellOut) {
				t.Fatalf("args=%v go=%q shell=%q", args, goOut, shellOut)
			}
			continue
		}
		if string(goOut) != "ai-instructions 3.1.0\n" || string(shellOut) != "ai-instructions 2.0.0\n" {
			t.Fatalf("args=%v go=%q shell=%q", args, goOut, shellOut)
		}
	}
}

func TestShellInstallThenGoCandidateReplacement(t *testing.T) {
	source := sourceProject(t)
	binary := buildAIC(t)
	home := t.TempDir()
	repo := filepath.Join(home, "repo")
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	body := filepath.Join(home, "rules")
	if err := os.WriteFile(body, []byte("shared rules\n"), 0600); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("shared rules\n")) }))
	defer server.Close()
	fakeBin := fakeCurl(t, home, body)
	env := compatibilityEnv(home, repo)
	env = append(env, "PATH="+fakeBin+":/usr/bin:/bin:/usr/sbin:/sbin", "AIC_COMPAT_BODY="+body)
	shell := exec.Command("sh", filepath.Join(source, "bin/ai-instructions"), "install", server.URL, "--shell", "none", "--tools", "codex")
	shell.Env = env
	if out, err := shell.CombinedOutput(); err != nil {
		t.Fatalf("shell install: %v\n%s", err, out)
	}
	goInstall := exec.Command(binary, "install", server.URL, "--shell", "none", "--tools", "codex")
	goInstall.Env = env
	if out, err := goInstall.CombinedOutput(); err != nil {
		t.Fatalf("go replacement: %v\n%s", err, out)
	}
	installed := filepath.Join(home, "config/bin/ai-instructions")
	cmd := exec.Command(installed, "version")
	cmd.Env = env
	out, err := cmd.Output()
	if err != nil || string(out) != "ai-instructions 3.1.0\n" {
		t.Fatalf("installed=%q err=%v", out, err)
	}
	status := exec.Command(installed, "status")
	status.Env = env
	statusOut, err := status.Output()
	if err != nil || !strings.Contains(string(statusOut), "[OK] runtime") {
		t.Fatalf("status=%q err=%v", statusOut, err)
	}
}

func TestImplementationStateTreesMatch(t *testing.T) {
	source := sourceProject(t)
	binary := buildAIC(t)
	root := t.TempDir()
	shellHome := filepath.Join(root, "shell")
	goHome := filepath.Join(root, "go")
	for _, home := range []string{shellHome, goHome} {
		if err := os.MkdirAll(filepath.Join(home, "repo/.git"), 0755); err != nil {
			t.Fatal(err)
		}
	}
	body := filepath.Join(root, "rules")
	if err := os.WriteFile(body, []byte("shared rules\n"), 0600); err != nil {
		t.Fatal(err)
	}
	fakeBin := fakeCurl(t, root, body)
	shellEnv := compatibilityEnv(shellHome, filepath.Join(shellHome, "repo"))
	shellEnv = append(shellEnv, "PATH="+fakeBin+":/usr/bin:/bin:/usr/sbin:/sbin", "AIC_COMPAT_BODY="+body)
	shell := exec.Command("sh", filepath.Join(source, "bin/ai-instructions"), "install", "https://rules.example/AGENTS.md", "--shell", "none", "--tools", "codex")
	shell.Env = shellEnv
	if out, err := shell.CombinedOutput(); err != nil {
		t.Fatalf("shell: %v\n%s", err, out)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("shared rules\n")) }))
	defer server.Close()
	goEnv := compatibilityEnv(goHome, filepath.Join(goHome, "repo"))
	goCmd := exec.Command(binary, "install", server.URL, "--shell", "none", "--tools", "codex")
	goCmd.Env = goEnv
	if out, err := goCmd.CombinedOutput(); err != nil {
		t.Fatalf("go: %v\n%s", err, out)
	}
	for _, rel := range []string{"runtime/AGENTS.md", "runtime/REVISION", "runtime/current", "codex/AGENTS.md"} {
		sType, sValue := treeValue(t, shellHome, rel)
		gType, gValue := treeValue(t, goHome, rel)
		sValue = strings.ReplaceAll(sValue, shellHome, "$HOME")
		gValue = strings.ReplaceAll(gValue, goHome, "$HOME")
		if sType != gType || sValue != gValue {
			t.Fatalf("%s shell=%s:%s go=%s:%s", rel, sType, sValue, gType, gValue)
		}
	}
}

func treeValue(t *testing.T, root, rel string) (string, string) {
	t.Helper()
	path := filepath.Join(root, rel)
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(path)
		if err != nil {
			t.Fatal(err)
		}
		return "link", target
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return "file", string(data)
}
