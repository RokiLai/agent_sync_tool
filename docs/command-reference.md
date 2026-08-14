# 命令参考

```text
aic <命令> [选项]
ai-instructions <命令> [选项]
```

两个命令名功能相同。

## `install`

```sh
aic install URL [--shell auto|zsh|bash|none] [--tools LIST] [--dry-run]
```

- `URL`：能够直接下载的单行 HTTP(S) `AGENTS.md` 地址；非交互安装必须提供。
- `--shell`：默认 `auto`，支持 `zsh`、`bash`、`none`。
- `--tools`：逗号分隔的 `codex`、`claude`、`agy`，默认全部。
- `--dry-run`：完成下载和预检，只展示结果，不写入目标。

安装会部署规则、安装工具本体、创建命令入口和选定的 AI 工具规则入口。

GitHub `blob/main`、`blob/master` 和 `blob/<40位提交SHA>` 文件页会自动转换为 raw 地址。转换后的地址用于下载并写入配置。其他 ref 不自动猜测；返回 `text/html` 的地址会被拒绝。

## `sync`

```sh
aic sync
```

从已保存来源下载并原子部署最新规则。成功信息写入 stderr，stdout 保持为空，便于 Shell wrapper 使用。

下载失败但已有有效 runtime 时返回成功并显示 last-known-good 警告；没有缓存时返回非零。

## `source`

查看当前来源：

```sh
aic source
aic source show
```

验证来源但不修改：

```sh
aic source test
aic source test URL
```

交互式更换来源：

```sh
aic source set URL
```

`source set` 要求真实终端。新 URL 与当前来源相同时直接成功；否则先下载和校验候选，再显示新旧来源并询问确认。只有 `y` 或 `Y` 会执行切换。

`source test` 和 `source set` 检测到可转换的 GitHub 文件页时，会显示规范化后的 raw URL；`source set` 确认后保存该 raw URL。

## `upgrade`

```sh
aic upgrade
```

先查询 GitHub Release，展示当前版本和最新正式版本。当前已是最新版本时直接结束，不下载完整二进制；发现新版本且运行于真实终端时，确认后才下载并安装。

交互终端使用单行动态进度条展示下载大小和百分比，并展示 SHA-256 校验、候选版本验证和原子安装状态。stdout 被重定向或在 CI 中运行时自动使用无 ANSI 控制符的逐行日志，并保持非交互自动升级兼容性。

可以用 `AIC_VERSION` 固定版本，用 `AIC_RELEASE_BASE_URL` 覆盖 Release 根地址。

## `status`

```sh
aic status
```

显示：

- 安装模式；
- `AGENTS.md` 来源；
- runtime 路径和 revision；
- Codex、Claude、Antigravity 入口状态。

## `doctor`

```sh
aic doctor
```

检查依赖、来源配置、runtime 内容一致性、AI 工具入口、已安装工具、命令入口和 Shell 集成。存在关键失败时返回非零；未安装的可选工具或可选入口以警告显示。

## `shell-init`

```sh
aic shell-init
```

输出 Zsh/Bash 可加载的集成脚本。安装器通常会将脚本保存到配置目录，并在对应的 `.zshrc` 或 `.bashrc` 中加入受管加载块。

集成脚本提供：

- `codex`、`claude`、`agy` wrapper；
- 启动工具前执行 `aic sync`；
- `cdx`、`cld`、`ag` 别名。

## `uninstall`

```sh
aic uninstall
```

要求交互式终端。命令先生成并展示完整计划，确认后严格执行同一份计划，不重新扫描目标。第一次确认删除受管安装对象，第二次确认是否删除 runtime；默认均为否。

## `version`

```sh
aic version
aic --version
aic -V
```

输出格式：

```text
ai-instructions 3.1.0
```

## `help`

```sh
aic help
aic --help
aic -h
```

## 环境变量

| 变量 | 作用 | 默认值 |
| --- | --- | --- |
| `AI_INSTRUCTIONS_RUNTIME_DIR` | runtime 根目录 | `~/.local/share/ai-instructions-runtime` |
| `AI_INSTRUCTIONS_CONFIG_DIR` | 配置和已安装工具目录 | `~/.config/ai-instructions` |
| `AI_INSTRUCTIONS_BIN_DIR` | 命令入口目录 | `~/.local/bin` |
| `CODEX_HOME` | Codex 配置目录 | `~/.codex` |
| `AI_INSTRUCTIONS_REPO` | 显式指定开发仓库路径 | 自动检测或默认路径 |
| `AIC_RELEASE_BASE_URL` | Release 下载根地址 | GitHub Releases |
| `AIC_VERSION` | 安装或升级的目标版本 | `latest` |

## 退出状态

- `0`：操作成功，或普通同步成功使用有效 last-known-good；
- 非零：参数错误、来源无效、严格同步失败、路径冲突、诊断存在关键失败、升级失败或卸载环境无效。
