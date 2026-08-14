# 安装、升级与恢复

Agent Sync Tool 通过 GitHub Release 提供单文件二进制。安装脚本会识别系统和架构，下载对应产物及 `checksums.txt`，通过 SHA-256 校验后才执行安装。

## 安装要求

- macOS、Linux 或 WSL 2
- `curl`
- `shasum` 或 `sha256sum`
- 一个能够直接下载的单行 HTTP(S) `AGENTS.md` 地址

## 标准安装

```sh
curl -fsSL https://raw.githubusercontent.com/RokiLai/agent_sync_tool/main/install.sh \
  | sh -s -- https://example.org/path/to/AGENTS.md
```

示例 URL 必须替换为实际规则文件地址。重定向后的最终地址也必须使用 HTTP(S)。

GitHub 文件页可以直接作为输入：

```sh
curl -fsSL https://raw.githubusercontent.com/RokiLai/agent_sync_tool/main/install.sh \
  | sh -s -- https://github.com/OWNER/REPOSITORY/blob/main/AGENTS.md
```

工具会显示提示，将明确的 `blob/main`、`blob/master` 或 `blob/<40位提交SHA>` 地址转换为 `raw.githubusercontent.com`，并把转换后的地址保存为规则来源。其他分支名可能包含 `/`，工具不会猜测其边界，请从 GitHub 的 Raw 按钮复制原始文件地址。

下载响应为 `text/html` 时安装会失败，避免把网页内容当作 AI 规则部署。

默认选项相当于：

```text
--shell auto
--tools codex,claude,agy
```

`--shell auto` 根据当前 `$SHELL` 选择 Zsh 或 Bash；无法识别时不修改 Shell 配置。

## 自定义工具和 Shell

只为 Codex 创建入口：

```sh
curl -fsSL https://raw.githubusercontent.com/RokiLai/agent_sync_tool/main/install.sh \
  | sh -s -- https://example.org/path/to/AGENTS.md \
      --tools codex
```

同时配置 Codex 和 Claude：

```sh
curl -fsSL https://raw.githubusercontent.com/RokiLai/agent_sync_tool/main/install.sh \
  | sh -s -- https://example.org/path/to/AGENTS.md \
      --tools codex,claude
```

不修改 `.zshrc` 或 `.bashrc`：

```sh
curl -fsSL https://raw.githubusercontent.com/RokiLai/agent_sync_tool/main/install.sh \
  | sh -s -- https://example.org/path/to/AGENTS.md \
      --shell none
```

明确选择 Shell：

```sh
# Zsh
... | sh -s -- https://example.org/path/to/AGENTS.md --shell zsh

# Bash
... | sh -s -- https://example.org/path/to/AGENTS.md --shell bash
```

## 固定版本安装

默认安装最新正式版本。固定到指定版本：

```sh
curl -fsSL https://raw.githubusercontent.com/RokiLai/agent_sync_tool/main/install.sh \
  | AIC_VERSION=vX.Y.Z sh -s -- https://example.org/path/to/AGENTS.md
```

将 `vX.Y.Z` 替换为 Releases 中存在的版本标签。

私有镜像或测试环境可以覆盖 Release 根地址：

```sh
AIC_RELEASE_BASE_URL=https://downloads.example.org/agent-sync/releases \
AIC_VERSION=vX.Y.Z \
sh install.sh https://example.org/path/to/AGENTS.md
```

## 安装前预检

安装会先完成规则下载、内容校验和全部目标路径检查。以下情况会拒绝安装且不写入任何目标：

- 命令入口位置存在普通文件；
- AI 工具规则入口已指向其他目标；
- 配置文件没有本工具的受管标识；
- Shell 配置包含不完整的受管块；
- runtime 布局不完整或包含非预期链接。

可以先查看计划：

```sh
agentsync install https://example.org/path/to/AGENTS.md --dry-run
```

## 安装后检查

```sh
agentsync version
agentsync status
agentsync doctor
```

`status` 展示安装模式、规则来源、runtime 版本和工具入口。`doctor` 进一步检查依赖、内容版本一致性、命令入口及 Shell 配置，并在存在关键失败时返回非零状态。

## 同步与更换来源

手动同步：

```sh
agentsync sync
```

查看或测试来源：

```sh
agentsync source
agentsync source test
agentsync source test https://example.org/other/AGENTS.md
```

交互式更换来源：

```sh
agentsync source set https://example.org/other/AGENTS.md
```

新来源会在询问确认前完成下载和校验。取消、下载失败或发布失败时，原来源和当前 runtime 保持不变。

## 升级

升级到最新正式版本：

```sh
agentsync upgrade
```

固定升级目标：

```sh
AIC_VERSION=vX.Y.Z agentsync upgrade
```

升级流程会：

1. 查询 checksum 清单和目标 Release 版本；
2. 展示当前版本与目标版本，已是最新版时直接结束；
3. 在真实终端中询问是否继续；
4. 动态展示当前平台候选二进制的下载进度；
5. 验证 SHA-256，并核对候选版本与目标 Release；
6. 在安装目录内原子替换工具本体。

非交互环境不会等待输入，继续保持自动升级行为；进度输出会降级为适合 CI 和日志文件的普通文本行。

下载、checksum、候选执行或替换失败时，当前工具保持不变。

### 从 v3.1.2 迁移命令名

v3.1.2 只有 `aic` 入口，可先用旧命令升级，再补建新入口：

```sh
aic upgrade
aic install
agentsync version
```

第二步会复用已保存的规则来源、创建 `agentsync`，并清理本工具管理的 `aic` 和 `ai-instructions` 旧链接。v3.2.1 的 Release 仍保留 `aic_*` 资产，仅用于让旧客户端完成这次迁移。

## 故障恢复

### 网络不可用

普通 `agentsync sync` 下载失败时，如果 runtime 中已有有效规则，会警告并继续使用最后一次成功版本。首次安装或更换来源不会用缓存冒充成功。

### 检查当前状态

```sh
agentsync status
agentsync doctor
```

runtime 当前内容可以直接读取：

```sh
cat ~/.local/share/agentsync-runtime/AGENTS.md
cat ~/.local/share/agentsync-runtime/REVISION
```

### 重新安装

对同一 URL 重复执行安装是幂等操作，可用于恢复缺失的受管入口：

```sh
curl -fsSL https://raw.githubusercontent.com/RokiLai/agent_sync_tool/main/install.sh \
  | sh -s -- https://example.org/path/to/AGENTS.md
```

重新安装不会覆盖不受管文件；遇到冲突时应先检查并人工决定如何处理该路径。

## 卸载

```sh
agentsync uninstall
```

卸载必须在交互式终端运行。工具会先展示固定计划，第一次确认是否删除受管命令、配置、Shell 块和 AI 工具入口，第二次确认是否同时删除规则 runtime。所有确认默认都是“否”。

未选择清理 runtime 时，可以稍后检查或手动备份已部署规则。
