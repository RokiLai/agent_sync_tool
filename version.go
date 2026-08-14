package agentsynctool

import (
	_ "embed"
	"strings"
)

//go:embed VERSION
var rawVersion string

// Version is the application version read from VERSION or injected via ldflags.
var Version = ""

func init() {
	if Version == "" {
		Version = strings.TrimSpace(rawVersion)
	}
}
