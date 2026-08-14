package identity

import (
	"reflect"
	"testing"
)

func TestPublicNames(t *testing.T) {
	if PrimaryCommand != "agentsync" || PrimaryArtifactPrefix != "agentsync" {
		t.Fatalf("unexpected primary names: %q %q", PrimaryCommand, PrimaryArtifactPrefix)
	}
	if got, want := CommandNames(), []string{"agentsync", "aic", "ai-instructions"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("CommandNames() = %q, want %q", got, want)
	}
	if LegacyArtifactPrefix != "aic" || ManagedBinaryName != "ai-instructions" || VersionOutputName != "ai-instructions" {
		t.Fatal("legacy compatibility names changed")
	}
	if TransitionSeries != "v3.2.x" {
		t.Fatalf("TransitionSeries = %q", TransitionSeries)
	}
	if !IsLegacyCommand("aic") || !IsLegacyCommand("ai-instructions") || IsLegacyCommand("agentsync") {
		t.Fatal("legacy command classification is incorrect")
	}
}
