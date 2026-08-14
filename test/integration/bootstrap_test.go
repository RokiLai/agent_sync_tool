package integration_test

import (
	"crypto/sha256"
	"fmt"
	"github.com/RokiLai/agent_sync_tool/internal/release"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestBootstrapInstallAndChecksumFailure(t *testing.T) {
	root := projectRoot(t)
	binary := buildAIC(t)
	home := t.TempDir()
	artifact, err := release.Artifact(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := os.ReadFile(binary)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(candidate)
	assets := filepath.Join(home, "assets")
	fake := filepath.Join(home, "fake-bin")
	if err := os.MkdirAll(assets, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(fake, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(assets, artifact), candidate, 0700); err != nil {
		t.Fatal(err)
	}
	checks := filepath.Join(assets, "checksums.txt")
	if err := os.WriteFile(checks, []byte(fmt.Sprintf("%x  %s\n", sum, artifact)), 0600); err != nil {
		t.Fatal(err)
	}
	curlScript := "#!/bin/sh\nout=\nurl=\nwhile [ \"$#\" -gt 0 ]; do case \"$1\" in -o) out=$2; shift 2 ;; --proto|--proto-redir) shift 2 ;; --fail|--silent|--show-error|--location) shift ;; *) url=$1; shift ;; esac; done\ncp \"$AIC_ASSETS/${url##*/}\" \"$out\"\n"
	if err := os.WriteFile(filepath.Join(fake, "curl"), []byte(curlScript), 0700); err != nil {
		t.Fatal(err)
	}
	unameScript := "#!/bin/sh\ncase \"$1\" in -s) printf '" + map[string]string{"darwin": "Darwin", "linux": "Linux"}[runtime.GOOS] + "\\n' ;; -m) printf '" + map[string]string{"amd64": "x86_64", "arm64": "arm64"}[runtime.GOARCH] + "\\n' ;; esac\n"
	if err := os.WriteFile(filepath.Join(fake, "uname"), []byte(unameScript), 0700); err != nil {
		t.Fatal(err)
	}
	rules := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("rules\n")) }))
	defer rules.Close()
	env := append(os.Environ(), "HOME="+home, "PATH="+fake+":/usr/bin:/bin:/usr/sbin:/sbin", "AIC_ASSETS="+assets, "AIC_RELEASE_BASE_URL=https://release.test/releases", "AI_INSTRUCTIONS_CONFIG_DIR="+filepath.Join(home, "config"), "AI_INSTRUCTIONS_RUNTIME_DIR="+filepath.Join(home, "runtime"), "AI_INSTRUCTIONS_BIN_DIR="+filepath.Join(home, "bin"), "CODEX_HOME="+filepath.Join(home, "codex"))
	cmd := exec.Command("sh", filepath.Join(root, "install.sh"), rules.URL, "--shell", "none", "--tools", "codex")
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("out=%s err=%v", out, err)
	}
	installed := filepath.Join(home, "config/bin/ai-instructions")
	before, err := os.ReadFile(installed)
	if err != nil {
		t.Fatal(err)
	}
	status := exec.Command(installed, "status")
	status.Env = env
	statusOut, err := status.CombinedOutput()
	if err != nil || !strings.Contains(string(statusOut), "[OK] 安装模式：Release二进制") || strings.Contains(string(statusOut), "[FAIL] 仓库不存在") {
		t.Fatalf("status=%s err=%v", statusOut, err)
	}
	doctor := exec.Command(installed, "doctor")
	doctor.Env = env
	doctorOut, err := doctor.CombinedOutput()
	if err != nil || !strings.Contains(string(doctorOut), "[OK] Release二进制安装") || !strings.Contains(string(doctorOut), "0 个失败") {
		t.Fatalf("doctor=%s err=%v", doctorOut, err)
	}
	if err := os.WriteFile(checks, []byte("deadbeef  "+artifact+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cmd = exec.Command("sh", filepath.Join(root, "install.sh"), rules.URL, "--shell", "none", "--tools", "codex")
	cmd.Env = env
	out, err = cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(out), "SHA-256") {
		t.Fatalf("out=%s err=%v", out, err)
	}
	after, _ := os.ReadFile(installed)
	if string(before) != string(after) {
		t.Fatal("installed changed after checksum failure")
	}
}
