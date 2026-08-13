# Shell 与 Go 迁移兼容性

迁移期保留 Shell 源实现，并将 Go 实现作为候选。两者直接共享以下用户数据，不需要格式迁移：

- `agents-url` 与 `repo-path` 受管配置；
- `versions/<git-blob-sha1>` runtime；
- `current`、`AGENTS.md` 和 `REVISION` 符号链接；
- Codex、Claude 与 Antigravity 规则入口。

本阶段不提供公开的 `AIC_IMPLEMENTATION` 环境变量。实现选择只存在于测试脚本：

```sh
scripts/run-implementation-contracts.sh shell
scripts/run-implementation-contracts.sh go
scripts/run-implementation-contracts.sh all
```

## 回退

Go 候选产生的配置与 runtime 可由 Shell 实现直接读取。回退只需恢复 Shell 工具本体，不应删除或重写用户 runtime。

## 平台范围

CI 在 macOS 与 Linux 运行测试，并交叉编译 Darwin/Linux 的 amd64、arm64目标。WSL amd64仍需Windows WSL或自托管运行器完成实机验证。
