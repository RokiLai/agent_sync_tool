package core

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/RokiLai/agent_sync_tool/internal/config"
	"github.com/RokiLai/agent_sync_tool/internal/identity"
)

func ShellInit(c config.Config) string {
	launcher := filepath.Join(c.BinDir, identity.PrimaryCommand)
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`%s
case ":$PATH:" in
    *":%s:"*) ;;
    *) PATH="%s:$PATH" ;;
esac
export PATH
`, config.ManagedMarker, c.BinDir, c.BinDir))

	enabledMap := map[string]bool{}
	for _, k := range c.EnabledTools {
		enabledMap[k] = true
	}
	if len(enabledMap) == 0 {
		for _, t := range identity.SupportedTools() {
			enabledMap[t.Key] = true
		}
	}

	var aliases []string
	for _, tool := range identity.SupportedTools() {
		if !enabledMap[tool.Key] {
			continue
		}
		sb.WriteString(fmt.Sprintf(`
%s() {
    "%s" sync --auto || return
    command %s "$@"
}
`, tool.BinaryName, launcher, tool.BinaryName))
		if tool.Alias != "" {
			aliases = append(aliases, fmt.Sprintf("alias %s=%s", tool.Alias, tool.BinaryName))
		}
	}

	if len(aliases) > 0 {
		sb.WriteString("\n" + strings.Join(aliases, "\n") + "\n")
	}

	return sb.String()
}
