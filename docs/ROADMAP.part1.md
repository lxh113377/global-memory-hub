# ROADMAP §6 分卷 — 判据实测记录

> 本卷是 ROADMAP.md **§6 的正文**，从主壳下沉而来（2026-10-14 r183，主壳因 G30 体量治理）。
> 主壳 §6 只留指针。**只搬不删**：下沉时原文逐行迁入，未做任何改写；回迁方式见卷末。
> §6.5 / §6.6 含本文件数处「被实测否证的旧断言」，引用它们时必须连同否证结论一起读——
> 那些断言散落在 §1 / §3 / DEPLOY.md 的正文里，本轮只声明不回改（回改即制造双源）。

## 6. 判据实测记录（2026-10-14 · r181 建 / r182 续）

> **本块删除即 G32 回退为未闭环。** 本块是 §5 增行的取证底账：每条读数来自本轮实跑的命令回显，不来自任何计划快照。
> 纪律同 §1 第 20–21 行：**不重贴被禁字面量**——判据里的个人标识正则只写形态，真值在本地不入库文件 `scripts/sanitize.local.json`。

### 6.1 本轮实跑的判据（解释器一律写绝对路径）

| 判据 | 命令 | 本轮读数 |
|---|---|---|
| G4 | `platform_parity.py --private <私有仓根>` | `PARITY OK` rc=0，两侧各 9 端 id 同集 |
| G4 | `pytest tests/test_platform_parity.py -q` | `1 passed`（须用 managed venv；系统 3.13.12 无 pytest） |
| G17 | `check_version_sync.py` / `--selftest` | `VERSION SYNC OK` rc=0 / `[GATE:version-selftest-pass] 9 drift + 1 clean` |
| G18 | `verify_seed_manifest.py` | 29 文件 ⇄ 29 登记全等，rc=0，`[GATE:seed-manifest-pass]` |
| G22 | `check_preset_config.py` / `--selftest` | rc=0（9 平台 / 2 预设）/ `[GATE:preset-selftest-pass] 10 rules + 2 clean` |
| G25 | `gh release view v0.3.0 --json assets` | 6 件：5 平台二进制 + `SHA256SUMS` |
| G26 | `bench_scale.py --selftest` | `[GATE:bench-selftest-pass]` |
| G27 | `check_api_contract.py` / `--selftest` | `API CONTRACT OK` rc=0 / `[GATE:api-contract-selftest-pass]` |
| G16 | 全仓个人标识扫描 | rc=0、2 命中，**均良性**（占位符 + 规则描述本身），真实标识 0 |
| G19/G28 | 九份治理文档 `Test-Path` | 九份全在 |
| G21 | `platforms.json` 解析 | 9 端：wb tr cx hm zc oc qd mm ds |

### 6.2 四处与 §1 原文不符的实测差异（只声明，**不回改 §1**，回改即制造双源）

1. **G4 腿数。** §1 写「带 **5 条变异腿**自测」；`tests/test_platform_parity.py` 的 `CASES` 实为 **4 条 = 1 正例（`control-unchanged`，期望 `EXIT_OK`）+ 3 变异**（`hub-missing-platform` / `hub-skills-kind-flipped` / `private-missing-junction-record`，均期望 `EXIT_DIFF`）。
2. **两个脚本没有 `--selftest`。** §1 G4 与 G18 的措辞容易让人写成 `… --selftest`，但 `platform_parity.py`（仅 `--hub/--private/--json`）与 `verify_seed_manifest.py`（仅 `--json`）实测均 `rc=2` usage 报错。前者的变异腿在 **pytest**，后者的反例腿在 **Go `manifest_test.go`**。
3. **`gh` 被 0 字节 shim 遮蔽。** `gh release view` 直接调用报 `Cannot run a document in the middle of a pipeline`；真身在 `C:\Program Files\GitHub CLI\gh.exe`。任何以 `gh` 开头的判据都必须带绝对路径，否则判据会在**别人的机器上**以不同方式失败。
4. **`--live` 与 `SKIPPED`。** `check_api_contract.py` 默认只静态对账，实测回显含 `live probe: SKIPPED`；该腿未跑不得记为通过。

### 6.3 第一轮未执行项（已于 §6.5 全部关闭，此段保留作时序）

> **本段描述的是 r181 当时的取数状态，已被 §6.5 取代——保留是为了留痕，不是现状。**

G15 / G18 的 Go 腿、G20 的 §7.2 对账、G24 的 `export` 冒烟、G28 的线上可达性，在 r181 均**未执行**。当时归因写的是「依赖 Go 工具链（本机不在 PATH，见 §3.10）或沙箱构建」——**其中「Go 不在 PATH」这个归因本身是错的**，见 §6.5 否证第 1 条：Go 一直在 PATH 上，这五项从来就能跑，是我没跑。

### 6.4 本轮未改动的东西

§1 全部 G 行、§0 读法、§2 优先级排序、§3 全部小节、§4 里程碑依赖图**均未改写**。本轮只做两件事：把 §5 的「G15–G30 本节未覆盖」缺口行换成实测行、补本节。

### 6.5 第二轮取证（2026-10-14 · r182）：补齐五项「未执行」，并否证三条旧断言

**补齐的（§5 对应行已逐条回填读数）**

| 项 | 实测 |
|---|---|
| G15 Go 腿 | `go test -count=1 -v ./internal/seed/` → 5 条 bootstrap 腿全 PASS；`go vet ./...` rc=0 |
| G18 Go 腿 | 同命令 7 条 `TestVerifyPack*` 全 PASS ⇒ **与 §1「1 正例 + 5 篡改反例 + 1 端到端」逐条对上** |
| G22 Go 腿 | `go test -count=1 -v ./internal/state/` → 6 条 `TestApplyPreset*` 全 PASS |
| G20 | §7.2 逐项对账表 11 行齐全，每行带落地提交 + 可复算命令 |
| G24 | 打在 **v0.3.0 下载产物**上 rc=0；默认 2,794 B vs `--include-content` 199,032 B（差 196,238 B，含 `MEMORY.md`）；毒令牌哨兵未被读；唯一落盘副作用是 0 字节日志 |
| G28 | 主站 20 链接，含 `#docs` 与七份文档的仓库直链 |
| G2 跨站腿 | **仍 `UNVERIFIED`**：`Test-NetConnection` 到 `lxh113377.github.io:443` = False（DNS 正常解析到 185.199.110.153，TCP 建连失败），`curl` exit 28 / http 000。镜像腿四件全 200 且与本地 `web/dist` 逐字节同源（`index-BX4eCaU5.js` 177,487 B、`index-B-NFB106.css` 20,566 B） |

**被实测否证的三条（只声明，不回改原行）**

1. **§3.10 第 1 条「Go 工具链不在 PATH」为假** —— `where go` → `C:\Program Files\Go\bin\go.exe`，go1.27.0，在 PATH 上。已登记 G35。本轮我据这条把 G15/G18/G22 误判为「未执行」，实为误判。
2. **本地 exe 没有 `export` 子命令** —— 真因是**本地 exe 陈旧**（早于 `export.go` 加入），不是功能缺失。同一命令打在 v0.3.0 产物上正常工作。
3. **`agent/cmd/fenjue-agent/dist` 与线上 `web/dist` 资产 hash 不同** —— 曾疑为「二进制内嵌的是陈旧前端」，落笔前用 `git check-ignore` + `git ls-files` + exe 字节串探针否证：`dist/*` 由 `.gitignore:4` 忽略、`fenjue-agent.exe` 由 `agent/.gitignore:1` 忽略，二者都是**未入库的本机构建产物**，各自与自身构建时刻自洽，**不构成对外交付缺陷**。

> 第 2、3 条是本轮 §5 硬前置（「判据落笔前必须实跑」）的**第一次实战**：两条「发现」都是在准备落笔时被对照实验按死的，其中第 3 条已经跑完了取数、正准备写进文档，被 `git check-ignore` 反证。**没这道闸，本节会多出两条不存在的缺陷。**

**新登记** G34（镜像站 200 兜底使 HTTP-200 成为空判据）、G35（§3.10 文档漂移）。

### 6.6 第三轮（G34 处置 · r183）：一次静默失败，和它是怎么被抓到的

| 阶段 | 不存在的路径 `/totally/bogus/path` | 既有四件 |
|---|---|---|
| 修复前（基线） | `200 \| 13741 \| text/html` | 200，字节不变 |
| 第一次重投（**只加** `_routes.json`） | `200 \| 13741 \| text/html` ← **完全没变** | 200，字节不变 |
| 第二次重投（加 `404.html`） | **`404 \| 143`** ✅ | 200，字节不变 |

**这一轮最有价值的东西是中间那一行。** `_routes.json` 单独投放后行为与修复前**逐字节相同**——
Cloudflare 把 `not_found_handling: "404-page"` **静默忽略**了，既不报错也不警告。若验收只打首页和已知资产，
这一轮会被判成「已修复」并合入。抓住它的唯一动作是：**每次投放后都去探一个保证不存在的路径**。

「保证不存在」是判据成立的前提——本轮第一次探 `/console/assets/index-Ci1XKrNf.js` 时拿到 200，
差点被读成「新资产没上」；实际它是**旧的真实资产**，只是本地 `web/dist` 已换成新 hash。
换成 `/totally/bogus/path` 这种语义上不可能存在的路径，判据才立得住。

顺带两件事：

- **DEPLOY.md 的 CF 段本轮改写。** 实测 `wrangler pages project list` 回 `Git Provider = No`
  ⇒ 镜像**非** Git-connected，push 到 `main` 不会自动重部署。原文第 13–18 行的
  「Connect to Git / push 自动重部署」从未成立。已改为可执行的手动步骤（含目录必须作位置参数，
  只 `cd` 不生效）。
- **`site/404.html` 与 `site/_routes.json` 会一并进入 GitHub Pages 产物根**（组装命令是 `cp -r site/.`）。
  GitHub Pages 不消费 `_routes.json`，只会把它当普通静态文件吐出 ⇒ 惰性无害，已核实。

---

**回迁与合并。** 本卷是 append-only 的过程证据，不参与「当前状态」判读——想知道现状读主壳 §1（任务台账）、
§5（验收判据）、§2（优先级）。本卷与主壳同名文件不同层：ROADMAP.md 是权威面，本卷是其 §6 的展开。
若主壳需要引用某条取证，直接引本卷的节号（§6.2 / §6.5 / §6.6），不要复制内容回主壳——那会把
append-only 的证据面变成需要逐处同步的镜像面。
