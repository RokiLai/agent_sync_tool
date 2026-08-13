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
	t.Helper()
	root := projectRoot(t)
	target := filepath.Join(t.TempDir(), "aic")
	cmd := exec.Command("go", "build", "-o", target, "./cmd/aic")
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
	sourceProject := os.Getenv("AI_INSTRUCTIONS_SOURCE_PROJECT")
	if sourceProject == "" {
		sourceProject = filepath.Join(root, "../ai-instructions")
	}
	if _, err := os.Stat(filepath.Join(sourceProject, "bin/ai-instructions")); err != nil {
		t.Skip("Shell source project unavailable")
	}
	shell := exec.Command("sh", filepath.Join(sourceProject, "bin/ai-instructions"), "help")
	shell.Env = append(os.Environ(), "HOME="+home)
	shellOut, err := shell.Output()
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(shellOut) {
		t.Fatal("Go and Shell help differ")
	}
}

func TestBinaryVersionAliases(t *testing.T) {
	binary := buildAIC(t)
	home := t.TempDir()
	for _, arg := range []string{"version", "--version", "-V"} {
		cmd := exec.Command(binary, arg)
		cmd.Env = append(os.Environ(), "HOME="+home)
		out, err := cmd.Output()
		if err != nil || string(out) != "ai-instructions 3.0.0\n" {
			t.Fatalf("arg=%s out=%q err=%v", arg, out, err)
		}
	}
}
