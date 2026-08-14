# Agent Sync Tool

Agent Sync Tool 是一个用于集中管理 AI 编程助手规则的命令行工具。它从一个 HTTP(S) 地址获取 `AGENTS.md`，以只读、可回滚的版本保存到本机，并为 Codex、Claude 和 Antigravity 创建统一入口。

主命令为 `agentsync`。v3.2.1 起新安装只创建该命令入口，重复安装会清理本工具管理的 `aic` 和 `ai-instructions` 旧链接。

核心 CLI 完全使用 Go 实现。`install.sh` 只负责首次下载和校验 Release 二进制；生成的 Shell 集成只负责在启动 AI 工具前同步规则，不包含另一套 CLI 实现。

## 功能

- 从一个远程 `AGENTS.md` 同步多种 AI 工具的共享规则。
- 使用内容版本目录和原子符号链接切换，避免读取到半写入状态。
- 网络失败时保留最后一次成功部署的有效规则。
- 切换规则来源前完成下载和校验，失败时保持现有配置不变。
- 安装前检查路径冲突，不覆盖普通文件或不受管配置。
- 使用 Release 二进制和 SHA-256 checksum 安装、升级。
- 提供 `status` 和 `doctor` 检查来源、runtime、工具入口及 Shell 集成。
- 卸载前展示固定计划，只删除本工具明确管理的对象。

## 支持平台

- macOS：Apple Silicon、Intel
- Linux：arm64、x86_64
- WSL 2：使用 Linux x86_64 产物

正式 Release 以 `agentsync_*` 为主产物。v3.2.1 仍额外提供内容相同的 `aic_*` 升级兼容产物，使 v3.1.2 客户端能够完成升级：

- `agentsync_Darwin_arm64`、`aic_Darwin_arm64`
- `agentsync_Darwin_x86_64`、`aic_Darwin_x86_64`
- `agentsync_Linux_arm64`、`aic_Linux_arm64`
- `agentsync_Linux_x86_64`、`aic_Linux_x86_64`
- `checksums.txt`

## 安装

准备一个能够直接下载的 `AGENTS.md` HTTPS 地址，然后执行：

```sh
curl -fsSL https://raw.githubusercontent.com/RokiLai/agent_sync_tool/main/install.sh \
  | sh -s -- https://example.org/path/to/AGENTS.md
```

请把示例地址替换为你自己的规则文件地址。默认会：

- 自动识别 Zsh 或 Bash；
- 为 Codex、Claude、Antigravity 创建规则入口；
- 仅将 `agentsync` 放入 `~/.local/bin`；
- 立即下载并部署第一份规则。

如果传入 GitHub 文件页面，例如：

```text
https://github.com/OWNER/REPOSITORY/blob/main/AGENTS.md
```

工具会自动转换为 `raw.githubusercontent.com` 原始文件地址并保存。为避免猜错复杂分支名，自动转换仅支持 `main`、`master` 或完整 40 位提交 SHA；其他分支请直接提供 raw URL。返回 HTML 的下载地址会被拒绝。

只配置 Codex 且不修改 Shell 启动文件：

```sh
curl -fsSL https://raw.githubusercontent.com/RokiLai/agent_sync_tool/main/install.sh \
  | sh -s -- https://example.org/path/to/AGENTS.md \
      --tools codex --shell none
```

固定安装指定版本：

```sh
curl -fsSL https://raw.githubusercontent.com/RokiLai/agent_sync_tool/main/install.sh \
  | AIC_VERSION=vX.Y.Z sh -s -- https://example.org/path/to/AGENTS.md
```

将 `vX.Y.Z` 替换为 Releases 中存在的版本标签。

完整安装和恢复说明见 [安装、升级与恢复](docs/install-and-recovery.md)。

## 快速开始

```sh
agentsync version
agentsync status
agentsync doctor
agentsync sync
```

查看当前规则来源：

```sh
agentsync source
```

验证另一个来源，但不修改配置：

```sh
agentsync source test https://example.org/new/AGENTS.md
```

交互式切换来源：

```sh
agentsync source set https://example.org/new/AGENTS.md
```

升级到最新正式版本：

```sh
agentsync upgrade
```

升级会先显示当前版本和最新正式版本。真实终端发现新版本时会请求确认，确认后以动态进度条展示下载、校验和原子安装进度；当前已是最新版本时不会下载完整二进制。输出被重定向或运行在 CI 中时，进度会自动降级为稳定的逐行日志并保持原有的自动升级行为。

## 工作方式

规则内容按 Git blob SHA-1 生成内容版本，发布到 runtime 的 `versions/<revision>`。`current` 通过原子符号链接切换到有效版本，兼容入口始终指向 `current/AGENTS.md`。

Shell 集成会在启动 Codex、Claude 或 Antigravity 前执行一次 `agentsync sync`。同步失败且没有有效缓存时，不会继续启动对应工具；已有有效缓存时会显示警告并继续使用最后一次成功版本。

详细设计见 [架构与数据安全](docs/architecture.md)。

## 默认目录

```text
~/.config/ai-instructions/              配置、已安装二进制和 Shell 集成
~/.local/share/ai-instructions-runtime/ 规则版本与当前版本链接
~/.local/bin/                            agentsync 命令入口
~/.codex/AGENTS.md                       Codex 规则入口
~/.claude/CLAUDE.md                      Claude 规则入口
~/.gemini/GEMINI.md                      Antigravity 规则入口
```

所有目录都可以通过环境变量调整，详见 [命令参考](docs/command-reference.md)。

## 开发

项目使用 Go 标准库，不依赖第三方 Go 模块。

```sh
make verify
```

`make verify` 会检查格式、运行 `go vet` 和全量 race 测试，并交叉构建 macOS、Linux 的 amd64 与 arm64 产物；GitHub Actions 使用相同的纯 Go 质量门禁。

构建当前平台二进制：

```sh
go build -o agentsync ./cmd/agentsync
```

构建并校验全部 Release 产物：

```sh
sh scripts/build-release.sh dist
sh scripts/verify-release.sh dist
```

## 更多文档

- [安装、升级与恢复](docs/install-and-recovery.md)
- [命令参考](docs/command-reference.md)
- [架构与数据安全](docs/architecture.md)
- [Releases](https://github.com/RokiLai/agent_sync_tool/releases)
