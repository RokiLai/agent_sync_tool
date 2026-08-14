package identity

const (
	PrimaryCommand        = "agentsync"
	LegacyShortCommand    = "aic"
	LegacyLongCommand     = "ai-instructions"
	ManagedBinaryName     = "ai-instructions"
	VersionOutputName     = "ai-instructions"
	PrimaryArtifactPrefix = "agentsync"
	LegacyArtifactPrefix  = "aic"
	TransitionSeries      = "v3.2.x"
)

func CommandNames() []string {
	return []string{PrimaryCommand, LegacyShortCommand, LegacyLongCommand}
}

func IsLegacyCommand(name string) bool {
	return name == LegacyShortCommand || name == LegacyLongCommand
}
