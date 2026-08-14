package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func projectRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../.."))
}

func buildAIC(t *testing.T) string {
	return buildAICVersion(t, "")
}

func buildAICVersion(t *testing.T, version string) string {
	t.Helper()
	root := projectRoot(t)
	target := filepath.Join(t.TempDir(), "aic")
	args := []string{"build", "-o", target}
	if version != "" {
		args = append(args, "-ldflags", "-X github.com/RokiLai/agent_sync_tool/internal/app.Version="+version)
	}
	args = append(args, "./cmd/aic")
	cmd := exec.Command("go", args...)
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	return target
}

func TestBinaryHelpMatchesGoldenAndShell(t *testing.T) {
	root := projectRoot(t)
	binary := buildAIC(t)
	home := t.TempDir()
	cmd := exec.Command(binary, "help")
	cmd.Env = append(os.Environ(), "HOME="+home)
	got, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(filepath.Join(root, "test/contract/golden/help.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatalf("Go help differs from golden")
	}
}

func TestBinaryVersionAliases(t *testing.T) {
	binary := buildAIC(t)
	home := t.TempDir()
	for _, arg := range []string{"version", "--version", "-V"} {
		cmd := exec.Command(binary, arg)
		cmd.Env = append(os.Environ(), "HOME="+home)
		out, err := cmd.Output()
		if err != nil || string(out) != "ai-instructions 3.1.1\n" {
			t.Fatalf("arg=%s out=%q err=%v", arg, out, err)
		}
	}
}
