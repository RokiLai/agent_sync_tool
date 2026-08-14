package upgrade

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/RokiLai/agent_sync_tool/internal/core"
	"github.com/RokiLai/agent_sync_tool/internal/identity"
)

type Options struct {
	Installed, BaseURL, Version, Artifact, CurrentVersion string
	Client                                                *http.Client
	Progress                                              func(Progress)
}

type Progress struct {
	Stage      string
	Name       string
	Downloaded int64
	Total      int64
	Done       bool
}

type Plan struct {
	CurrentVersion string
	TargetVersion  string
	Artifact       string
	checksums      []byte
}

type Result struct {
	Changed bool
	Version string
}

var releaseTagPattern = regexp.MustCompile(`^[vV]?[0-9]+\.[0-9]+\.[0-9]+(?:[-+][0-9A-Za-z.-]+)?$`)

func Check(ctx context.Context, o Options) (Plan, error) {
	o = defaults(o)
	if o.Artifact == "" {
		var err error
		o.Artifact, err = core.CurrentArtifact()
		if err != nil {
			return Plan{}, err
		}
	}
	if err := validateInstalled(o.Installed); err != nil {
		return Plan{}, err
	}
	checks, redirects, err := download(ctx, o.Client, core.ReleaseURL(o.BaseURL, o.Version, "checksums.txt"), nil)
	if err != nil {
		return Plan{}, fmt.Errorf("查询最新版本失败；当前工具保持不变: %w", err)
	}
	target := strings.TrimPrefix(o.Version, "v")
	if o.Version == "latest" {
		target = releaseVersion(o.BaseURL, redirects)
	}
	return Plan{CurrentVersion: strings.TrimPrefix(o.CurrentVersion, "v"), TargetVersion: target, Artifact: o.Artifact, checksums: checks}, nil
}

func Apply(ctx context.Context, o Options, plan Plan) (Result, error) {
	o = defaults(o)
	expected := core.ParseChecksums(plan.checksums)[plan.Artifact]
	if expected == "" {
		return Result{}, errors.New("checksum清单缺少当前平台产物；当前工具保持不变")
	}
	emit(o, Progress{Stage: "download", Name: plan.Artifact})
	candidate, _, err := download(ctx, o.Client, core.ReleaseURL(o.BaseURL, o.Version, plan.Artifact), func(downloaded, total int64) {
		emit(o, Progress{Stage: "download", Name: plan.Artifact, Downloaded: downloaded, Total: total})
	})
	if err != nil {
		return Result{}, fmt.Errorf("下载候选工具失败；当前工具保持不变: %w", err)
	}
	emit(o, Progress{Stage: "download", Name: plan.Artifact, Downloaded: int64(len(candidate)), Total: int64(len(candidate)), Done: true})
	emit(o, Progress{Stage: "checksum", Name: "SHA-256"})
	if err := core.VerifyChecksum(candidate, expected); err != nil {
		return Result{}, fmt.Errorf("%w；当前工具保持不变", err)
	}
	emit(o, Progress{Stage: "checksum", Name: "SHA-256", Done: true})

	dir := filepath.Dir(o.Installed)
	temp, err := os.CreateTemp(dir, "."+identity.PrimaryCommand+"-upgrade.*")
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
	emit(o, Progress{Stage: "candidate", Name: "候选版本"})
	out, err := exec.CommandContext(ctx, tempPath, "version").Output()
	versionPrefix := identity.VersionOutputName + " "
	if err != nil || !strings.HasPrefix(string(out), versionPrefix) {
		return Result{}, errors.New("候选工具校验失败；当前工具保持不变")
	}
	version := strings.TrimSpace(strings.TrimPrefix(string(out), versionPrefix))
	if plan.TargetVersion != "" && version != plan.TargetVersion {
		return Result{}, fmt.Errorf("候选版本不匹配：expected=%s actual=%s；当前工具保持不变", plan.TargetVersion, version)
	}
	emit(o, Progress{Stage: "candidate", Name: version, Done: true})
	current, err := os.ReadFile(o.Installed)
	if err != nil {
		return Result{}, err
	}
	if string(current) == string(candidate) {
		return Result{Version: version}, nil
	}
	emit(o, Progress{Stage: "install", Name: version})
	if err := os.Rename(tempPath, o.Installed); err != nil {
		return Result{}, fmt.Errorf("原子替换失败；当前工具保持不变: %w", err)
	}
	emit(o, Progress{Stage: "install", Name: version, Done: true})
	return Result{Changed: true, Version: version}, nil
}

func Run(ctx context.Context, o Options) (Result, error) {
	plan, err := Check(ctx, o)
	if err != nil {
		return Result{}, err
	}
	return Apply(ctx, o, plan)
}

func defaults(o Options) Options {
	if o.BaseURL == "" {
		o.BaseURL = core.DefaultReleaseBaseURL
	}
	if o.Version == "" {
		o.Version = "latest"
	}
	if o.Client == nil {
		o.Client = http.DefaultClient
	}
	return o
}

func validateInstalled(path string) error {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return errors.New("工具本体未安装、不是普通文件或不受管")
	}
	return nil
}

func download(ctx context.Context, client *http.Client, rawURL string, progress func(int64, int64)) ([]byte, []string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, nil, err
	}
	redirects := []string{rawURL}
	redirectClient := *client
	originalCheckRedirect := client.CheckRedirect
	redirectClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if originalCheckRedirect != nil {
			if err := originalCheckRedirect(req, via); err != nil {
				return err
			}
		} else if len(via) >= 10 {
			return errors.New("stopped after 10 redirects")
		}
		redirects = append(redirects, req.URL.String())
		return nil
	}
	resp, err := redirectClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, nil, fmt.Errorf("HTTP状态：%s", resp.Status)
	}
	reader := io.Reader(resp.Body)
	if progress != nil {
		reader = &progressReader{reader: resp.Body, total: resp.ContentLength, notify: progress}
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, nil, err
	}
	if len(data) == 0 {
		return nil, nil, errors.New("下载内容为空")
	}
	return data, redirects, nil
}

func releaseVersion(base string, redirects []string) string {
	baseURL, err := url.Parse(strings.TrimSuffix(base, "/"))
	if err != nil || baseURL.Scheme == "" || baseURL.Host == "" {
		return ""
	}
	prefix := strings.TrimSuffix(baseURL.Path, "/") + "/download/"
	const suffix = "/checksums.txt"
	for _, raw := range redirects {
		candidate, err := url.Parse(raw)
		if err != nil || !strings.EqualFold(candidate.Scheme, baseURL.Scheme) || !strings.EqualFold(candidate.Host, baseURL.Host) {
			continue
		}
		if !strings.HasPrefix(candidate.Path, prefix) || !strings.HasSuffix(candidate.Path, suffix) {
			continue
		}
		tag := strings.TrimSuffix(strings.TrimPrefix(candidate.Path, prefix), suffix)
		if strings.Contains(tag, "/") || !releaseTagPattern.MatchString(tag) {
			continue
		}
		return strings.TrimPrefix(strings.TrimPrefix(tag, "v"), "V")
	}
	return ""
}

type progressReader struct {
	reader     io.Reader
	total      int64
	downloaded int64
	notify     func(int64, int64)
}

func (r *progressReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	r.downloaded += int64(n)
	if n > 0 {
		r.notify(r.downloaded, r.total)
	}
	return n, err
}

func emit(o Options, p Progress) {
	if o.Progress != nil {
		o.Progress(p)
	}
}
