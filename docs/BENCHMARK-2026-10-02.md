# 同类开源项目对标报告 · 2026-10-02

> 对标对象：本仓库（`global-memory-hub`，下称 **hub**）与 GitHub 上同种竞品 / 内容源 / 灵感源。
> 取证时间：2026-10-02。全部元数据经 GitHub API 认证态实取；星标为取数当日快照。
> 证据标记：`[实测]` = 本机一手实测；`[API]` = GitHub API 返回；`[UNVERIFIED]` = 取数失败，**不折算为 0，也不参与评分**。

---

## 0. 取证方法与可信度声明

| 通道 | 结果 | 处置 |
|---|---|---|
| `git` 本地 | hub HEAD `2cf6496` v0.2.2，工作树干净 | 采用 |
| `gh` CLI | **PATH 中 `gh` 解析到 0 字节占位文件，静默无输出** | 改用绝对路径 `C:\Program Files\GitHub CLI\gh.exe` 后全部取数成功 |
| `api.github.com` 匿名 | `403 rate limit exceeded` | 不使用匿名通道 |
| `api.github.com` 认证态 | 正常 | 采用 |

> 这条本身就是一条工程结论：**「命令静默无输出」与「取数成功但结果为空」在默认 shell 下不可区分**，任何自动化取证都必须先验证执行通道本身（命令存在 + 认证态 + 非空输出），否则会把通道故障误读成事实。

---

## 1. 对比总览表

### 1.1 同种竞品（做同一件事：让多个 Agent 共享技能/配置）

| 项目 | 星标 | 语言 | 覆盖端数 | 核心机制 | 与 hub 的关系 |
|---|---|---|---|---|---|
| **[vercel-labs/skills](https://github.com/vercel-labs/skills)** | 32,964 | TypeScript | **79 agents** | 多源 `add`（GitHub/GitLab/Azure/SSH/本地/URL）；symlink 与 `--copy` 双模；project / global / agent 三层作用域；`-y` 非交互；`skills use` 临时态不安装 | **最强正面竞品**，生态量级差 2 个数量级 |
| **[xingkongliang/skills-manager](https://github.com/xingkongliang/skills-manager)** | 5,366 | Rust | **50+ tools** | 命名 **Presets** 一键批量启停；Linked Workspaces（任意目录作根）；Project Workspaces 双向对比同步；marketplace（skills.sh）；备份 + 多设备同步；**让 Agent 驱动管理器**而非直接写目录 | **形态最接近的竞品**（一处管理、多端同步） |
| **[qufei1993/skills-hub](https://github.com/qufei1993/skills-hub)** | 1,717 | Rust | 多工具 | 跨平台桌面端，技能一处管理并同步到多工具全局目录 | 桌面应用形态对照 |
| **[luongnv89/asm](https://github.com/luongnv89/asm)** | 947 | TypeScript | 21 agents | 通用技能管理器；安全扫描；token 驻留会计 | 治理机制对照 |
| **[dyoshikawa/rulesync](https://github.com/dyoshikawa/rulesync)** | 1,488 | TypeScript | — | 规则编译/同步 CLI | 规则同步对照 |

### 1.2 内容源（技能从哪来）

| 项目 | 星标 | 语言 | 许可 |
|---|---|---|---|
| [anthropics/skills](https://github.com/anthropics/skills) | 179,387 | Python | 无 |
| [obra/superpowers](https://github.com/obra/superpowers) | 294,175 | Shell | MIT |

### 1.3 灵感源（记忆/上下文层，做的是 hub 的另一半）

| 项目 | 星标 | 语言 | 许可 |
|---|---|---|---|
| [mem0ai/mem0](https://github.com/mem0ai/mem0) | 66,463 | Python | Apache-2.0 |
| [topoteretes/cognee](https://github.com/topoteretes/cognee) | 31,301 | Python | Apache-2.0 |
| [supermemoryai/supermemory](https://github.com/supermemoryai/supermemory) | 31,062 | TypeScript | MIT |

### 1.4 三态取证台账（`[UNVERIFIED]` 不折算 0）

| 待定项 | 状态 | 依据 |
|---|---|---|
| `swiz`（merge+.bak+幂等+dry-run） | `[UNVERIFIED]` | 多次搜索无同名实体，疑为内部代号或已改名 |
| `CapSync` | `[UNVERIFIED]` | 多次搜索无同名实体 |
| `tepeumut/agentpull` | `[部分证实]` `[API]` | 仓存在但仅 ★1；其自述为「从任意 git host 拉取各家 agent 配置」，**任务卡所述「manifest+SHA-256+冲突三分类」在该仓未获证实** |

### 1.5 hub 基线 `[实测]`

| 维度 | 现状 |
|---|---|
| 星标 / fork | **1 / 0**（创建于 2026-10-01，仓龄 1 天） |
| 语言 / 许可 | Go / MIT |
| 覆盖平台 | **9 个**（`wb` `tr` `cx` `hm` `zc` `oc` `qd` 为 ga；`mm` `ds` 为 beta） |
| 挂载形态 | **3 种**：目录链接 / 物理镜像（`cx`）/ 逐技能链接（`hm`，约 160 条） |
| 管理根 | **双根：memory + skills** |
| 服务形态 | 回环 agent `127.0.0.1:7799` + 每次启动随机令牌 + 四道安全闸 + Windows ACL 加固 |
| 种子包 | memory 骨架 4 件 + 技能 **18 条**（全部过消毒门禁） |
| CI | 3 条 workflow：`ci`（vet + test + smoke，三平台矩阵；parity 为 advisory）/ `pages` / `release`（5 目标） |
| 文档 | `README.md`(192 行) + `README.en.md`(194 行) + INSTALL×4 + DEPLOY + ROADMAP + SKILLS-TIERS |
| 缺失文档 | **无**。⚠️ **2026-10-02 复核校正**：本行原写作「缺失 CONTRIBUTING / SECURITY / CHANGELOG / ARCHITECTURE / FAQ」，该口径取自初次盘点时点，五件已于同日落地（提交 `000a0b3`），与 §7.2 的对账表矛盾。**以本行为准**，逐项落地态见 §7.2 |

> **§1.5 的时点声明**：本节是报告取数时点（`2cf6496`）的 hub 基线快照。复核轮（`910cd0e`）实测该节有两处已过期——① 上表的缺失文档一行；② 文档清单未含后补的 `BENCHMARK` / `SKILLS-TIERS` / `G8-DOCKER-PROTOCOL`。其余项复核仍成立（星标 1、9 平台、双根、3 种挂载形态、回环 + 四道安全闸）。逐项复算命令见 §7.1。

---

## 2. 八维逐项对比与差距量化

### 维度 1 · 功能覆盖

| 能力 | hub | vercel-labs/skills | skills-manager | 差距 |
|---|---|---|---|---|
| 端覆盖 | 9 | 79 | 50+ | **落后 1 个数量级** |
| 记忆库共享 | **有（对标全无）** | 无 | 无 | **领先** |
| 技能库共享 | 有 | 有 | 有 | 持平 |
| 挂载形态粒度 | link / mirror / per-skill | symlink / copy | symlink / copy | **领先** |
| 命名预设批量启停 | 无 | 无 | **有（Presets）** | 落后 |
| 项目级工作区 | 无 | 有（project scope） | **有（双向对比同步）** | 落后 |
| 技能市场 / registry | 无 | 有（`skills add <repo>` 源生态） | 有（skills.sh） | 落后 |
| 非交互 / CI 友好 | 有（CLI 天然） | 有（`-y`/`--all`） | 无（GUI 为主） | 持平 |
| Agent 驱动的管理出口 | 无（仅 HTTP API） | 无 | **有** | 落后 |
| 多设备同步 | 无（**定位为不联网**） | 无 | 有 | 定位差异，非缺陷 |
| 可逆软关闭 + 全量回滚 | **有（备份含 manifest 变更清单）** | 无 | 无 | **领先** |

**结论**：hub 在「记忆 + 技能双根」「挂载形态粒度」「可逆与可回滚」三点上是**结构性领先**；在「端数量」「生态位（预设/市场/项目工作区）」「Agent 自主管理出口」三点上落后。

### 维度 2 · 技术架构

| 项 | hub | 对标 |
|---|---|---|
| 形态 | 单文件二进制 + 内嵌 React 控制台 | vercel-labs: Node CLI；skills-manager: Rust 桌面 |
| 前端 | React + Vite + Tailwind，4 页，中英 i18n | vercel-labs: 无 GUI；skills-manager: 桌面原生 |
| 后端 | Go `net/http`，7 个 API 端点 | — |
| 单一真相源 | `platforms.json`，**运行期派生**（非编译期生成） | 各自 registry |
| 跨平台一致性 | Windows 刻意用 junction（免提权），Unix 用 symlink | — |

**评价**：架构选择正确且有取舍自觉（Windows 免提权是有意为之并写进了文档）。**运行期派生**避免了编译期代码生成带来的漂移，这一点优于多数同类实现。

### 维度 3 · 实现方式

| 机制 | hub | 对标 |
|---|---|---|
| 完整性校验 | **无（种子包无 manifest）** | vercel-labs 有 lockfile；agentpull 路线有 SHA-256 manifest |
| 冲突处理 | 写前全量备份 + 零覆盖铁律 | swiz 路线为 merge+.bak（`[UNVERIFIED]`） |
| 幂等 | 有（`seed` 幂等，第二次必跳过） | — |
| 失败回滚 | 有（per-skill 并发 8 + 临时目录 rename + 失败回滚） | — |
| 版本一致性 | **无机器判据**（四处版本号手工同步，已发生过漂移） | — |
| 安全扫描 | 消毒门禁（身份/路径/体积 + 终检 fail-closed） | asm 有安全扫描；rulesync 有 lint 定位 |

**关键差距**：hub 的**零覆盖 + 全量备份 + 回滚**在同类中属上游水准，但**上游缺一环**：种子包内容没有完整性凭据。也就是说，「零覆盖」保证了不破坏用户数据，却不能证明**写入的内容本身**没被篡改。

### 维度 4 · 性能

| 项 | hub | 对标 |
|---|---|---|
| 启动成本 | 单文件，秒级 | vercel-labs `npx` 冷启动明显更慢 |
| per-skill 挂载 | 并发上限 8 + 原子 rename | vercel-labs symlink 逐条 |
| 运行时开销 | 常驻 HTTP 服务，可关 | 桌面应用常驻 |
| 大库规模 | **未做规模压测**（无实测数据） | — |

**评价**：单文件 + 原生链接在启动与挂载上优于 `npx` 路线；但**规模上限无实测数据**，是文档空白。

### 维度 5 · 可扩展性

| 扩展点 | hub | 对标 |
|---|---|---|
| 新增平台 | **改 `platforms.json` 一个文件，不改代码** | 各自需改代码或配置 |
| 新增挂载形态 | 需加 Go 代码 | 相当 |
| 新增前端页 | React 路由 | vercel-labs 无 GUI |
| 私有化适配 | 路径用 `~`/`%VAR%`/`$VAR`，禁个人绝对路径 | — |

**评价**：平台扩展性是 hub 的**最强项**，已做到配置化 + 零代码。

### 维度 6 · 维护状态

| 项 | hub | 对标 |
|---|---|---|
| 提交节奏 | 建仓 1 天，4 个 tag | vercel-labs 昨日仍有提交 |
| CI 维度 | 3 条 workflow / 4 类 job | vercel-labs 有 `.husky` + tests |
| 测试覆盖 | seed 4 例 + parity 测试 | — |
| advisory 腿 | **有，且明确声明「不接阻断链」** | 少见（多数把 advisory 混作 pass） |
| 三态语义 | **UNVERIFIED 与 OK 严格区分** | 同上 |

**评价**：**诚实度是 hub 的隐性优势**。parity 脚本把「测不到」明确区别于「相等」，并写明「a side could not be measured; this is not a pass」，且 CI 里用 `continue-on-error` 挂 advisory 腿——这在同类仓中罕见。

### 维度 7 · 文档完善度

| 项 | hub | 对标 |
|---|---|---|
| README 双语 | 中 192 行 / 英 194 行，**结构一一对应** | vercel-labs 有 zh 版本 |
| 安装文档 | Windows / macOS / Linux + 英文 Windows | skills-manager 有中文说明 |
| 路线图 | 有（20 KB，含未完成任务与优先级） | — |
| 分发策略说明 | 有（`SKILLS-TIERS.md` 三档） | — |
| 徽章 | **无** | 竞品普遍有 build/status/趋势徽章 |
| CONTRIBUTING | **无** | 多数有 |
| SECURITY | **无** | 多数有 |
| CHANGELOG | **无** | 多数有 |
| ARCHITECTURE | **无** | — |

**评价**：文档**内容质量高但门面缺失**。缺徽章意味着仓库首页无法呈现任何可信度信号；缺三件治理文档意味着贡献者与安全报告者没有落点。

### 维度 8 · 适用场景

| 场景 | hub 适配度 | 说明 |
|---|---|---|
| 本机多 Agent 共享记忆 + 技能 | **最优** | 双根 + 9 端 + 可逆 |
| 纯技能分发（团队/项目级） | 弱 | 无项目作用域、无 registry |
| 团队协作（多人共享一套技能） | 弱 | 无远程源、无 lockfile |
| CI 环境内的技能安装 | 中 | CLI 可用，但无 `--all`/`-y` 类批量语义 |
| 完全离线的隐私敏感环境 | **最优** | 不联网 + 回环 + ACL 加固 |

---

## 3. 改进建议清单

排序依据：**(真实事故/事故隐患) > (可信度门面缺失) > (能力生态位) > (锦上添花)**。每项标注完成标准，完成标准一律要求**可机器判定**。

### P0 —— 建议本轮落地

| # | 项 | 依据 | 完成标准 |
|---|---|---|---|
| **P0-1** | **清除仓内个人标识明文** | `[实测]` 种子构建脚本把个人姓名、学号、用户名作为**消毒规则的字面量**写进了公开仓（6 处）。消毒本身有效（种子包内零残留），但**规则字面量本身就是泄漏面**，且任何 fork/镜像/爬取都会带走 | 仓内 `grep` 个人标识 = 0 命中；消毒规则改为**通用启发式**（正则识别学号形态、用户目录形态），不再依赖针对特定个人的字面量；终检 fail-closed 保持 |
| **P0-2** | **版本一致性机器判据** | `[实测]` 版本号分散在 4 处（git tag / `router.go` 常量 / `platforms.json` / `seed.go`），且**已发生过实际漂移**（提交自述「drifted since v0.2.1, caught by docker round」）。靠人记必然复发 | CI 新增判据腿：tag ↔ 产品版本常量 ↔ 平台定义版本 三者一致；种子包版本另立子判据（语义独立，不与产品版本强绑）；**篡改反例腿必红**（否则判据视为未成立） |
| **P0-3** | **种子包 manifest（SHA-256）** | `[实测]` 种子包无任何完整性凭据。「零覆盖」保证不破坏用户数据，但无法证明**写入内容本身**未被篡改——这是分发链路上最实质的缺口 | 构建脚本生成 `manifest.json`（路径 + SHA-256 + 字节数，字典序保证确定性）；播种前**在任何写入之前** fail-closed 全量校验；`go test` 覆盖正例 + **篡改反例必失败**；独立校验脚本做第二实现交叉验（防「自比恒等」假通过） |
| **P0-4** | **治理五件套** | 原缺 CONTRIBUTING / SECURITY / CHANGELOG。安全闸有四道（Host 校验 / 来源白名单 / CORS / 恒定时间令牌比较）却无对外的 SECURITY 说明。**校正（2026-10-02 收口轮）**：本项落地时实际产出五件（增 ARCHITECTURE 与 FAQ），与 P1-3 合并计一项，故本表统一称「五件套」 | 五件套落盘并从 README 双向链接；SECURITY 明确「仅监听回环 + 令牌模型 + 漏洞报告渠道」 |
| **P0-5** | **README 徽章与定位对比** | 缺徽章 = 首页无任何可信度信号；缺定位说明 = 新访客无法判断「该不该用我」 | 徽章指向**真实 workflow**（CI / Release / License / 平台数）；补「同类项目定位对比表」；徽章链接实测可达 |

### P1 —— 建议次轮落地

| # | 项 | 依据 | 完成标准 |
|---|---|---|---|
| ~~P1-1~~ | ~~仓库卫生：构建产物已入库~~ | **`[否证实测]` 本条不成立**。初判依据是"dist 下存在 155 KB 的 JS 且仓库工作树干净"，据此推断产物被跟踪；随后用 `git ls-files agent/cmd/fenjue-agent/dist` 复验，实际**只有 `.gitkeep` 被跟踪**，`.gitignore` 规则正常生效（`git check-ignore -v` 命中 `.gitignore:4`）。那条 JS 只是本地构建产物，从未入库 | **无需处理**。保留本条作为「推断必须复验」的实例：同一次盘点里，另一处 `git grep` 得到的结论经复验成立，而这一处仅凭文件存在 + 工作树干净就下了结论，属于把「现象」当「机制」 |
| **P1-2** | **seed 内容安全扫描规则集扩展** | 现有门禁只查身份/路径/体积，**不查凭据形态与危险模式**（注入、危险命令、可执行片段） | 规则集扩展并**每条规则配正反例**；终检保持 fail-closed |
| **P1-3** | **ARCHITECTURE + FAQ** | 挂载三形态、软关闭语义、种子零覆盖、双根关系都只散落在 README 段落里，无整体视图 | 架构图 + FAQ 落盘并从 README 链接 |
| **P1-4** | **ROADMAP 增补** | 需登记：MCP/CLI 出口调研、parity 结果公开化、规模压测（当前无任何规模上限数据） | 增补项与本报告建议对齐，不重复既有 G0–G13 编号 |

### P2 —— 登记待办，本轮不做

| # | 项 | 理由 |
|---|---|---|
| **P2-1** | 命名预设（Presets）批量启停 | 生态位能力，需真实使用反馈再定形态 |
| **P2-2** | 项目级工作区 | 同上 |
| **P2-3** | Agent 自主管理出口（MCP/CLI） | 需先确定安全边界（令牌如何交给 Agent） |
| **P2-4** | 发行产物附校验和 | 优先级低于仓内完整性判据 |

### 明确**不做**（避免重复建设）`[实测]`

- **平台矩阵扩展**：本轮盘点发现 `platforms.json` **已是 9 端**（含 `zc`/`oc`/`qd`），相关路径定义与 parity 校验覆盖**已存在** ⇒ 对应设想项**降级为已存在**，不重复建设。仅 `mm`/`ds` 的端到端真实会话验证登记为待办（需真实会话配合，不自动化）。
- **双语对齐**：README 中文 192 行 / 英文 194 行，**结构已一一对应** ⇒ 降级为「随新章节同步」，不列为独立改进项。

---

## 4. 可实施路径

### 阶段 M1 · 可信度地基（P0-1 ~ P0-3）

1. 种子构建脚本改启发式规则，仓内个人标识归零
2. 新增版本一致性判据脚本 + 篡改反例腿
3. 种子包 manifest 生成 + 播种前 fail-closed 校验 + Go 测试双例 + 独立交叉验脚本
4. CI 接入两条新腿

**里程碑判据**：正常绿 + 篡改必红，两条都必须实跑见到输出。

### 阶段 M2 · 门面与治理（P0-4 ~ P0-5 + P1-1）

1. 治理五件套落盘（CONTRIBUTING / SECURITY / CHANGELOG / ARCHITECTURE / FAQ；架构与 FAQ 原为本报告 P1-3，落地时并入本阶段）
2. README 徽章 + 定位对比表
3. 徽章渲染核验（CI / Release / License / 平台数 四个徽章的链接与图片都要实测可达）

**里程碑判据**：徽章链接实测可达；README 与英文版章节结构一一对应。

### 阶段 M3 · 纵深（P1-2 ~ P1-4）

安全扫描规则集 + 架构文档 + 路线图增补。

### 阶段 M4 · 登记（P2）

预设 / 项目工作区 / Agent 出口 / 发行校验和 —— 只登记，不实现。

---

## 5. 本报告的已知局限

1. **星标是快照**，反映取数当刻的社区认可度，不反映质量；hub 建仓仅 1 天，★1 不具备可比性。
2. **三项 `[UNVERIFIED]`**（`swiz` / `CapSync` / `agentpull` 的机制描述）未参与任何评分，也未被当作「不存在」使用。
3. **性能维度无一手数据**：hub 未做规模压测，本报告不做任何性能数字断言。
4. **对标池偏窄**：以 GitHub 公开仓为限，未覆盖 IDE 插件形态与闭源同类产品。
5. **本报告不含任何私有体系细节**，仅描述公开仓自身与其公开竞品。

---

## 6. 来源

- hub 自身：`D:\global-memory-hub` @ `2cf6496`，2026-10-02 实测
- 对标元数据：GitHub REST API（认证态），取数当日 2026-10-02
- 各项目主页见 §1 表格内链接

---

## 7. 执行状态回写（2026-10-02 收口轮）

本节解决一个具体问题：§3 的建议清单与 §4 的阶段路径全部用**将来时**写成（「建议本轮落地」「本轮不做」），而报告产出后这些条目已陆续实施。读者若只读 §3/§4，会把已完成的事误判为待办。本节把**实施结果**逐条回写，与 [ROADMAP](ROADMAP.md) 的 G 编号一一对应。

### 7.1 对账基线（复算命令）

```
git log --oneline -12
git describe --tags
git rev-list --left-right --count origin/main...HEAD
gh run list --limit 10
```

对账时实测：`HEAD = 910cd0e`（`git describe --tags` = `v0.2.2-10-g910cd0e`）、工作树干净、与 `origin/main` **0 ahead / 0 behind**、最近 10 条 CI/Pages/Release 结论全 `success`。§6 记录的 `2cf6496` 是**报告取数时点**，不是当前 HEAD，两者不矛盾。

### 7.2 逐项落地状态

| 报告编号 | ROADMAP | 状态 | 落地提交 | 可复算命令 |
|---|---|---|---|---|
| P0-1 清除仓内个人标识明文 | G16 | **已落地** | `53d71a3`（另 `1a49ba8` 配套忽略本地消毒规则） | `python scripts/build_seed.py --selftest` |
| P0-2 版本一致性机器判据 | G17 | **已落地** | `f01ef24` | `python scripts/check_version_sync.py`（篡改常量 ⇒ rc=1；无 tag 浅克隆 ⇒ UNVERIFIED 不折算通过） |
| P0-3 种子包 manifest（SHA-256） | G18 | **已落地** | `f656252`，配套 `890b764` | `python scripts/verify_seed_manifest.py`（+1 字节 ⇒ rc=1）；Go 侧 `go test ./agent/internal/seed/` |
| P0-4 治理五件套 | G19 | **已落地** | `000a0b3` | 目录实测 `docs/{CONTRIBUTING,SECURITY,CHANGELOG,ARCHITECTURE,FAQ}.md` |
| P0-5 README 徽章与定位对比 | G19 | **已落地** | `d230a73` | README 徽章指向 `.github/workflows/{ci,release}.yml` 真实 workflow |
| P1-1 仓库卫生（构建产物入库） | — | **已否证，不需处理** | — | `git ls-files agent/cmd/fenjue-agent/dist` 只含 `.gitkeep`；`git check-ignore -v` 命中 `.gitignore:4` |
| P1-2 seed 内容安全扫描规则集扩展 | G19 | **已落地** | `53d71a3` | 每条规则配正反例，由 `--selftest` 强制 |
| P1-3 ARCHITECTURE + FAQ | G19 | **已落地**（并入 P0-4 计数） | `000a0b3` | 同 P0-4 |
| P1-4 ROADMAP 增补 | — | **已落地** | `3fc6e60` | G16–G26 已登记；G22–G26 保持 TODO |
| 平台矩阵扩展 | G21 | **已撤销（重复建设）** | — | `platforms.json` 实测已是 9 端；mm/ds 转正改由 G8 承载 |
| 双语对齐 | — | **降级为随新章节同步** | `d230a73` | README 中英双版徽章与定位表同步 |
| P2-1 ~ P2-4 | G22–G25 | **仍为 TODO（按计划本轮不做）** | — | ROADMAP 状态列 |

### 7.3 本轮（收口轮）实际改动

**无任何运行时行为变更**：未改 `agent/`、`scripts/`、`web/`、`site/`、`platforms.json`、`.github/workflows/`。改动面只有三份文档——本节、本报告两处「三件套→五件套」措辞校正（对齐 ROADMAP G19 的实际口径）、ROADMAP 的提交号补登与进度更新、CHANGELOG 一条记录。

### 7.4 仍开放的待办（不在本轮范围）

- `mm` / `ds` 由 beta 转 ga：程序侧装载行为已在 Docker 沙箱证实，**消费行为仍 UNVERIFIED**，需一台装有对应桌面客户端的干净机器跑 enable → 真实会话 → 读库验证（见 ROADMAP G8 与 [G8-DOCKER-PROTOCOL.md](G8-DOCKER-PROTOCOL.md)）。
- G22–G26 五项 TODO（预设批量启停 / 项目级工作区 / Agent 自主管理出口 / 发行校验和 / 规模压测）。其中 G24 明确**先定安全边界再实现**。
- 规模压测缺口：本报告 §5.3 已声明不做性能数字断言，在 G26 补数据前该口径不变。
