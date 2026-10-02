# Architecture

一句话：**一个 Go 单文件程序 + 一份平台定义文件 + 一份权威库，让多个 Agent 看到同一份记忆与技能。**

## 1. 全局视图

```mermaid
flowchart TB
    subgraph host["本机 · 不出网"]
        direction TB
        AGENT["fenjue-agent<br/>Go 单文件"]
        CFG[("platforms.json<br/>平台定义 · 运行期读取")]
        ROOTS[("权威库<br/>memory 根 / skills 根")]
        SEED[("种子包 pack<br/>go:embed + manifest")]
        TRASH[("备份区<br/>~/.fenjue/trash")]
    end

    subgraph agents["各 Agent 程序目录"]
        WB["wb · WorkBuddy"]
        TR["tr · TRAE SOLO CN"]
        CX["cx · Codex CLI"]
        HM["hm · Hermes Agent"]
        ZC["zc · ZCode"]
        OC["oc · OpenCode"]
        QD["qd · Qoder CN"]
        MM["mm · MiniMax Code beta"]
        DS["ds · DeepSeek Harness beta"]
    end

    AGENT -->|"读"| CFG
    AGENT -->|"播种 / 校验"| SEED
    AGENT -->|"写前备份"| TRASH
    AGENT -->|"建链接 / 镜像"| ROOTS
    ROOTS -.->|"目录链接 / 镜像 / 逐技能链接"| WB
    ROOTS -.-> TR
    ROOTS -.-> CX
    ROOTS -.-> HM
    ROOTS -.-> ZC
    ROOTS -.-> OC
    ROOTS -.-> QD
    ROOTS -.-> MM
    ROOTS -.-> DS
```

实线是程序行为，虚线是文件系统层面的链接关系。**权威库只有一份**，各 Agent 目录里的是链接，因此任何一端改内容，其余端立即可见。

## 2. 模块职责

| 模块 | 职责 | 关键约束 |
|---|---|---|
| `internal/platform` | 读 `platforms.json`，展开路径变量 | 路径只允许 `~` / `%VAR%` / `$VAR`，禁个人绝对路径 |
| `internal/seed` | 首次播种 + **完整性校验** | 零覆盖：非空库一律跳过；校验失败 fail-closed |
| `internal/mount` | 三种挂载形态的具体实现 | 原子操作：临时目录建好再 rename，失败回滚 |
| `internal/inject` | 在 Agent 的入口文件写入标记 | 幂等；软关闭只停注入、保留链接 |
| `internal/safeio` | 备份、回收、路径校验、Windows ACL | 任何写操作前先备份 |
| `internal/server` | 回环 HTTP 服务 + 四道安全闸 | 只绑 `127.0.0.1`；除健康检查外都要令牌 |
| `internal/state` | 状态核验（verify） | **测不到就报 UNVERIFIED，不折算为通过** |
| `cmd/fenjue-agent` | CLI 入口 + 前端内嵌 | 前端必须先构建 |

## 3. 挂载的四种形态

| 形态 | 用在哪 | 关闭的含义 |
|---|---|---|
| 目录链接 | 默认 | 断开链接（软关闭时保留链接、只停注入） |
| 物理镜像 | `cx` | **停止同步**，已复制过去的文件不会自动删除 |
| 逐技能链接 | `hm`，约 160 条 | 批量摘除，批量恢复 |
| 不写穿 | 任何根本身已是链接时 | 完全不碰 |

## 4. 首次启用一条平台时发生什么

```mermaid
sequenceDiagram
    participant U as 用户（控制台/CLI）
    participant A as agent
    participant V as seed.VerifyPack
    participant S as 权威库

    U->>A: enable hm
    A->>A: 解析 platforms.json + 展开路径
    A->>V: 校验内嵌种子包
    alt 校验失败（哈希不符 / 清单缺失）
        V-->>A: 报错
        A-->>U: 拒绝，且不写任何文件
    else 校验通过
        V-->>A: 通过
        A->>S: 根为空则播种；非空则零覆盖跳过
        A->>A: 写前备份 → 建链接 → 写入口标记
        A-->>U: 返回备份号（可一键还原）
    end
```

**为什么校验在播种之前**：校验失败时若已经写了一部分文件，就留下半成品，用户还得手工清理。先校验后写入，失败面就是零。

## 5. 令牌在请求链路上的位置

```mermaid
flowchart LR
    B["浏览器 / 页面"] -->|"Host 头"| G1{"闸1 Host 校验"}
    G1 -->|"Origin 在白名单"| G2{"闸2 来源白名单"}
    G2 -->|"OPTIONS 正确应答"| G3{"闸3 CORS"}
    G3 -->|"URL 片段带令牌"| G4{"闸4 令牌（恒定时间比较）"}
    G4 -->|"|"| API["业务接口"]
    G1 -.->|拒绝| X["403"]
    G2 -.->|拒绝| X
    G4 -.->|拒绝| X
```

令牌通过 URL 片段携带：片段不会随 HTTP 请求发给服务器，也不会进入 Referer，因此从浏览器地址栏取到它的那次导航不会把它泄露给服务器日志。

## 6. 三条不变量

代码里到处在守的东西，写在这里方便对照：

1. **零覆盖**：已存在且非空的库，任何情况下都不被播种内容改写。
2. **可逆**：写前必备份，响应里回传备份号；软关闭不删链接。
3. **测不到就说测不到**：`state` 与 `parity` 都有 UNVERIFIED 态，且明确「UNVERIFIED 不是通过」。

## 7. 延伸阅读

- [FAQ](FAQ.md) —— 常见问题
- [安全策略](SECURITY.md) —— 四道闸与已知限制
- [贡献指南](CONTRIBUTING.md) —— 改种子包或加平台的规约
