package core

import (
	"fmt"
	"path/filepath"

	"github.com/RokiLai/agent_sync_tool/internal/config"
)

func ShellInit(c config.Config) string {
	launcher := filepath.Join(c.BinDir, "ai-instructions")
	return fmt.Sprintf(`%s
case ":$PATH:" in
    *":%s:"*) ;;
    *) PATH="%s:$PATH" ;;
esac
export PATH

codex() {
    "%s" sync || return
    command codex "$@"
}

claude() {
    "%s" sync || return
    command claude "$@"
}

agy() {
    "%s" sync || return
    command agy "$@"
}

alias cdx=codex
alias cld=claude
alias ag=agy
`, config.ManagedMarker, c.BinDir, c.BinDir, launcher, launcher, launcher)
}
