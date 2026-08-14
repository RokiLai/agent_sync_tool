# 架构与数据安全

Agent Sync Tool 将远程规则、版本存储、AI 工具入口和命令行生命周期分开管理。核心目标是在同步、升级或进程中断时，始终保留一份完整可读的规则。

## 组件

```text
cmd/agentsync
  └─ internal/app          命令解析、输出和退出码
      ├─ source            URL 校验与 HTTP 下载
      ├─ runtime           同步、内容版本和 last-known-good
      ├─ install           安装计划、预检和执行
      ├─ integration       AI 工具入口与 Shell 集成
      ├─ diagnose          status 和 doctor
      ├─ upgrade           Release 升级
      ├─ terminalprogress  TTY 动态进度与非交互日志
      └─ uninstall         卸载计划和执行

internal/core              稳定的共享兼容原语
internal/identity          主命令、兼容入口和 Release 产物名称
internal/config            路径和受管配置
internal/managedfs         原子写入和符号链接
internal/lock              跨进程同步锁
```

业务模块通过应用层编排或基础模块共享稳定能力。`terminalprogress` 是终端展示适配层，只消费升级进度事件，不负责下载、校验或安装决策。

## 配置布局

默认配置目录：

```text
~/.config/agentsync/
├── agents-url
├── repo-path
├── bin/
│   └── agentsync
└── shell-integration.sh
```

`agents-url` 保存规则来源。配置文件使用固定受管标识，读取时要求目标是普通文件且标识有效，避免把符号链接或任意文件当作可信配置。

## Runtime 布局

```text
~/.local/share/agentsync-runtime/
├── versions/
│   └── <revision>/
│       ├── AGENTS.md
│       └── REVISION
├── current -> versions/<revision>
├── AGENTS.md -> current/AGENTS.md
└── REVISION -> current/REVISION
```

版本目录中的文件权限为 `0444`。内容版本使用 Git blob SHA-1：

```text
SHA1("blob " + 内容字节数 + NUL + 原始内容)
```

它用于稳定标识规则内容，不承担 Release 下载的完整性校验；发布二进制使用 SHA-256 checksum。

## 原子同步

同步顺序：

```text
读取并验证来源
→ 获取 runtime 锁
→ 下载到目标文件系统内的临时文件
→ 检查非空内容
→ 计算 revision
→ 创建只读候选版本
→ 校验或发布版本目录
→ 原子替换 current
→ 清理候选状态并释放锁
```

同一个 revision 已存在时，工具会比较实际内容；发现不一致会失败，不会复用冲突目录。

## Last-known-good

普通同步下载失败时，工具检查当前 runtime：

- 当前版本有效：返回缓存状态并输出警告；
- 没有有效版本：同步失败并返回非零状态。

首次安装和来源切换属于严格操作，必须成功获得并发布新内容，不能用旧缓存替代。

## 来源安全

- 来源必须是单行 HTTP(S) URL；
- 每次重定向后的 URL 仍必须是 HTTP(S)；
- 非 2xx 响应和空内容会失败；
- 来源配置只作为数据读取，不会作为 Shell 代码执行；
- 来源切换在确认前完成候选下载与校验；
- 发布失败时恢复旧来源和旧 runtime。

## 锁和中断

同步锁位于 `${runtimeDir}.lock/pid`。工具识别活跃所有者和陈旧锁，等待次数达到上限后返回失败。收到中断信号或操作失败时，会清理由当前进程创建的锁与候选文件。

## 受管文件边界

安装和卸载只操作满足以下条件的对象：

- 使用本工具固定 marker 的配置文件；
- 指向精确预期目标的符号链接；
- 本工具生成的 runtime 结构；
- 位于明确 begin/end marker 之间的 Shell 配置块。

普通文件、目标不符的链接、marker 缺失的配置和不完整 Shell 块会被保留并报告冲突或警告。

## Release 安全

安装和升级针对当前系统选择固定资产名称，下载 `checksums.txt` 后验证 SHA-256。候选二进制通过 `version` 自检后，升级才会在同一目录内使用原子重命名替换当前工具。

v3.2.1 的主资产使用 `agentsync_<OS>_<ARCH>`；同一二进制仍以 `aic_<OS>_<ARCH>` 发布一次，仅使只认识旧资产名的 v3.1.2 能升级。新安装只创建 `agentsync` 入口；配置目录、环境变量和内部受管二进制名称保持不变。

升级分为两个阶段：先从 Release 重定向和 checksum 清单识别目标版本，展示当前版本与目标版本并在交互终端请求确认；确认后才流式下载候选二进制。下载过程通过进度事件交给终端展示层，TTY 使用动态单行进度，非交互环境输出稳定日志。候选版本必须与目标 Release 一致，任何下载、checksum、版本或替换失败都不会修改当前工具。
