# ADR-001：Agent 写权限契约

- **状态**：已接受（2026-10-14）
- **取代**：ROADMAP `G29` 的「令牌如何交给 Agent / Agent 可写哪些路径」两个未决项
- **不做**：MCP server、token 作用域、token 体系改动（理由见 §6）

## 1. 背景

`G24`（`export` 只读出口）落地时刻意把安全边界收窄到只读，理由写得很明确：

> 不联网、不监听端口、不需要令牌：直接读库，不经过 HTTP，因此不存在
> 「令牌怎么交给 Agent」这个还没定论的问题（那是 G24 的写入口）。

所以从那时起，「Agent 要能写」这件事是**明确挂着的**，`G29` 记的就是它。

## 2. 先纠正 G29 的前提

`G29` 原文把这件事写成「令牌如何交给 Agent、Agent 可写哪些路径」。本轮实读源码后，两点需要更正：

**其一，写能力早就有，而且不需要令牌。** CLI 的 `enable` / `disable` / `preset` / `uninstall`
走的是**本地用户权限**——本轮在沙箱里跑 `enable wb`，`rc=0`、全程未要求任何令牌。
这不是设计出来的，是实现的副产物，且从未被写成契约。

**其二，真正缺的不是「令牌怎么给」，而是「路径能写到哪」。** `middleware.go` 的 HTTP 鉴权是
**单一扁平 token**（`X-Fenjue-Token`，`subtle.ConstantTimeCompare` 常量时间比较），
持有者可以调 `/api/roots` 改库根、可以调 `/api/platforms/*` 增删同步。**全仓不存在路径作用域**
（grep `allowlist|whitelist|白名单|scope` 只命中浏览器 Origin，不是路径）。

而 `POST /api/roots` 原本的校验只有两条：

```go
if filepath.Clean(exp) == "" || strings.ContainsAny(filepath.Clean(exp), "<>") {
```

也就是说：**库根可以指向任意路径**，随后 `enable` 会在那里执行 `seed.Bootstrap(cfg.Roots)` 写入。
写入口一旦交给 Agent，这等于「以本机权限任意写盘」。

## 3. 决定

**采「CLI 写 = 本地用户权限、免令牌」为正式契约**，并补两件事：路径策略 + 审计。

### 3.1 授权模型（确认现状，不改）

| 面 | 授权 | 本轮是否改 |
|---|---|---|
| CLI `enable/disable/preset/uninstall/export` | 本地用户权限，**不需要令牌** | 否（只是把现状正式写成契约） |
| HTTP `/api/*` | 单一扁平 token，`/api/health` 与静态页豁免 | 否（不动 token 体系） |

**这是本 ADR 的核心取舍。** 承认「CLI 写不需要令牌」，意味着**任何能在本机跑进程的 Agent
都天然拥有完整写能力**。我们不试图用令牌去缩小这个面——那既做不到（同一用户照样能直接调 CLI），
也会制造「HTTP 面看起来更安全」的错觉。真正的边界是**路径策略**。

### 3.2 路径策略

新增 `safeio.RootPolicy`（`agent/internal/safeio/rootpolicy.go`）：

- 默认允许集 = **用户主目录 ∪ `~/.fenjue`**，即「只许写用户家目录之内」。
- 判定是**前缀包含**，且必须防前缀混淆：`C:\Users\<you>` **不得**放行 `C:\Users\<you>-evil`。
  实现用「相等 或 以 prefix + 分隔符 开头」，不用裸 `strings.HasPrefix`。
- Windows 下比较**大小写不敏感**。
- 判定前先 `Clean`，否则 `…/foo/../bar` 能绕过字面比较。
- 放宽必须**显式**：`--allow-root <prefix>`（可重复）。默认不做任何猜测。
- 拒绝时的错误信息**必须点名 `--allow-root`**，否则排查者只看到「invalid root」而不知如何合法化。

> **与拍板口径的一处偏离，需要说明。** 选项里写的是「路径白名单 = 只允许写统一库根」，
> 但**按字面执行会直接打死 `enable`**——实测 `state.Enable` 写四处：
> `os.MkdirAll(p.Home)`、`seed.Bootstrap(cfg.Roots)`、`mount.MirrorSync/PerSkillEnable(m.To)`、
> `inject.Apply(inj.Path)`。后三者落在**平台自己的目录**里，正是功能本身。
> 故实际策略是「**库根覆盖**必须在允许集内」，平台路径仍由 `platforms.json` 声明、
> 不接受请求参数直接指定。这是对「更严格」原则的让步，让步的理由是功能不成立。

### 3.3 审计

新增 `logx.Audit(op, target, ok, detail)`，输出到 `~/.fenjue/logs/agent.log`，
固定前缀 `[AUDIT]`，固定字段序 `op / target / ok / detail`。

- 排查命令：`grep "\[AUDIT\]" ~/.fenjue/logs/agent.log`
- 取 `INFO` 级别（只进日志、不上 stderr）：审计是**记录**不是**告警**；真有失败时调用方已收到非 0 返回码。
- 接入点：`state.Enable` / `state.Disable`（用**具名返回值 + defer** 覆盖每一个返回点，
  包括 `nothing to disable` 这类早退）、`POST /api/roots`（成功与拒绝两路都落）。
- **值必须转义方括号。** 否则 detail 里嵌一段 `[AUDIT] op=fake …` 就能伪造出第二条审计行，
  而排查命令恰好是 `grep "\[AUDIT\]"`。这条不是理论风险，是本轮反例腿实测逮到的：
  第一版 `sanitize` 只压平换行，测试立刻报「exactly one marker, got 2」。
- 空值统一为 `-`，避免 grep 时看到 `target=` 后面空着、像漏了字段。

## 4. 判据

| 项 | 命令 | 通过条件 |
|---|---|---|
| 路径策略 | `go test -count=1 ./internal/safeio/` | 7 条全绿，**含前缀混淆反例腿** |
| 审计格式 | `go test -count=1 ./internal/logx/` | 4 条全绿，**含伪造标记反例腿** |
| 全量回归 | `go test -count=1 ./...` | 全绿；`go build ./...` 与 `go vet ./...` rc=0 |
| 端到端 | 见 `docs/ROADMAP.md` §5 的 G29 行 | 家目录外库根被拒（400）+ 家目录内被收 + 审计行可 grep |

**必须带 `-count=1`**：不带时命中 `(cached)`，那是缓存不是本轮验证——本轮已在 G15 腿上踩过一次。

## 5. 未覆盖项（明写，不假装没有）

- **只拦写入，不拦读取。** `loadRootsOverride` 仍会照读历史上已写入策略外的 `roots.json`。
  改读取侧会在有人故意把库放在 `D:\` 时**静默回落**到默认库根——那是更坏的行为。
  故本轮明确不动，登记为未覆盖。
- **`inject.Apply` 的目标**来自 `platforms.json`，本轮未加「目标必须在平台 home 之内」的判定。
- **CLI 侧无路径策略。** `enable` 的平台路径不受 `RootPolicy` 约束（它约束的是库根覆盖）。
  若 `platforms.json` 本身被篡改，策略挡不住——那需要给 `platforms.json` 加签名或校验和，属另一条 ADR。

## 6. 为什么不做 MCP / token 作用域

- **MCP**：会把「传输在哪跑」「进程怎么管」「授权怎么给」三个新问题一次引入，
  而现有 CLI 已经具备全部写能力。包一层 MCP 得到的是生态兼容性，不是新能力。
- **token 作用域**：安全模型更正，但需改中间件、令牌存储与轮换。
  且按 §3.1 的论证，**收窄 HTTP 面并不收窄实际攻击面**（同一用户可直接调 CLI），
  收益与成本不匹配。
- **解锁条件**：出现「需要在**不共享用户身份**的上下文中驱动写入」的真实诉求。
  在那之前，加作用域只是让 HTTP 面看起来更安全。

## 7. 相关

- `docs/ROADMAP.md` §1 `G29` / §5 `G29`
- `docs/ROADMAP.md` §5 段首硬前置（判据落笔前必须实跑）
- `agent/internal/safeio/path.go`（既有路径工具，本 ADR 是它的约束面）
- `agent/internal/logx/logx.go`（既有分级日志，审计是它的一个正交出口）