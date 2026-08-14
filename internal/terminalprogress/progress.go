package terminalprogress

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/RokiLai/agent_sync_tool/internal/upgrade"
)

type Renderer struct {
	out         io.Writer
	interactive bool
	mu          sync.Mutex
	started     time.Time
	stage       string
}

func New(out io.Writer, interactive bool) *Renderer {
	return &Renderer{out: out, interactive: interactive}
}

func (r *Renderer) Update(p upgrade.Progress) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.interactive {
		r.line(p)
		return
	}
	if r.stage != p.Stage {
		r.started = time.Now()
		r.stage = p.Stage
	}
	label := label(p)
	if p.Stage == "download" && p.Total > 0 {
		percent := int(p.Downloaded * 100 / p.Total)
		if percent > 100 {
			percent = 100
		}
		width := 20
		filled := percent * width / 100
		speed := ""
		if elapsed := time.Since(r.started).Seconds(); elapsed >= 0.1 && p.Downloaded > 0 {
			speed = "  " + size(int64(float64(p.Downloaded)/elapsed)) + "/s"
		}
		fmt.Fprintf(r.out, "\r\033[2K%-24s [%s%s] %3d%%  %s/%s%s", label, strings.Repeat("█", filled), strings.Repeat("░", width-filled), percent, size(p.Downloaded), size(p.Total), speed)
	} else if p.Done {
		fmt.Fprintf(r.out, "\r\033[2K%-24s ✓", label)
	} else {
		frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		frame := frames[int(time.Since(r.started)/(100*time.Millisecond))%len(frames)]
		if p.Stage == "download" && p.Downloaded > 0 {
			fmt.Fprintf(r.out, "\r\033[2K%-24s %s  %s", label, frame, size(p.Downloaded))
		} else {
			fmt.Fprintf(r.out, "\r\033[2K%-24s %s", label, frame)
		}
	}
	if p.Done {
		fmt.Fprintln(r.out)
	}
}

func (r *Renderer) Fail() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.interactive && r.stage != "" {
		fmt.Fprint(r.out, "\r\033[2K")
	}
}

func (r *Renderer) line(p upgrade.Progress) {
	if !p.Done {
		return
	}
	switch p.Stage {
	case "download":
		fmt.Fprintf(r.out, "[OK] 下载完成：%s（%s）\n", p.Name, size(p.Downloaded))
	case "checksum":
		fmt.Fprintln(r.out, "[OK] SHA-256 校验通过")
	case "candidate":
		fmt.Fprintf(r.out, "[OK] 候选版本验证通过：%s\n", p.Name)
	case "install":
		fmt.Fprintf(r.out, "[OK] 安装完成：%s\n", p.Name)
	}
}

func label(p upgrade.Progress) string {
	switch p.Stage {
	case "download":
		return "下载 " + p.Name
	case "checksum":
		return "验证 SHA-256"
	case "candidate":
		return "验证候选版本 " + p.Name
	case "install":
		return "安装新版本 " + p.Name
	default:
		return p.Name
	}
}

func size(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	value := float64(bytes)
	units := []string{"KiB", "MiB", "GiB"}
	for _, name := range units {
		value /= unit
		if value < unit || name == units[len(units)-1] {
			return fmt.Sprintf("%.1f %s", value, name)
		}
	}
	return ""
}
