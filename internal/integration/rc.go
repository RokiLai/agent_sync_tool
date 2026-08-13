package integration

import "github.com/RokiLai/agent_sync_tool/internal/core"

func RCPath(home, shell string) string       { return core.RCPath(home, shell) }
func ValidateRC(path string) error           { return core.ValidateRC(path) }
func InstallRC(path, shellFile string) error { return core.InstallRC(path, shellFile) }
func RemoveRC(path string) error             { return core.RemoveRC(path) }
