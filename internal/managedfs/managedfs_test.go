package managedfs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAtomicWriteAndSymlinks(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config/value")
	if err := AtomicWrite(path, []byte("value\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(path); string(data) != "value\n" {
		t.Fatalf("data=%q", data)
	}
	link := filepath.Join(dir, "link")
	if err := EnsureSymlink(link, "one"); err != nil {
		t.Fatal(err)
	}
	if err := EnsureSymlink(link, "one"); err != nil {
		t.Fatal(err)
	}
	if err := EnsureSymlink(link, "two"); err == nil {
		t.Fatal("expected conflict")
	}
	if err := AtomicSymlink(link, "two"); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.Readlink(link); got != "two" {
		t.Fatalf("target=%q", got)
	}
}

func TestEnsureSymlinkRejectsFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(path, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := EnsureSymlink(path, "target"); err == nil {
		t.Fatal("expected conflict")
	}
}
