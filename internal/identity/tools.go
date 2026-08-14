package identity

import (
	"path/filepath"
)

type ToolSpec struct {
	Key         string
	DisplayName string
	BinaryName  string
	Alias       string
	HomeSubDir  string
	TargetFile  string
}

func (t ToolSpec) TargetPath(homeDir, codexHome string) string {
	if t.Key == "codex" {
		return filepath.Join(codexHome, t.TargetFile)
	}
	return filepath.Join(homeDir, t.HomeSubDir, t.TargetFile)
}

func SupportedTools() []ToolSpec {
	return []ToolSpec{
		{
			Key:         "codex",
			DisplayName: "Codex",
			BinaryName:  "codex",
			Alias:       "cdx",
			HomeSubDir:  ".codex",
			TargetFile:  "AGENTS.md",
		},
		{
			Key:         "claude",
			DisplayName: "Claude",
			BinaryName:  "claude",
			Alias:       "cld",
			HomeSubDir:  ".claude",
			TargetFile:  "CLAUDE.md",
		},
		{
			Key:         "agy",
			DisplayName: "Antigravity",
			BinaryName:  "agy",
			Alias:       "ag",
			HomeSubDir:  ".gemini",
			TargetFile:  "GEMINI.md",
		},
	}
}

func DefaultToolKeys() []string {
	tools := SupportedTools()
	keys := make([]string, len(tools))
	for i, t := range tools {
		keys[i] = t.Key
	}
	return keys
}

func FindTool(key string) (ToolSpec, bool) {
	for _, t := range SupportedTools() {
		if t.Key == key {
			return t, true
		}
	}
	return ToolSpec{}, false
}
