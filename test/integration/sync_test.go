package integration_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

func TestBinarySyncAndLastKnownGood(t *testing.T) {
	binary := buildAIC(t)
	home := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("rules\n")) }))
	configDir := filepath.Join(home, "config")
	runtimeDir := filepath.Join(home, "runtime")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "agents-url"), []byte("# ai-instructions AGENTS URL v1\n"+server.URL+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	run := func() (string, string, error) {
		cmd := exec.Command(binary, "sync")
		cmd.Env = append(os.Environ(), "HOME="+home, "AI_INSTRUCTIONS_CONFIG_DIR="+configDir, "AI_INSTRUCTIONS_RUNTIME_DIR="+runtimeDir)
		var stdout, stderr strings.Builder
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err := cmd.Run()
		return stdout.String(), stderr.String(), err
	}
	stdout, stderr, err := run()
	if err != nil || stdout != "" || !strings.Contains(stderr, "AI 指令已同步") {
		t.Fatalf("out=%q errout=%q err=%v", stdout, stderr, err)
	}
	server.Close()
	stdout, stderr, err = run()
	if err != nil || stdout != "" || !strings.Contains(stderr, "最后一次成功部署") {
		t.Fatalf("cache out=%q errout=%q err=%v", stdout, stderr, err)
	}
	if _, err := os.Readlink(filepath.Join(runtimeDir, "current")); err != nil {
		t.Fatal(err)
	}
}

func TestBinaryConcurrentSync(t *testing.T) {
	binary := buildAIC(t)
	home := t.TempDir()
	var requests sync.WaitGroup
	requests.Add(2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		_, _ = w.Write([]byte("rules\n"))
		requests.Done()
	}))
	defer server.Close()
	configDir := filepath.Join(home, "config")
	runtimeDir := filepath.Join(home, "runtime with 空格")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "agents-url"), []byte("# ai-instructions AGENTS URL v1\n"+server.URL+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	env := append(os.Environ(), "HOME="+home, "AI_INSTRUCTIONS_CONFIG_DIR="+configDir, "AI_INSTRUCTIONS_RUNTIME_DIR="+runtimeDir)
	commands := []*exec.Cmd{exec.Command(binary, "sync"), exec.Command(binary, "sync")}
	outputs := make([]strings.Builder, len(commands))
	for i, cmd := range commands {
		cmd.Env = env
		cmd.Stderr = &outputs[i]
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
	}
	for i, cmd := range commands {
		if err := cmd.Wait(); err != nil {
			t.Fatalf("command %d: %v stderr=%s", i, err, outputs[i].String())
		}
	}
	requests.Wait()
	if _, err := os.Readlink(filepath.Join(runtimeDir, "current")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(runtimeDir + ".lock"); !os.IsNotExist(err) {
		t.Fatalf("lock remains: %v", err)
	}
}

func TestBinarySIGTERMCleansLock(t *testing.T) {
	binary := buildAIC(t)
	home := t.TempDir()
	started := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { close(started); <-r.Context().Done() }))
	defer server.Close()
	configDir := filepath.Join(home, "config")
	runtimeDir := filepath.Join(home, "runtime")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "agents-url"), []byte("# ai-instructions AGENTS URL v1\n"+server.URL+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, "sync")
	cmd.Env = append(os.Environ(), "HOME="+home, "AI_INSTRUCTIONS_CONFIG_DIR="+configDir, "AI_INSTRUCTIONS_RUNTIME_DIR="+runtimeDir)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-started:
	case <-ctx.Done():
		t.Fatal("request did not start")
	}
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Wait(); err == nil {
		t.Fatal("SIGTERM should return non-zero")
	}
	deadline := time.Now().Add(time.Second)
	for {
		_, err := os.Stat(runtimeDir + ".lock")
		if os.IsNotExist(err) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("lock remains: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestBinarySyncAutoTTL(t *testing.T) {
	binary := buildAIC(t)
	home := t.TempDir()
	var requestCount atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		_, _ = w.Write([]byte("rules\n"))
	}))
	defer server.Close()

	configDir := filepath.Join(home, "config")
	runtimeDir := filepath.Join(home, "runtime")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "agents-url"), []byte("# ai-instructions AGENTS URL v1\n"+server.URL+"\n"), 0600); err != nil {
		t.Fatal(err)
	}

	run := func(args ...string) (string, string, error) {
		cmdArgs := append([]string{"sync"}, args...)
		cmd := exec.Command(binary, cmdArgs...)
		cmd.Env = append(os.Environ(), "HOME="+home, "AI_INSTRUCTIONS_CONFIG_DIR="+configDir, "AI_INSTRUCTIONS_RUNTIME_DIR="+runtimeDir)
		var stdout, stderr strings.Builder
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err := cmd.Run()
		return stdout.String(), stderr.String(), err
	}

	// 1. 首次 auto sync：本地无缓存，必须触发请求
	stdout, stderr, err := run("--auto")
	if err != nil || stdout != "" || stderr != "" || requestCount.Load() != 1 {
		t.Fatalf("first auto sync: out=%q errout=%q err=%v reqs=%d", stdout, stderr, err, requestCount.Load())
	}

	// 2. 第二次 auto sync：在 TTL 内，跳过网络请求，静默退出
	stdout, stderr, err = run("--auto")
	if err != nil || stdout != "" || stderr != "" || requestCount.Load() != 1 {
		t.Fatalf("second auto sync within TTL: out=%q errout=%q err=%v reqs=%d", stdout, stderr, err, requestCount.Load())
	}

	// 3. 手动 sync：无视 TTL 强制联网请求，并输出同步提示
	stdout, stderr, err = run()
	if err != nil || stdout != "" || !strings.Contains(stderr, "AI 指令已同步") || requestCount.Load() != 2 {
		t.Fatalf("manual sync: out=%q errout=%q err=%v reqs=%d", stdout, stderr, err, requestCount.Load())
	}

	// 4. 不支持的参数
	_, stderr, err = run("--invalid-flag")
	if err == nil || !strings.Contains(stderr, "不支持的参数") {
		t.Fatalf("invalid flag should fail, errout=%q err=%v", stderr, err)
	}
}
