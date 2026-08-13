# 安装、升级与恢复

安装脚本根据操作系统和架构下载GitHub Release二进制及`checksums.txt`，SHA-256验证成功后才执行安装。

```sh
curl -fsSL https://raw.githubusercontent.com/RokiLai/agent_sync_tool/main/install.sh | sh -s -- https://example.com/AGENTS.md
```

测试或镜像环境可设置`AIC_RELEASE_BASE_URL`；固定版本可设置`AIC_VERSION=vX.Y.Z`。

`aic upgrade`使用相同下载与校验流程，并在安装目录内原子替换。下载、checksum或候选执行失败时，当前工具保持不变。

回退到Shell实现时保留现有配置和runtime，恢复旧工具本体即可，无需迁移用户数据。
