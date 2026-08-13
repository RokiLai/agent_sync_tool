package runtime

import "github.com/RokiLai/agent_sync_tool/internal/core"

func Revision(data []byte) string { return core.Revision(data) }

type State = core.State

func Inspect(dir string) State { return core.InspectRuntime(dir) }

func Size(data []byte) string { return core.Size(data) }
