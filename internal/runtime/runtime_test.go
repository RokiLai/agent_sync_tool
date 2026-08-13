package runtime

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRevisionVectors(t *testing.T) {
	vectors := map[string]string{
		"":        "e69de29bb2d1d6434b8b29ae775ad8c2e48c5391",
		"hello\n": "ce013625030ba8dba906f756967f9e9ca394464a",
		"# Personal test rules\n\n- keep tests enabled\n": "4f2c79608944b96c293d9462186c432dd9aecf16",
	}
	for input, expected := range vectors {
		if got := Revision([]byte(input)); got != expected {
			t.Errorf("Revision(%q)=%s want %s", input, got, expected)
		}
	}
}

func TestInspectLegacyAndVersioned(t *testing.T) {
	legacy := t.TempDir()
	if err := os.WriteFile(filepath.Join(legacy, "AGENTS.md"), []byte("rules\n"), 0444); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacy, "REVISION"), []byte("abc\n"), 0444); err != nil {
		t.Fatal(err)
	}
	if state := Inspect(legacy); !state.Valid || state.Revision != "abc" {
		t.Fatalf("legacy state: %#v", state)
	}
	versioned := t.TempDir()
	versionDir := filepath.Join(versioned, "versions/abc")
	if err := os.MkdirAll(versionDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(versionDir, "AGENTS.md"), []byte("rules\n"), 0444); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(versionDir, "REVISION"), []byte("abc\n"), 0444); err != nil {
		t.Fatal(err)
	}
	for link, target := range map[string]string{"current": "versions/abc", "AGENTS.md": "current/AGENTS.md", "REVISION": "current/REVISION"} {
		if err := os.Symlink(target, filepath.Join(versioned, link)); err != nil {
			t.Fatal(err)
		}
	}
	if state := Inspect(versioned); !state.Valid || state.Revision != "abc" {
		t.Fatalf("versioned state: %#v", state)
	}
}

func TestInspectRejectsIncomplete(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("rules"), 0600); err != nil {
		t.Fatal(err)
	}
	if Inspect(dir).Valid {
		t.Fatal("incomplete runtime accepted")
	}
}

func TestSize(t *testing.T) {
	if got := Size([]byte("你好")); got != "6" {
		t.Fatalf("Size=%s want 6", got)
	}
}
