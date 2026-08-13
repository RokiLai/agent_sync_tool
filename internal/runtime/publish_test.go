package runtime

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPublishAndIdempotence(t *testing.T) {
	dir := t.TempDir()
	candidate, _ := NewCandidate([]byte("rules\n"))
	p := Publisher{Dir: dir}
	if err := p.Publish(candidate); err != nil {
		t.Fatal(err)
	}
	if err := p.Publish(candidate); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.Readlink(filepath.Join(dir, "current")); got != filepath.Join("versions", candidate.Revision) {
		t.Fatalf("current=%q", got)
	}
	for _, name := range []string{"AGENTS.md", "REVISION"} {
		info, err := os.Stat(filepath.Join(dir, "versions", candidate.Revision, name))
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0444 {
			t.Fatalf("%s mode=%o", name, info.Mode().Perm())
		}
	}
}

func TestPublishRejectsConflict(t *testing.T) {
	dir := t.TempDir()
	candidate, _ := NewCandidate([]byte("rules\n"))
	version := filepath.Join(dir, "versions", candidate.Revision)
	if err := os.MkdirAll(version, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(version, "AGENTS.md"), []byte("foreign"), 0444); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(version, "REVISION"), []byte(candidate.Revision+"\n"), 0444); err != nil {
		t.Fatal(err)
	}
	if err := (Publisher{Dir: dir}).Publish(candidate); err == nil {
		t.Fatal("expected conflict")
	}
}

func TestPublishMigratesLegacy(t *testing.T) {
	dir := t.TempDir()
	data := []byte("rules\n")
	rev := Revision(data)
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), data, 0444); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "REVISION"), []byte(rev+"\n"), 0444); err != nil {
		t.Fatal(err)
	}
	candidate, _ := NewCandidate(data)
	if err := (Publisher{Dir: dir}).Publish(candidate); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Readlink(filepath.Join(dir, "AGENTS.md")); err != nil {
		t.Fatal(err)
	}
}

func TestCandidateAndLinkConflicts(t *testing.T) {
	if _, err := NewCandidate(nil); err == nil {
		t.Fatal("empty candidate accepted")
	}
	dir := t.TempDir()
	candidate, _ := NewCandidate([]byte("rules\n"))
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("foreign"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := (Publisher{Dir: dir}).Publish(candidate); err == nil {
		t.Fatal("expected link conflict")
	}
}

func TestPublishRejectsMalformedLegacyRevision(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("rules"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "REVISION"), []byte("bad revision\n"), 0600); err != nil {
		t.Fatal(err)
	}
	candidate, _ := NewCandidate([]byte("new"))
	if err := (Publisher{Dir: dir}).Publish(candidate); err == nil {
		t.Fatal("expected legacy error")
	}
}
