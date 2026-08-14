package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPathPrecedence(t *testing.T) {
	home := t.TempDir()
	env := map[string]string{"HOME": home, "AI_INSTRUCTIONS_RUNTIME_DIR": filepath.Join(home, "custom runtime"), "AI_INSTRUCTIONS_REPO": filepath.Join(home, "repo")}
	c, err := Load(func(key string) (string, bool) { value, ok := env[key]; return value, ok }, "/missing/aic", "status")
	if err != nil {
		t.Fatal(err)
	}
	if c.RuntimeDir != env["AI_INSTRUCTIONS_RUNTIME_DIR"] || c.RepositorySource != "environment" {
		t.Fatalf("unexpected config: %#v", c)
	}
	if c.ConfigDir != filepath.Join(home, ".config/ai-instructions") {
		t.Fatalf("unexpected config dir: %s", c.ConfigDir)
	}
}

func TestLoadRequiresHome(t *testing.T) {
	if _, err := Load(func(string) (string, bool) { return "", false }, "aic", "help"); err == nil {
		t.Fatal("expected HOME error")
	}
}

func TestReadManagedValueRejectsSymlink(t *testing.T) {
	dir := t.TempDir()
	real := filepath.Join(dir, "real")
	if err := os.WriteFile(real, []byte(AgentsURLMarker+"\nhttps://example.test/AGENTS.md\n"), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadManagedValue(link, AgentsURLMarker); err == nil {
		t.Fatal("symlink must be rejected")
	}
}

func TestReadManagedValue(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agents-url")
	if err := os.WriteFile(path, []byte(AgentsURLMarker+"\nhttps://example.test/AGENTS.md\n"), 0600); err != nil {
		t.Fatal(err)
	}
	value, err := ReadManagedValue(path, AgentsURLMarker)
	if err != nil || value != "https://example.test/AGENTS.md" {
		t.Fatalf("value=%q err=%v", value, err)
	}
}

func TestLoadSavedAndDefaultRepository(t *testing.T) {
	home := t.TempDir()
	configDir := filepath.Join(home, ".config/ai-instructions")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	repo := filepath.Join(home, "saved repo")
	if err := os.WriteFile(filepath.Join(configDir, "repo-path"), []byte(RepoPathMarker+"\n"+repo+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	lookup := func(key string) (string, bool) {
		if key == "HOME" {
			return home, true
		}
		return "", false
	}
	c, err := Load(lookup, "/missing/aic", "status")
	if err != nil || c.RepositoryDir != repo || c.RepositorySource != "saved" {
		t.Fatalf("saved config: %#v err=%v", c, err)
	}
	if err := os.Remove(filepath.Join(configDir, "repo-path")); err != nil {
		t.Fatal(err)
	}
	c, err = Load(lookup, "/missing/aic", "status")
	if err != nil || c.RepositorySource != "default" || c.RepositoryDir != filepath.Join(home, ".local/share/ai-instructions") {
		t.Fatalf("default config: %#v err=%v", c, err)
	}
}

func TestLoadDetectsReleaseFromManagedDefaultPath(t *testing.T) {
	home := t.TempDir()
	configDir := filepath.Join(home, ".config/ai-instructions")
	installedDir := filepath.Join(configDir, "bin")
	defaultRepo := filepath.Join(home, ".local/share/ai-instructions")
	if err := os.MkdirAll(installedDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "repo-path"), []byte(RepoPathMarker+"\n"+defaultRepo+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(installedDir, "ai-instructions"), []byte("binary"), 0700); err != nil {
		t.Fatal(err)
	}
	lookup := func(key string) (string, bool) {
		if key == "HOME" {
			return home, true
		}
		return "", false
	}
	c, err := Load(lookup, filepath.Join(installedDir, "ai-instructions"), "status")
	if err != nil || c.RepositorySource != "release" || c.RepositoryDir != defaultRepo {
		t.Fatalf("release config: %#v err=%v", c, err)
	}
	if err := os.MkdirAll(filepath.Join(defaultRepo, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	c, err = Load(lookup, filepath.Join(installedDir, "ai-instructions"), "status")
	if err != nil || c.RepositorySource != "saved" {
		t.Fatalf("repository config: %#v err=%v", c, err)
	}
}

func TestDetectRepository(t *testing.T) {
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(repo, "bin"), 0755); err != nil {
		t.Fatal(err)
	}
	tool := filepath.Join(repo, "bin/ai-instructions")
	if err := os.WriteFile(tool, []byte("tool"), 0600); err != nil {
		t.Fatal(err)
	}
	got, ok := DetectRepository(tool)
	want, err := filepath.EvalSymlinks(repo)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || got != want {
		t.Fatalf("got=%q ok=%v", got, ok)
	}
}

func TestReadManagedValueRejectsMalformed(t *testing.T) {
	for _, content := range []string{"", "wrong\nvalue\n", AgentsURLMarker + "\n"} {
		path := filepath.Join(t.TempDir(), "value")
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := ReadManagedValue(path, AgentsURLMarker); err == nil {
			t.Fatalf("accepted %q", content)
		}
	}
}
