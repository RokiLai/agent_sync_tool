package identity

import (
	"reflect"
	"testing"
)

func TestPublicNames(t *testing.T) {
	if PrimaryCommand != "agentsync" || PrimaryArtifactPrefix != "agentsync" {
		t.Fatalf("unexpected primary names: %q %q", PrimaryCommand, PrimaryArtifactPrefix)
	}
	if got, want := CommandNames(), []string{"agentsync"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("CommandNames() = %q, want %q", got, want)
	}
	if got, want := HistoricalCommandNames(), []string{"agentsync", "aic", "ai-instructions"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("HistoricalCommandNames() = %q, want %q", got, want)
	}
	if LegacyArtifactPrefix != "aic" || ManagedBinaryName != "agentsync" || VersionOutputName != "ai-instructions" {
		t.Fatal("legacy compatibility names changed")
	}
	if !IsLegacyCommand("aic") || !IsLegacyCommand("ai-instructions") || IsLegacyCommand("agentsync") {
		t.Fatal("legacy command classification is incorrect")
	}
}
