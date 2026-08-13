package release

import (
	"context"
	"net/http"

	"github.com/RokiLai/agent_sync_tool/internal/core"
)

const DefaultBaseURL = core.DefaultReleaseBaseURL

func Artifact(goos, goarch string) (string, error) { return core.Artifact(goos, goarch) }
func CurrentArtifact() (string, error)             { return core.CurrentArtifact() }
func URL(base, version, name string) string        { return core.ReleaseURL(base, version, name) }
func Download(ctx context.Context, client *http.Client, rawURL string) ([]byte, error) {
	return core.DownloadRelease(ctx, client, rawURL)
}
func ParseChecksums(data []byte) map[string]string { return core.ParseChecksums(data) }
func Verify(data []byte, expected string) error    { return core.VerifyChecksum(data, expected) }
