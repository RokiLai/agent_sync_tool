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
	"sync/atomic"
	"testing"
)

func TestBinaryGoToGoUpgradeAndRollback(t *testing.T) {
	binary := buildAICVersion(t, "3.3.0")
	candidateBinary := buildAIC(t)
	home := t.TempDir()
	configDir := filepath.Join(home, "config/bin")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	installed := filepath.Join(configDir, "ai-instructions")
	old := []byte("#!/bin/sh\nprintf 'ai-instructions 1.0.0\\n'\n")
	if err := os.WriteFile(installed, old, 0700); err != nil {
		t.Fatal(err)
	}
	candidate, err := os.ReadFile(candidateBinary)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(candidate)
	artifact, err := release.Artifact(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Fatal(err)
	}
	var bad atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/latest/download/checksums.txt" {
			http.Redirect(w, r, "/download/v3.3.1/checksums.txt", http.StatusFound)
			return
		}
		if filepath.Base(r.URL.Path) == "checksums.txt" {
			if bad.Load() {
				fmt.Fprintf(w, "deadbeef  %s\n", artifact)
			} else {
				fmt.Fprintf(w, "%x  %s\n", sum, artifact)
			}
			return
		}
		_, _ = w.Write(candidate)
	}))
	defer server.Close()
	env := append(os.Environ(), "HOME="+home, "AI_INSTRUCTIONS_CONFIG_DIR="+filepath.Join(home, "config"), "AIC_RELEASE_BASE_URL="+server.URL)
	cmd := exec.Command(binary, "upgrade")
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil || !strings.Contains(string(out), "升级成功：v3.3.0 → v3.3.1") || strings.Contains(string(out), "\x1b[") {
		t.Fatalf("out=%s err=%v", out, err)
	}
	before, _ := os.ReadFile(installed)
	bad.Store(true)
	cmd = exec.Command(binary, "upgrade")
	cmd.Env = env
	if err := cmd.Run(); err == nil {
		t.Fatal("expected checksum failure")
	}
	after, _ := os.ReadFile(installed)
	if string(before) != string(after) {
		t.Fatal("installed changed")
	}
}
