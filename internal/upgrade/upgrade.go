package upgrade

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/RokiLai/agent_sync_tool/internal/core"
)

type Options struct {
	Installed, BaseURL, Version, Artifact string
	Client                                *http.Client
}
type Result struct {
	Changed bool
	Version string
}

func Run(ctx context.Context, o Options) (Result, error) {
	if o.BaseURL == "" {
		o.BaseURL = core.DefaultReleaseBaseURL
	}
	if o.Version == "" {
		o.Version = "latest"
	}
	if o.Client == nil {
		o.Client = http.DefaultClient
	}
	if o.Artifact == "" {
		var err error
		o.Artifact, err = core.CurrentArtifact()
		if err != nil {
			return Result{}, err
		}
	}
	info, err := os.Lstat(o.Installed)
	if err != nil || !info.Mode().IsRegular() {
		return Result{}, errors.New("工具本体未安装、不是普通文件或不受管")
	}
	checks, err := core.DownloadRelease(ctx, o.Client, core.ReleaseURL(o.BaseURL, o.Version, "checksums.txt"))
	if err != nil {
		return Result{}, fmt.Errorf("下载checksum失败；当前工具保持不变: %w", err)
	}
	expected := core.ParseChecksums(checks)[o.Artifact]
	if expected == "" {
		return Result{}, errors.New("checksum清单缺少当前平台产物；当前工具保持不变")
	}
	candidate, err := core.DownloadRelease(ctx, o.Client, core.ReleaseURL(o.BaseURL, o.Version, o.Artifact))
	if err != nil {
		return Result{}, fmt.Errorf("下载候选工具失败；当前工具保持不变: %w", err)
	}
	if err := core.VerifyChecksum(candidate, expected); err != nil {
		return Result{}, fmt.Errorf("%w；当前工具保持不变", err)
	}
	dir := filepath.Dir(o.Installed)
	temp, err := os.CreateTemp(dir, ".aic-upgrade.*")
	if err != nil {
		return Result{}, err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if _, err = temp.Write(candidate); err == nil {
		err = temp.Chmod(0700)
	}
	if closeErr := temp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return Result{}, err
	}
	out, err := exec.CommandContext(ctx, tempPath, "version").Output()
	if err != nil || !strings.HasPrefix(string(out), "ai-instructions ") {
		return Result{}, errors.New("候选工具校验失败；当前工具保持不变")
	}
	version := strings.TrimSpace(strings.TrimPrefix(string(out), "ai-instructions "))
	current, err := os.ReadFile(o.Installed)
	if err != nil {
		return Result{}, err
	}
	if string(current) == string(candidate) {
		return Result{Version: version}, nil
	}
	if err := os.Rename(tempPath, o.Installed); err != nil {
		return Result{}, fmt.Errorf("原子替换失败；当前工具保持不变: %w", err)
	}
	return Result{Changed: true, Version: version}, nil
}
