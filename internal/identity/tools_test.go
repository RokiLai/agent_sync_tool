package identity

import (
	"path/filepath"
	"testing"
)

func TestSupportedTools(t *testing.T) {
	tools := SupportedTools()
	if len(tools) != 3 {
		t.Fatalf("expected 3 tools, got %d", len(tools))
	}
	keys := DefaultToolKeys()
	if len(keys) != 3 || keys[0] != "codex" || keys[1] != "claude" || keys[2] != "agy" {
		t.Fatalf("unexpected DefaultToolKeys: %v", keys)
	}
}

func TestFindTool(t *testing.T) {
	tool, ok := FindTool("claude")
	if !ok || tool.DisplayName != "Claude" || tool.Alias != "cld" {
		t.Fatalf("unexpected FindTool claude result: %+v, ok=%v", tool, ok)
	}
	_, ok = FindTool("nonexistent")
	if ok {
		t.Fatalf("expected false for nonexistent tool")
	}
}

func TestTargetPath(t *testing.T) {
	home := "/Users/test"
	codexHome := "/Users/test/.custom_codex"

	codex, _ := FindTool("codex")
	if got := codex.TargetPath(home, codexHome); got != filepath.Join(codexHome, "AGENTS.md") {
		t.Fatalf("unexpected codex path: %s", got)
	}

	claude, _ := FindTool("claude")
	if got := claude.TargetPath(home, codexHome); got != filepath.Join(home, ".claude/CLAUDE.md") {
		t.Fatalf("unexpected claude path: %s", got)
	}

	agy, _ := FindTool("agy")
	if got := agy.TargetPath(home, codexHome); got != filepath.Join(home, ".gemini/GEMINI.md") {
		t.Fatalf("unexpected agy path: %s", got)
	}
}
