package integration_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	agentsynctool "github.com/RokiLai/agent_sync_tool"
)

func TestQualityGateKeepsRaceTimeoutVetAndCrossBuild(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	path := filepath.Join(filepath.Dir(file), "..", "..", "Makefile")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, command := range []string{"go vet ./...", "go test -race -timeout 2m ./...", "$(MAKE) cross-build"} {
		if !strings.Contains(text, command) {
			t.Fatalf("quality gate is missing %q", command)
		}
	}
}

func TestReleaseBuildReadsVersionFile(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	path := filepath.Join(filepath.Dir(file), "..", "..", "scripts", "build-release.sh")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, expected := range []string{"VERSION", "AIC_BUILD_VERSION", "command_name=agentsync", "primary_artifact_prefix=agentsync", "legacy_artifact_prefix=aic", "[vV]*) version=${version#?}"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("release build is missing %q", expected)
		}
	}
}

func TestVersionFileMatchesAppVersion(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	path := filepath.Join(filepath.Dir(file), "..", "..", "VERSION")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	expected := strings.TrimSpace(string(data))
	if expected == "" {
		t.Fatal("VERSION file is empty")
	}
	if agentsynctool.Version != expected {
		t.Fatalf("agentsynctool.Version = %q, want %q", agentsynctool.Version, expected)
	}
}
