package terminalprogress

import (
	"bytes"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/RokiLai/agent_sync_tool/internal/upgrade"
)

func TestInteractiveDownloadProgress(t *testing.T) {
	var out bytes.Buffer
	r := New(&out, true)
	r.Update(upgrade.Progress{Stage: "download", Name: "agentsync_Darwin_arm64", Downloaded: 512, Total: 1024})
	r.Update(upgrade.Progress{Stage: "download", Name: "agentsync_Darwin_arm64", Downloaded: 1024, Total: 1024, Done: true})
	got := out.String()
	for _, want := range []string{"\r\033[2K", "50%", "100%", "█"} {
		if !strings.Contains(got, want) {
			t.Fatalf("output %q missing %q", got, want)
		}
	}
}

func TestNonInteractiveProgressHasStableLines(t *testing.T) {
	var out bytes.Buffer
	r := New(&out, false)
	r.Update(upgrade.Progress{Stage: "download", Name: "agentsync_Linux_x86_64", Downloaded: 2048, Total: 2048})
	r.Update(upgrade.Progress{Stage: "download", Name: "agentsync_Linux_x86_64", Downloaded: 2048, Total: 2048, Done: true})
	r.Update(upgrade.Progress{Stage: "checksum", Name: "SHA-256", Done: true})
	got := out.String()
	if strings.Contains(got, "\r") || strings.Contains(got, "\033[") || !strings.Contains(got, "下载完成") || !strings.Contains(got, "SHA-256 校验通过") {
		t.Fatalf("unexpected output %q", got)
	}
}

func TestInteractiveFailureClearsDynamicLine(t *testing.T) {
	var out bytes.Buffer
	r := New(&out, true)
	r.Update(upgrade.Progress{Stage: "download", Name: "agentsync", Downloaded: 10})
	r.Fail()
	if !strings.HasSuffix(out.String(), "\r\033[2K") {
		t.Fatalf("failure did not clear line: %q", out.String())
	}
}

func TestInteractiveStageLabelsAndUnknownLength(t *testing.T) {
	var out bytes.Buffer
	r := New(&out, true)
	r.Update(upgrade.Progress{Stage: "download", Name: "agentsync", Downloaded: 2048, Total: -1})
	for _, event := range []upgrade.Progress{
		{Stage: "checksum", Name: "SHA-256", Done: true},
		{Stage: "candidate", Name: "3.2.0", Done: true},
		{Stage: "install", Name: "3.2.0", Done: true},
	} {
		r.Update(event)
	}
	got := out.String()
	for _, want := range []string{"2.0 KiB", "验证 SHA-256", "验证候选版本 3.2.0", "安装新版本 3.2.0"} {
		if !strings.Contains(got, want) {
			t.Fatalf("output %q missing %q", got, want)
		}
	}
}

type safeBuffer struct {
	b  bytes.Buffer
	mu sync.Mutex
}

func (s *safeBuffer) Write(p []byte) (n int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *safeBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

func TestInteractiveSpinnerCyclesAndClearsOnStop(t *testing.T) {
	var out safeBuffer
	s := StartSpinner(&out, true, "正在检查更新...")
	time.Sleep(180 * time.Millisecond)
	s.Stop()
	s.Stop() // idempotent

	got := out.String()
	if !strings.Contains(got, "正在检查更新...") {
		t.Fatalf("expected message in output: %q", got)
	}
	if !strings.Contains(got, "\r\033[2K") {
		t.Fatalf("expected ANSI line clear code: %q", got)
	}
	if !strings.HasSuffix(got, "\r\033[2K") {
		t.Fatalf("expected stopped spinner to end with line clear: %q", got)
	}
}

func TestNonInteractiveSpinnerPrintsStaticLine(t *testing.T) {
	var out bytes.Buffer
	s := StartSpinner(&out, false, "正在检查更新...")
	s.Stop()

	got := out.String()
	if got != "正在检查更新...\n" {
		t.Fatalf("unexpected non-interactive output: %q", got)
	}
	if strings.Contains(got, "\033[") || strings.Contains(got, "\r") {
		t.Fatalf("non-interactive output must not contain ANSI escapes: %q", got)
	}
}

func TestNonInteractiveSpinnerEmptyMessageNoOutput(t *testing.T) {
	var out bytes.Buffer
	s := StartSpinner(&out, false, "")
	s.Stop()

	if out.Len() != 0 {
		t.Fatalf("expected empty output, got: %q", out.String())
	}
}
