# agent_sync_tool

`ai-instructions` 的 Go 兼容性重构项目。源 Shell 项目保持独立，默认位于相邻目录 `../ai-instructions`。

当前已完成 Go 迁移阶段 6，提供Release二进制安装、checksum验证和原子升级，并保持Shell/Go双实现兼容验证。现有36个Shell黑盒场景、稳定帮助与版本输出、受管marker和Git blob SHA-1固定向量记录在`test/contract`。

双实现兼容验证与回退说明见 `docs/migration-compatibility.md`。

本地验证：

```sh
sh scripts/verify-shell-contracts.sh
go test -race ./...
```

也可以通过 `AI_INSTRUCTIONS_SOURCE_PROJECT` 指定只读源项目路径。
