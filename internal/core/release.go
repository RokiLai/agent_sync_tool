package core

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"

	"github.com/RokiLai/agent_sync_tool/internal/identity"
)

const DefaultReleaseBaseURL = "https://github.com/RokiLai/agent_sync_tool/releases"

func Artifact(goos, goarch string) (string, error) {
	osName := map[string]string{"darwin": "Darwin", "linux": "Linux"}[goos]
	arch := map[string]string{"amd64": "x86_64", "arm64": "arm64"}[goarch]
	if osName == "" || arch == "" {
		return "", fmt.Errorf("不支持的平台：%s/%s", goos, goarch)
	}
	return identity.PrimaryArtifactPrefix + "_" + osName + "_" + arch, nil
}

func LegacyArtifact(goos, goarch string) (string, error) {
	name, err := Artifact(goos, goarch)
	if err != nil {
		return "", err
	}
	return strings.Replace(name, identity.PrimaryArtifactPrefix+"_", identity.LegacyArtifactPrefix+"_", 1), nil
}

func CurrentArtifact() (string, error) { return Artifact(runtime.GOOS, runtime.GOARCH) }

func ReleaseURL(base, version, name string) string {
	base = strings.TrimSuffix(base, "/")
	if version == "" || version == "latest" {
		return base + "/latest/download/" + name
	}
	return base + "/download/" + version + "/" + name
}

func DownloadRelease(ctx context.Context, client *http.Client, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP状态：%s", resp.Status)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, errors.New("下载内容为空")
	}
	return data, nil
}

func ParseChecksums(data []byte) map[string]string {
	result := map[string]string{}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 {
			result[strings.TrimPrefix(fields[1], "*")] = fields[0]
		}
	}
	return result
}

func VerifyChecksum(data []byte, expected string) error {
	sum := sha256.Sum256(data)
	actual := hex.EncodeToString(sum[:])
	if !strings.EqualFold(actual, expected) {
		return fmt.Errorf("SHA-256校验失败：expected=%s actual=%s", expected, actual)
	}
	return nil
}
