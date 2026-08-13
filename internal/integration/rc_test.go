package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RokiLai/agent_sync_tool/internal/config"
)

func TestInstallRemoveRC(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".zshrc")
	if err := os.WriteFile(path, []byte("user\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := InstallRC(path, "/tmp/shell.sh"); err != nil {
		t.Fatal(err)
	}
	if err := InstallRC(path, "/tmp/shell.sh"); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if strings.Count(string(data), config.BlockBegin) != 1 {
		t.Fatalf("data=%s", data)
	}
	if err := RemoveRC(path); err != nil {
		t.Fatal(err)
	}
	data, _ = os.ReadFile(path)
	if strings.Contains(string(data), config.BlockBegin) || !strings.Contains(string(data), "user") {
		t.Fatalf("data=%s", data)
	}
}
func TestValidateIncompleteRC(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rc")
	if err := os.WriteFile(path, []byte(config.BlockBegin+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if ValidateRC(path) == nil {
		t.Fatal("expected error")
	}
	if RemoveRC(path) == nil {
		t.Fatal("expected error")
	}
}
func TestRCPath(t *testing.T) {
	if RCPath("/home/x", "zsh") != "/home/x/.zshrc" || RCPath("/home/x", "none") != "" {
		t.Fatal("unexpected rc path")
	}
}
