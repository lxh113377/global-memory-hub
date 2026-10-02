# G8 沙箱协议（mm / ds 由 beta 转 ga 的取证规范）

> 目的：把「能不能转 ga」这件事从口头判断变成**可复现的机器取证**。
> 本文件定义环境要求与输入输出格式；资产在 `scripts/g8_sandbox/`。

## 0. 结论先说（2026-10-02 实测）

| 验证层 | mm | ds | 判据 |
|---|---|---|---|
| 程序侧装载行为 | **PASS** | **PASS** | 沙箱内 enable/verify/link/zero-overwrite 全绿 |
| 目标程序消费行为 | **UNVERIFIED** | **UNVERIFIED** | 沙箱内无 headless 客户端 |

**因此 mm/ds 仍为 beta。** 本轮消掉的是「程序侧装载行为未证」这一条阻塞理由，
剩下的「消费行为未证」需要一台装有对应桌面客户端的机器。

为什么消费行为在容器里测不了：两个程序都是**桌面应用**。实测 `~/.minimax/bin` 下只有
`mavis-trash`（垃圾清理工具），没有主 CLI；npm 上无对应包。容器内无 GUI，
即使装上也无法发起真实会话。

## 1. 环境配置要求

| 项 | 要求 | 理由 |
|---|---|---|
| Docker | daemon 可用，Server 29.x | 已实测 29.8.0 |
| 基础镜像 | `node:22-alpine`（或任意 alpine） | 底层即 alpine；产物 `CGO_ENABLED=0` 静态链接，alpine 可直接跑 |
| 程序来源 | Release 的 `fenjue-agent-linux-amd64` | 与对外发布物同一份二进制，避免「测的不是发的」 |
| 宿主挂载 | **禁止任何路径挂载** | 隔离是本方案的全部意义：作者机上跑产品路径会把真实库根改指 |
| 容器内根 | `/g8/roots/{memory,skills}` | 沙箱内自有库根 |
| 容器内 home | `/g8/home/.minimax` / `/g8/home/.dsh` | 与宿主同名目录不冲突 |

不复用宿主的 `platforms.json`：容器实例单独维护，避免把宿主 `site.*`、真实域名带进沙箱。

## 2. 输入格式规范

输入只有一个文件：`scripts/g8_sandbox/platforms.g8.json`。

| 字段 | 约束 |
|---|---|
| `schema` | 必须 `fenjue-platforms-v1`（版本一致性判据会校验它与 `version` 配套） |
| `version` | 语义化版本，主版本号须与 `schema` 后缀 `-vN` 一致 |
| `roots` | **容器内绝对路径**，不得出现宿主路径或用户名 |
| `site.*` | `primary` 用 `https://example.invalid` 占位，不写真实域名 |
| `platforms[].home` | 容器内绝对路径 |
| `platforms[].os` | 仅 `["linux"]` |
| `platforms[].support` | 保持 `beta`（本协议不负责改档，见 §0） |
| `platforms[].notes` | 写明消费行为为何 UNVERIFIED |

只保留 `mm` 与 `ds` 两个平台：一次只验一件事，混入更多平台会让失败归因变难。

## 3. 输出格式规范

输出为 **NDJSON**（每行一个 JSON 对象），末尾附一行 `summary`。

每条记录字段：

| 字段 | 含义 |
|---|---|
| `step` | `enable` / `verify` / `link` / `zero-overwrite` / `consumption` / `unknown-platform` |
| `platform` | 该步所属平台（`link` 步为 `<platform>/<kind>`） |
| `rc` | 该步子命令的真实退出码 |
| `verdict` | `PASS` / `RED` / `UNVERIFIED` |
| `evidence` | 证据串，**必须与 `platform` 对应** |

`summary` 行给出 `rc`（整体）、`failed`、`unverified` 三个字段。

### 退出码三态

| 码 | 含义 |
|---|---|
| 0 | 全部必测步通过（`UNVERIFIED` 不在此列，单独记账） |
| 1 | 有必测步判红 |
| 2 | 步不可测（`UNVERIFIED`）|

**`UNVERIFIED` 绝不算进通过**：`consumption` 恒为 `UNVERIFIED` 但不拉低整体 rc，
因为它是**已知的能力边界**，不是待修的缺陷。把它算进失败会逼人伪造证据。

### 两条格式硬规

1. **evidence 必须与 platform 对应。** `verify` 一次打印所有平台，若直接截断前 N 字符，
   `ds` 的证据会变成 `mm` 的内容——看起来证据错配，实为截断。本协议要求先按平台
   `grep` 出该平台段落再截断。
2. **rc 必须是该子命令的真实退出码。** 不可在同一句里用 `$?` 展开，
   也不可经管道后再读（管道末端会洗白退出码）。

## 4. 必测步与反例腿

| 步 | 断言 |
|---|---|
| `enable` | 沙箱首启：建 home、建根、空库播种 |
| `verify` | 两个挂载点均 OK |
| `link` | `memory` / `skills` 都是**真符号链接**且指向沙箱根 |
| `zero-overwrite` | disable 后再 enable 必须命中零覆盖（非空库不被改写） |
| `consumption` | 恒 `UNVERIFIED`（见 §0） |
| `unknown-platform` | **反例腿**：未声明平台必须被拒绝 |

反例腿不是形式：没有它，「命令全部成功」与「命令真的在检查」无法区分。

## 5. 复现

```bash
cd scripts/g8_sandbox
# 先取与对外发布同一份二进制
curl -L -o fenjue-agent-linux-amd64 \
  https://github.com/lxh113377/global-memory-hub/releases/download/v0.2.2/fenjue-agent-linux-amd64
docker build -t fenjue-g8:1.0 .
docker run --rm fenjue-g8:1.0; echo "rc=$?"   # 退出码直读，不经管道
```

实测输出：`rc=0`，`summary.failed` 空，`summary.unverified` 为
`consumption:mm consumption:ds`。

## 6. 转 ga 还需要什么

一台装有 MiniMax Code / DeepSeek Harness Desktop 的机器（非作者机，避免改指真实库根），
在其上跑：`enable` → 真实会话 → 让程序读一次库 → 确认读到。

两者都通过后，才可以把 `platforms.json` 里对应条目的 `support` 改为 `ga`。
在那之前，`beta` 是诚实标注，不是保守。

---
## 相关

- [ROADMAP](ROADMAP.md) G8 条目（阻塞理由已按本协议更新）
- [架构说明](ARCHITECTURE.md)
