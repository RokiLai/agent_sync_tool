package core

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RokiLai/agent_sync_tool/internal/config"
)

func TestURLAndRevisionContracts(t *testing.T) {
	if err := ValidateURL("https://example.test/AGENTS.md"); err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{"", "file:///tmp/rules", "https://example.test/a\nb"} {
		if err := ValidateURL(raw); err == nil {
			t.Fatalf("ValidateURL(%q) accepted", raw)
		}
	}
	if got := Revision([]byte("hello\n")); got != "ce013625030ba8dba906f756967f9e9ca394464a" {
		t.Fatalf("Revision=%s", got)
	}
	if Size([]byte("你好")) != "6" {
		t.Fatal("Size must count bytes")
	}
}

func TestNormalizeGitHubBlobURL(t *testing.T) {
	raw := "https://github.com/RokiLai/agents/blob/main/rules/AGENTS.md?plain=1#top"
	got, changed, err := NormalizeURL(raw)
	if err != nil || !changed || got != "https://raw.githubusercontent.com/RokiLai/agents/main/rules/AGENTS.md" {
		t.Fatalf("got=%q changed=%v err=%v", got, changed, err)
	}
	sha := strings.Repeat("a", 40)
	got, changed, err = NormalizeURL("https://github.com/o/r/blob/" + sha + "/AGENTS.md")
	if err != nil || !changed || got != "https://raw.githubusercontent.com/o/r/"+sha+"/AGENTS.md" {
		t.Fatalf("got=%q changed=%v err=%v", got, changed, err)
	}
}

func TestNormalizeURLRejectsGuessing(t *testing.T) {
	for _, raw := range []string{
		"https://github.com/o/r/blob/feature/rules/AGENTS.md",
		"https://github.com:443/o/r/blob/main/AGENTS.md",
		"https://user@github.com/o/r/blob/main/AGENTS.md",
		"https://github.example.com/o/r/blob/main/AGENTS.md",
		"https://github.com/o/r/tree/main/AGENTS.md",
	} {
		got, changed, err := NormalizeURL(raw)
		if err != nil || changed || got != raw {
			t.Fatalf("raw=%q got=%q changed=%v err=%v", raw, got, changed, err)
		}
	}
	if _, _, err := NormalizeURL("file:///tmp/AGENTS.md"); err == nil {
		t.Fatal("invalid URL accepted")
	}
}

func TestReleaseContracts(t *testing.T) {
	if got, err := Artifact("linux", "amd64"); err != nil || got != "agentsync_Linux_x86_64" {
		t.Fatalf("artifact=%q err=%v", got, err)
	}
	if got, err := LegacyArtifact("linux", "amd64"); err != nil || got != "aic_Linux_x86_64" {
		t.Fatalf("legacy artifact=%q err=%v", got, err)
	}
	if _, err := Artifact("windows", "amd64"); err == nil {
		t.Fatal("unsupported platform accepted")
	}
	if got := ReleaseURL("https://example.test/releases/", "v2", "aic"); got != "https://example.test/releases/download/v2/aic" {
		t.Fatal(got)
	}
	data := []byte("binary")
	sum := sha256.Sum256(data)
	checks := ParseChecksums([]byte(fmt.Sprintf("%x  aic\n", sum)))
	if err := VerifyChecksum(data, checks["aic"]); err != nil {
		t.Fatal(err)
	}
	if err := VerifyChecksum([]byte("changed"), checks["aic"]); err == nil {
		t.Fatal("checksum mismatch accepted")
	}
}

func TestRuntimeCandidatePublishAndInspect(t *testing.T) {
	dir := t.TempDir()
	if _, err := NewCandidate(nil); err == nil {
		t.Fatal("empty candidate accepted")
	}
	candidate, err := NewCandidate([]byte("rules\n"))
	if err != nil {
		t.Fatal(err)
	}
	if err := (Publisher{Dir: dir}).Publish(candidate); err != nil {
		t.Fatal(err)
	}
	state := InspectRuntime(dir)
	if !state.Valid || state.Revision != candidate.Revision {
		t.Fatalf("state=%#v", state)
	}
	if got, err := os.Readlink(filepath.Join(dir, "current")); err != nil || got != filepath.Join("versions", candidate.Revision) {
		t.Fatalf("current=%q err=%v", got, err)
	}
}

func TestRCInstallValidateAndRemove(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".zshrc")
	if err := os.WriteFile(path, []byte("user\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := InstallRC(path, "/tmp/shell.sh"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateRC(path); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(data), config.BlockBegin) {
		t.Fatalf("data=%q err=%v", data, err)
	}
	if err := RemoveRC(path); err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile(path)
	if err != nil || !strings.Contains(string(data), "user\n") || strings.Contains(string(data), config.BlockBegin) || strings.Contains(string(data), config.BlockEnd) {
		t.Fatalf("data=%q err=%v", data, err)
	}
	if RCPath("/home/test", "zsh") != "/home/test/.zshrc" || RCPath("/home/test", "none") != "" {
		t.Fatal("unexpected RC path")
	}
}
