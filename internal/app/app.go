package app

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/RokiLai/agent_sync_tool/internal/config"
	"github.com/RokiLai/agent_sync_tool/internal/diagnose"
	"github.com/RokiLai/agent_sync_tool/internal/install"
	"github.com/RokiLai/agent_sync_tool/internal/integration"
	"github.com/RokiLai/agent_sync_tool/internal/lock"
	"github.com/RokiLai/agent_sync_tool/internal/managedfs"
	"github.com/RokiLai/agent_sync_tool/internal/runtime"
	"github.com/RokiLai/agent_sync_tool/internal/source"
	"github.com/RokiLai/agent_sync_tool/internal/uninstall"
	"github.com/RokiLai/agent_sync_tool/internal/upgrade"
)

var Version = "3.0.1"

type Dependencies struct {
	Stdin          io.Reader
	Stdout, Stderr io.Writer
	LookupEnv      config.LookupEnv
	Executable     string
	HTTPClient     *http.Client
	Diagnose       diagnose.Dependencies
	IsTerminal     func() bool
}

func Main(ctx context.Context, args []string, deps Dependencies) int {
	if deps.Stdout == nil {
		deps.Stdout = io.Discard
	}
	if deps.Stdin == nil {
		deps.Stdin = strings.NewReader("")
	}
	if deps.IsTerminal == nil {
		deps.IsTerminal = func() bool { return false }
	}
	if deps.Stderr == nil {
		deps.Stderr = io.Discard
	}
	if deps.LookupEnv == nil {
		deps.LookupEnv = os.LookupEnv
	}
	if deps.HTTPClient == nil {
		deps.HTTPClient = source.NewHTTPClient()
	}
	if deps.Diagnose.LookPath == nil {
		deps.Diagnose = diagnose.DefaultDependencies()
	}
	command := "help"
	if len(args) > 0 {
		command, args = args[0], args[1:]
	}
	c, err := config.Load(deps.LookupEnv, deps.Executable, command)
	if err != nil {
		return fail(deps.Stderr, err.Error())
	}
	switch command {
	case "help", "-h", "--help":
		if len(args) != 0 {
			return fail(deps.Stderr, "help 不接受参数")
		}
		fmt.Fprint(deps.Stdout, Usage)
	case "version", "--version", "-V":
		if len(args) != 0 {
			return fail(deps.Stderr, "version 不接受参数")
		}
		fmt.Fprintf(deps.Stdout, "ai-instructions %s\n", Version)
	case "source":
		return sourceCommand(ctx, args, c, deps)
	case "install":
		shellEnv, _ := deps.LookupEnv("SHELL")
		options, err := install.Parse(args, shellEnv)
		if err != nil {
			return fail(deps.Stderr, err.Error())
		}
		options.Executable = deps.Executable
		if options.URL == "" {
			if raw, readErr := diagnose.ReadAgentsURL(c); readErr == nil {
				options.URL = raw
			}
		}
		installer := install.Installer{Config: c, Download: func(ctx context.Context, raw string) ([]byte, error) {
			return source.Download(ctx, deps.HTTPClient, raw)
		}}
		plan, err := installer.Prepare(ctx, options)
		if err != nil {
			return fail(deps.Stderr, fmt.Sprintf("安装预检失败：%v；未修改任何文件", err))
		}
		fmt.Fprintf(deps.Stdout, "[INFO] 工具仓库：%s\n[INFO] AGENTS.md 来源：%s\n[INFO] runtime：%s\n[INFO] Shell：%s\n", c.RepositoryDir, options.URL, c.RuntimeDir, options.Shell)
		if options.DryRun {
			fmt.Fprintln(deps.Stdout, "[INFO] 当前为 dry-run，不会修改文件")
			fmt.Fprintln(deps.Stdout, "[INFO] dry-run 完成")
			return 0
		}
		if err := install.Execute(plan, c); err != nil {
			return fail(deps.Stderr, err.Error())
		}
		if options.Shell == "none" {
			fmt.Fprintln(deps.Stdout, "[OK] 安装完成；未配置 Shell wrapper")
		} else {
			fmt.Fprintf(deps.Stdout, "[OK] 安装完成；请新开终端或执行：. %s\n", plan.ShellRC)
		}
	case "sync":
		if len(args) != 0 {
			return fail(deps.Stderr, "sync 不接受参数")
		}
		raw, err := diagnose.ReadAgentsURL(c)
		if err != nil {
			return fail(deps.Stderr, "未配置有效的 AGENTS.md 来源；请运行 aic source set <URL>")
		}
		syncer := newSyncer(c, deps)
		state, err := syncer.Sync(ctx, raw, false)
		if errors.Is(err, runtime.ErrUsingCache) {
			fmt.Fprintf(deps.Stderr, "警告：AGENTS.md 下载或校验失败；使用最后一次成功部署的有效缓存：%s\n", state.Revision)
			return 0
		}
		if err != nil {
			return fail(deps.Stderr, err.Error())
		}
		fmt.Fprintf(deps.Stderr, "AI 指令已同步：%s\n", state.Revision)
	case "status":
		if len(args) != 0 {
			return fail(deps.Stderr, "status 不接受参数")
		}
		diagnose.Status(deps.Stdout, c, deps.Diagnose)
	case "doctor":
		if len(args) != 0 {
			return fail(deps.Stderr, "doctor 不接受参数")
		}
		shell, _ := deps.LookupEnv("SHELL")
		if !diagnose.Doctor(deps.Stdout, c, deps.Diagnose, shell) {
			return 1
		}
	case "shell-init":
		if len(args) != 0 {
			return fail(deps.Stderr, "shell-init 不接受参数")
		}
		fmt.Fprint(deps.Stdout, integration.ShellInit(c))
	case "uninstall":
		if len(args) != 0 {
			return fail(deps.Stderr, "uninstall 不接受参数")
		}
		if !deps.IsTerminal() {
			return fail(deps.Stderr, "uninstall requires an interactive terminal.")
		}
		shellEnv, _ := deps.LookupEnv("SHELL")
		plan := uninstall.Build(c, filepath.Base(shellEnv))
		uninstall.Print(deps.Stdout, plan)
		reader := bufio.NewReader(deps.Stdin)
		fmt.Fprint(deps.Stderr, "是否继续卸载？ [y/N] ")
		answer, _ := reader.ReadString('\n')
		if !yes(answer) {
			fmt.Fprintln(deps.Stdout, "[INFO] 已取消卸载")
			return 0
		}
		fmt.Fprintf(deps.Stderr, "是否同时删除规则缓存 runtime？\n%s\n[y/N] ", c.RuntimeDir)
		purgeAnswer, _ := reader.ReadString('\n')
		purge := yes(purgeAnswer)
		if err := uninstall.Execute(plan, purge); err != nil {
			return fail(deps.Stderr, err.Error())
		}
		fmt.Fprintln(deps.Stdout, "[OK] 卸载完成。")
		if purge {
			fmt.Fprintln(deps.Stdout, "[INFO] 规则 runtime 已删除。")
		} else {
			fmt.Fprintf(deps.Stdout, "规则 runtime 已保留：\n%s\n", c.RuntimeDir)
		}
	case "upgrade":
		if len(args) != 0 {
			return fail(deps.Stderr, "upgrade 不接受参数")
		}
		base, _ := deps.LookupEnv("AIC_RELEASE_BASE_URL")
		version, _ := deps.LookupEnv("AIC_VERSION")
		installed := filepath.Join(c.ConfigDir, "bin/ai-instructions")
		result, err := upgrade.Run(ctx, upgrade.Options{Installed: installed, BaseURL: base, Version: version, Client: deps.HTTPClient})
		if err != nil {
			return fail(deps.Stderr, err.Error())
		}
		if result.Changed {
			fmt.Fprintf(deps.Stdout, "[OK] 工具升级完成：%s\n", result.Version)
		} else {
			fmt.Fprintf(deps.Stdout, "[OK] 工具已是最新版本：%s\n", result.Version)
		}
	default:
		return fail(deps.Stderr, fmt.Sprintf("未知命令：%s（运行 ai-instructions help 查看帮助）", command))
	}
	return 0
}

func sourceCommand(ctx context.Context, args []string, c config.Config, deps Dependencies) int {
	action := "show"
	if len(args) > 0 {
		action, args = args[0], args[1:]
	}
	switch action {
	case "show":
		if len(args) != 0 {
			return fail(deps.Stderr, "source show 不接受参数")
		}
		raw, err := diagnose.ReadAgentsURL(c)
		if err != nil {
			return fail(deps.Stderr, "未配置有效的 AGENTS.md 来源")
		}
		fmt.Fprintf(deps.Stdout, "当前 AGENTS.md 来源：\n%s\n", raw)
	case "test":
		if len(args) > 1 {
			return fail(deps.Stderr, "source test 最多接受一个 URL")
		}
		raw := ""
		if len(args) == 1 {
			raw = args[0]
		} else {
			var err error
			raw, err = diagnose.ReadAgentsURL(c)
			if err != nil {
				return fail(deps.Stderr, "未配置有效的 AGENTS.md 来源")
			}
		}
		if err := source.ValidateURL(raw); err != nil {
			return fail(deps.Stderr, err.Error())
		}
		fmt.Fprintf(deps.Stdout, "正在检查：\n%s\n", raw)
		data, err := source.Download(ctx, deps.HTTPClient, raw)
		if err != nil {
			fmt.Fprintf(deps.Stderr, "来源检查失败：\n%s\n", raw)
			return fail(deps.Stderr, "原因：下载或校验失败")
		}
		fmt.Fprintf(deps.Stdout, "来源有效。\n内容大小：%s bytes\n内容版本：%s\n", runtime.Size(data), runtime.Revision(data))
	case "set":
		if len(args) != 1 {
			return fail(deps.Stderr, "用法：aic source set <URL>")
		}
		newURL := args[0]
		if err := source.ValidateURL(newURL); err != nil {
			return fail(deps.Stderr, err.Error())
		}
		oldURL, err := diagnose.ReadAgentsURL(c)
		if err != nil {
			return fail(deps.Stderr, "未配置有效的 AGENTS.md 来源")
		}
		if newURL == oldURL {
			fmt.Fprintln(deps.Stdout, "[INFO] 新来源与当前来源相同，无需修改。")
			return 0
		}
		if !deps.IsTerminal() {
			return fail(deps.Stderr, "source set requires an interactive terminal.")
		}
		fmt.Fprintf(deps.Stdout, "当前来源：\n%s\n新来源：\n%s\n正在验证新来源...\n", oldURL, newURL)
		data, err := source.Download(ctx, deps.HTTPClient, newURL)
		if err != nil {
			return fail(deps.Stderr, "新来源验证失败；当前来源和 runtime 保持不变")
		}
		candidate, err := runtime.NewCandidate(data)
		if err != nil {
			return fail(deps.Stderr, "新来源验证失败；当前来源和 runtime 保持不变")
		}
		fmt.Fprintf(deps.Stdout, "验证成功。\n内容大小：%s bytes\n内容版本：%s\n", runtime.Size(candidate.Data), candidate.Revision)
		fmt.Fprint(deps.Stderr, "是否切换规则来源？ [y/N] ")
		line, _ := bufio.NewReader(deps.Stdin).ReadString('\n')
		if strings.TrimSpace(line) != "y" && strings.TrimSpace(line) != "Y" {
			fmt.Fprintln(deps.Stdout, "[INFO] 已取消，当前来源未修改。")
			return 0
		}
		if err := commitSource(ctx, c, newURL, candidate); err != nil {
			return fail(deps.Stderr, "新来源切换失败；当前来源和 runtime 已保持不变")
		}
		fmt.Fprintln(deps.Stdout, "[OK] AGENTS.md 来源已更新")
	default:
		return fail(deps.Stderr, fmt.Sprintf("未知 source 操作：%s（支持 show、test、set）", action))
	}
	return 0
}

func newSyncer(c config.Config, deps Dependencies) runtime.Syncer {
	return runtime.Syncer{RuntimeDir: c.RuntimeDir, Download: func(ctx context.Context, raw string) ([]byte, error) {
		return source.Download(ctx, deps.HTTPClient, raw)
	}, LockOptions: lock.Options{}}
}

func commitSource(ctx context.Context, c config.Config, rawURL string, candidate runtime.Candidate) error {
	path := filepath.Join(c.ConfigDir, "agents-url")
	old, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if _, err := config.ReadManagedValue(path, config.AgentsURLMarker); err != nil {
		return err
	}
	l, err := lock.Acquire(ctx, c.RuntimeDir+".lock", lock.Options{})
	if err != nil {
		return err
	}
	defer l.Release()
	oldCurrent, oldCurrentErr := os.Readlink(filepath.Join(c.RuntimeDir, "current"))
	newData := []byte(fmt.Sprintf("%s\n%s\n", config.AgentsURLMarker, rawURL))
	if err := managedfs.AtomicWrite(path, newData, 0600); err != nil {
		return err
	}
	if err := (runtime.Publisher{Dir: c.RuntimeDir}).Publish(candidate); err != nil {
		if restoreErr := managedfs.AtomicWrite(path, old, 0600); restoreErr != nil {
			return fmt.Errorf("runtime 切换失败，且无法恢复原来源配置: %v", restoreErr)
		}
		if oldCurrentErr == nil {
			_ = managedfs.AtomicSymlink(filepath.Join(c.RuntimeDir, "current"), oldCurrent)
		}
		return err
	}
	return nil
}

func fail(w io.Writer, message string) int { fmt.Fprintf(w, "错误：%s\n", message); return 1 }
func yes(value string) bool                { value = strings.TrimSpace(value); return value == "y" || value == "Y" }

const Usage = `用法：aic <命令> [选项]
完整命令名：ai-instructions

命令：
  install      安装工具、同步规则、创建 AI 入口并配置 Shell
  sync         从已保存的 HTTP(S) 链接原子部署最新 AGENTS.md
  source       查看、验证或更换 AGENTS.md 来源链接
  upgrade      从 origin/main 安全升级工具本体
  status       显示仓库、runtime 和入口状态
  doctor       检查依赖、版本一致性、入口和 Shell 配置
  shell-init   输出 Zsh/Bash wrapper，可供 source/eval 使用
  uninstall    删除本工具明确管理的入口和 Shell 配置
  version      显示版本

install 选项：
  install [URL] [选项]         URL 是可直接下载的 AGENTS.md 链接
  --shell auto|zsh|bash|none   默认 auto
  --tools LIST                 codex,claude,agy 的逗号列表，默认全部
  --dry-run                    只显示计划，不修改

source：
  source                       查看当前 AGENTS.md 来源
  source test [URL]            验证当前来源或候选 URL，不作修改
  source set URL               验证并交互式切换 AGENTS.md 来源

uninstall：
  交互预览完整卸载计划，确认后可选删除 runtime

环境变量：
  AI_INSTRUCTIONS_REPO
  AI_INSTRUCTIONS_RUNTIME_DIR
  AI_INSTRUCTIONS_CONFIG_DIR
  AI_INSTRUCTIONS_BIN_DIR
  CODEX_HOME
`
