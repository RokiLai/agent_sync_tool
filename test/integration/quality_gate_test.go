package integration_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestImplementationContractsKeepRaceAndTimeout(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	path := filepath.Join(filepath.Dir(file), "..", "..", "scripts", "run-implementation-contracts.sh")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	command := "go test -race -timeout 2m ./..."
	if count := strings.Count(string(data), command); count != 2 {
		t.Fatalf("quality gate command count=%d want 2", count)
	}
}

func TestReleaseBuildNormalizesTagVersion(t *testing.T) {
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
	for _, expected := range []string{"AIC_BUILD_VERSION:-3.0.0", "[vV]*) version=${version#?}"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("release build is missing %q", expected)
		}
	}
}
