package identity

const (
	PrimaryCommand        = "agentsync"
	LegacyShortCommand    = "aic"
	LegacyLongCommand     = "ai-instructions"
	ManagedBinaryName     = "agentsync"
	VersionOutputName     = "ai-instructions"
	PrimaryArtifactPrefix = "agentsync"
	LegacyArtifactPrefix  = "aic"
)

func CommandNames() []string {
	return []string{PrimaryCommand}
}

func HistoricalCommandNames() []string {
	return []string{PrimaryCommand, LegacyShortCommand, LegacyLongCommand}
}

func IsLegacyCommand(name string) bool {
	return name == LegacyShortCommand || name == LegacyLongCommand
}
