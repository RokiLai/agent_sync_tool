package integration

import (
	"github.com/RokiLai/agent_sync_tool/internal/config"
	"github.com/RokiLai/agent_sync_tool/internal/core"
)

func ShellInit(c config.Config) string { return core.ShellInit(c) }
