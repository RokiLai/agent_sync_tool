package integration

import (
	"strings"
	"testing"

	"github.com/RokiLai/agent_sync_tool/internal/config"
)

func TestShellInit(t *testing.T) {
	out := ShellInit(config.Config{Paths: config.Paths{BinDir: "/tmp/bin"}})
	for _, expected := range []string{config.ManagedMarker, `"/tmp/bin/agentsync" sync --auto || return`, "alias cdx=codex", "alias cld=claude", "alias ag=agy"} {
		if !strings.Contains(out, expected) {
			t.Errorf("missing %q", expected)
		}
	}
}
