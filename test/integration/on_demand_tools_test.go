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

func TestOnDemandToolsInstallationAndLifecycle(t *testing.T) {
	binary := buildAIC(t)
	home := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("rule content\n"))
	}))
	defer server.Close()

	binDir := filepath.Join(home, ".local/bin")
	configDir := filepath.Join(home, ".config/ai-instructions")
	runtimeDir := filepath.Join(home, ".local/share/ai-instructions-runtime")
	repoDir := filepath.Join(home, "repo")
	_ = os.MkdirAll(filepath.Join(repoDir, ".git"), 0755)
	_ = os.WriteFile(filepath.Join(home, ".zshrc"), []byte("# test rc\n"), 0644)

	baseEnv := []string{
		"HOME=" + home,
		"SHELL=/bin/zsh",
		"PATH=" + binDir + ":" + os.Getenv("PATH"),
		"AI_INSTRUCTIONS_REPO=" + repoDir,
		"AI_INSTRUCTIONS_CONFIG_DIR=" + configDir,
		"AI_INSTRUCTIONS_RUNTIME_DIR=" + runtimeDir,
		"AI_INSTRUCTIONS_BIN_DIR=" + binDir,
	}

	run := func(args ...string) (string, string, error) {
		cmd := exec.Command(binary, args...)
		cmd.Env = baseEnv
		var stdout, stderr strings.Builder
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err := cmd.Run()
		return stdout.String(), stderr.String(), err
	}

	// 1. 测试显式指定 --tools claude
	stdout, stderr, err := run("install", server.URL, "--tools", "claude", "--shell", "zsh")
	if err != nil {
		t.Fatalf("install failed: err=%v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "[OK] 安装完成") {
		t.Fatalf("expected install ok, got: %s", stdout)
	}

	// 验证只创建了 claude 的符号链接，未创建 codex 或 agy 符号链接
	claudePath := filepath.Join(home, ".claude/CLAUDE.md")
	if _, err := os.Readlink(claudePath); err != nil {
		t.Fatalf("expected claude symlink at %s, err=%v", claudePath, err)
	}
	codexPath := filepath.Join(home, ".codex/AGENTS.md")
	if _, err := os.Lstat(codexPath); !os.IsNotExist(err) {
		t.Fatalf("codex path should not exist: %s", codexPath)
	}
	agyPath := filepath.Join(home, ".gemini/GEMINI.md")
	if _, err := os.Lstat(agyPath); !os.IsNotExist(err) {
		t.Fatalf("agy path should not exist: %s", agyPath)
	}

	// 验证 enabled-tools 写入
	enabledData, err := os.ReadFile(filepath.Join(configDir, "enabled-tools"))
	if err != nil || !strings.Contains(string(enabledData), "claude") {
		t.Fatalf("enabled-tools invalid: %s, err=%v", string(enabledData), err)
	}

	// 2. 验证 shell-init 仅为 claude 生成 wrapper
	stdout, stderr, err = run("shell-init")
	if err != nil {
		t.Fatalf("shell-init failed: %v", err)
	}
	if !strings.Contains(stdout, "claude() {") || !strings.Contains(stdout, "alias cld=claude") {
		t.Fatalf("expected claude wrapper in shell-init, got:\n%s", stdout)
	}
	if strings.Contains(stdout, "codex() {") || strings.Contains(stdout, "agy() {") {
		t.Fatalf("shell-init should not contain uninstalled tools, got:\n%s", stdout)
	}

	// 3. 验证 status 输出
	stdout, stderr, err = run("status")
	if err != nil {
		t.Fatalf("status failed: %v", err)
	}
	if !strings.Contains(stdout, "[OK] Claude 入口：") {
		t.Fatalf("expected OK for Claude in status, got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "[INFO] Codex 入口未安装") || !strings.Contains(stdout, "[INFO] Antigravity 入口未安装") {
		t.Fatalf("expected INFO for uninstalled tools in status, got:\n%s", stdout)
	}

	// 4. 验证 doctor 诊断
	stdout, stderr, err = run("doctor")
	// doctor 应通过，对于未启用的 codex / agy 不应该报 Warning 导致退出码异常
	if !strings.Contains(stdout, "[OK] Claude 入口正确") {
		t.Fatalf("expected doctor ok for claude, got:\n%s", stdout)
	}
	if strings.Contains(stdout, "[WARN] Codex 入口未受管") || strings.Contains(stdout, "[WARN] Antigravity 入口未受管") {
		t.Fatalf("doctor should not warn for unenabled tools, got:\n%s", stdout)
	}
}

func TestAutoDetectToolsInstallation(t *testing.T) {
	binary := buildAIC(t)
	home := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("rule content\n"))
	}))
	defer server.Close()

	binDir := filepath.Join(home, ".local/bin")
	configDir := filepath.Join(home, ".config/ai-instructions")
	runtimeDir := filepath.Join(home, ".local/share/ai-instructions-runtime")
	repoDir := filepath.Join(home, "repo")
	_ = os.MkdirAll(filepath.Join(repoDir, ".git"), 0755)
	_ = os.WriteFile(filepath.Join(home, ".zshrc"), []byte("# test rc\n"), 0644)

	// 事先只创建 .gemini 配置目录模拟本机仅配置了 Antigravity (agy)
	_ = os.MkdirAll(filepath.Join(home, ".gemini"), 0755)

	baseEnv := []string{
		"HOME=" + home,
		"SHELL=/bin/zsh",
		"PATH=" + binDir + ":/usr/bin:/bin",
		"AI_INSTRUCTIONS_REPO=" + repoDir,
		"AI_INSTRUCTIONS_CONFIG_DIR=" + configDir,
		"AI_INSTRUCTIONS_RUNTIME_DIR=" + runtimeDir,
		"AI_INSTRUCTIONS_BIN_DIR=" + binDir,
	}

	run := func(args ...string) (string, string, error) {
		cmd := exec.Command(binary, args...)
		cmd.Env = baseEnv
		var stdout, stderr strings.Builder
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err := cmd.Run()
		return stdout.String(), stderr.String(), err
	}

	// 执行自动探测安装（不传 --tools）
	_, stderr, err := run("install", server.URL, "--shell", "zsh")
	if err != nil {
		t.Fatalf("install auto failed: err=%v stderr=%s", err, stderr)
	}

	// 验证自动探测到 agy 并仅为 agy 创建软链接
	agyPath := filepath.Join(home, ".gemini/GEMINI.md")
	if _, err := os.Readlink(agyPath); err != nil {
		t.Fatalf("expected agy symlink at %s, err=%v", agyPath, err)
	}
	claudePath := filepath.Join(home, ".claude/CLAUDE.md")
	if _, err := os.Lstat(claudePath); !os.IsNotExist(err) {
		t.Fatalf("claude path should not exist: %s", claudePath)
	}

	// 验证 enabled-tools 内容为 agy
	enabledData, _ := os.ReadFile(filepath.Join(configDir, "enabled-tools"))
	if strings.TrimSpace(string(enabledData)) != "# agentsync enabled tools v1\nagy" {
		t.Fatalf("unexpected enabled-tools: %s", string(enabledData))
	}
}
