# Agent Sync Tool

[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-macOS%20%7C%20Linux%20%7C%20WSL2-lightgrey.svg)]()

> 集中管理、原子切换与无感同步 AI 编程助手（Codex / Claude / Antigravity）规则的命令行工具。

---

## ✨ 核心特性

- 🎯 **一处定义，多端同步**：从单一远程 `AGENTS.md` 自动分发规则至多种 AI 编程助手。
- ⚡ **版本管理与原子切换**：基于 Git Blob SHA-1 计算内容版本，通过原子符号链接切换，杜绝中间态。
- 🛡️ **安全回退与离线可用**：网络异常时自动使用最后一次有效缓存，切换来源失败不破坏当前配置。
- 🚀 **纯 Go 原生与无感集成**：零第三方外部 Go 依赖，结合 Shell 启动钩子实现终端启动时无感同步。

---

## 📦 快速安装

支持 **macOS**（Apple Silicon / Intel）、**Linux**（arm64 / x86_64）及 **WSL 2**。

准备好可公开访问的 `AGENTS.md` 规则地址后，执行以下命令即可完成安装：

```sh
curl -fsSL https://raw.githubusercontent.com/RokiLai/agent_sync_tool/main/install.sh \
  | sh -s -- https://example.org/path/to/AGENTS.md
```

> **💡 提示**：如果提供 GitHub blob 页面链接（如 `https://github.com/OWNER/REPO/blob/main/AGENTS.md`），工具会自动转换为对应的 raw 文件地址。

如需自定义目标工具、指定安装版本或进行无 Shell 集成安装，请参阅 [安装、升级与恢复指南](docs/install-and-recovery.md)。

---

## 🛠️ 常用命令速查

```sh
# === 状态与诊断 ===
agentsync status             # 查看当前配置来源与生效版本
agentsync doctor             # 诊断工具入口与 Shell 集成健康度
agentsync sync               # 手动触发一次规则同步

# === 规则来源管理 ===
agentsync source             # 查看当前规则 URL
agentsync source test <url>  # 测试并校验新来源（不修改配置）
agentsync source set <url>   # 切换到新的规则来源

# === 工具升级与维护 ===
agentsync upgrade            # 检查并一键安全升级到最新正式版本
agentsync version            # 查看当前 CLI 版本
```

完整命令参数与环境变量说明见 [命令参考手册](docs/command-reference.md)。

---

## 🗂️ 规则入口映射

安装后，`agentsync` 会在各 AI 助手的标准路径创建符号链接，统一指向当前生效的规则版本：

| AI 编程助手 | 规则入口路径 | 说明 |
| :--- | :--- | :--- |
| **Codex** | `~/.codex/AGENTS.md` | OpenAI Codex 规则入口 |
| **Claude** | `~/.claude/CLAUDE.md` | Anthropic Claude Code 规则入口 |
| **Antigravity** | `~/.gemini/GEMINI.md` | Google Antigravity 规则入口 |

默认配置与运行时数据分别存放在 `~/.config/agentsync/` 与 `~/.local/share/agentsync-runtime/`。

---

## ⚙️ 工作机制

1. **版本发布**：下载远程规则后按内容 Hash 发布至 `runtime/versions/<revision>`。
2. **原子切换**：通过原子替换 `runtime/current` 软链接完成新版本上线。
3. **无感拦截**：在 Shell 启动 AI 工具前自动触发增量同步检查，保证规则时刻最新。

详细架构设计与安全约束见 [架构与数据安全](docs/architecture.md)。

---

## 💻 本地开发

项目使用 Go 标准库实现，不依赖第三方 Go 依赖项。

```sh
# 运行完整质量门禁（代码格式、go vet、全量 race 测试与跨平台编译）
make verify

# 构建当前平台二进制
go build -o agentsync ./cmd/agentsync

# 打包并校验全平台 Release 产物
sh scripts/build-release.sh dist
sh scripts/verify-release.sh dist
```

---

## 📖 更多文档

- [安装、升级与恢复指南](docs/install-and-recovery.md)
- [CLI 命令与环境变量参考](docs/command-reference.md)
- [架构设计与数据安全](docs/architecture.md)
- [GitHub Releases 页面](https://github.com/RokiLai/agent_sync_tool/releases)

---

## 📄 许可证

本项目基于 [Apache License 2.0](LICENSE) 开源。
