# Changelog

本项目遵循 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/)，版本号遵循 [语义化版本](https://semver.org/lang/zh-CN/)。

## [Unreleased]

### 新增

- 种子包完整性凭据：`pack/manifest.json` 记录每个播种文件的 SHA-256 与字节数。
  播种前在任何写入之前 fail-closed 全量校验，双向对账（清单登记但缺失、存在但未登记都判为差异）。
- `scripts/verify_seed_manifest.py`：与 Go 侧实现**相互独立**的第二套校验，用于交叉验证，
  避免「实现与自己一致」造成的假通过。
- `scripts/check_version_sync.py`：版本一致性判据，带 9 条漂移反例自测。
  判据按语义分层：产品版本（tag 对代码常量）、种子包版本（清单对代码常量）、
  平台定义版本（schema 后缀对配置版本号），不把语义不同的版本号硬绑在一起。
- CI 新增 `consistency` job（阻断型），覆盖上述判据。
- 种子内容安全门禁扩展：新增凭据形态与危险命令两类规则，每条规则配正反例，由 `--selftest` 强制。
- 治理文档：[CONTRIBUTING.md](CONTRIBUTING.md)、[SECURITY.md](SECURITY.md)、[ARCHITECTURE.md](ARCHITECTURE.md)。
- [BENCHMARK-2026-10-02.md](BENCHMARK-2026-10-02.md)：同类开源项目八维对标与改进清单。
- G8 沙箱取证协议：[G8-DOCKER-PROTOCOL.md](G8-DOCKER-PROTOCOL.md) 与 `scripts/g8_sandbox/`。
  在隔离容器内对 `mm` / `ds` 跑 enable/verify/link/zero-overwrite 与一条反例腿，
  输出 NDJSON、退出码三态。实测 rc=0，程序侧装载行为由此证实；消费行为仍 UNVERIFIED。
- **规模压测**：[scripts/bench_scale.py](scripts/bench_scale.py) 与 [PERFORMANCE.md](PERFORMANCE.md)。
  三档规模（9/50/200 端、50/300/1000 技能、500/3000/10000 库文件），五项指标取中位数：
  冷启动恒定约 0.5 s（库文件翻 20 倍也不变）、单端启用恒定（端数翻 22 倍也不变）、
  技能挂载吞吐 40–65 links/s（**单端 1000 技能约需 22 s**，是同维度唯一实质短板）、
  全量校验随（端数 + 技能数）线性而**与库文件数无关**、常驻内存 11.2–13.8 MB（下界，内嵌前端为占位产物）。
  压测在仓内沙箱进行，子进程带重定向的 `USERPROFILE`/`HOME`，**不触碰真实 `~/.fenjue`**；
  端口向系统动态申请，避免残留进程造成假失败。同档重跑抖动可达 3 倍，一次被磁盘负载污染的样本已丢弃并写进文档。
- **HTTP API 文档**：[API.md](API.md)。7 个端点的鉴权、请求体、响应体、状态码与四道安全闸对调用方的影响，
  含三处与直觉相反的事实（鉴权头是 `X-Fenjue-Token` 而非 `Authorization`；Origin 白名单只对带 Origin 头的请求生效；
  `POST /api/roots` 的 `applied` 恒为 false，因为需重启才生效）。
- **API 契约判据**：[scripts/check_api_contract.py](scripts/check_api_contract.py)。
  文档与 `router.go` 路由注册表**双向对账**（少写或多写都判红），`--live` 模式在沙箱起真实 serve 实测状态码
  （22 项检查）。阻断型腿进 `consistency` job，实测腿以 advisory 腿单独接入。
- **命名预设（批量启停）**：`platforms.json` 新增可选 `presets` 段（1.1.0），
  `fenjue-agent preset <name> [--action enable|disable] [--dry-run]` 与 `POST /api/presets/{name}`，
  控制台新增预设卡片。成员由配置决定，程序不内置任何分组；编排层复用既有 `Enable`/`Disable`，
  因此备份、回滚、幂等与注入语义与单端完全一致。批量停用一律软关闭。
  判据：[scripts/check_preset_config.py](scripts/check_preset_config.py)（Python 独立第二实现）。
- **只读出口**：`fenjue-agent export [--include-content] [--format json|markdown]`。
  给脚本与另一个 AI 端读取统一库现状：**只读、不联网、不监听端口、不需令牌、默认不含正文**。
- 发行产物附带 `SHA256SUMS`（合并全部构建产物后生成，并在发版前核对条目数）。
- 对外主站新增**文档中心**（12 个文档入口，逐条实测可达），此前只链到仓库首页与安装文档。

### 变更

- `platforms.json` 版本 1.0.0 → 1.1.0：新增可选 `presets` 段。向后兼容，没有该段的旧配置照常加载（零预设）。

- 种子构建脚本的消毒规则由**针对特定个人的字面量**改为**通用启发式**（正则形态）。
  原写法把个人标识写进了随仓库公开的脚本里，字面量本身即泄漏面；
  改后规则对任何用户名与私有根名都生效，精度靠本地不入库文件 `scripts/sanitize.local.json` 补齐。
- 路径消毒后统一分隔符，避免产出混合形态路径。
- 对标报告与路线图做了一次双向对账（提交号 ↔ 条目），并把实施结果回写为报告 §7「执行状态回写」：
  原 §3/§4 全部用将来时写成，条目落地后读者会把已完成的事误判为待办。
  对账同时校正了报告与 [ROADMAP](ROADMAP.md) 之间「治理三件套 / 五件套」的措辞不一致。
  **本条为纯文档变更，无任何运行时行为改动。**

### 修复

- 盘符残留规则不再误伤 URL（`https://` 里的 `s:/`、`file:///` 里的盘符），
  也不再误伤形如 `https://example.com/home/...` 的链接。

## [0.2.2] - 2026-10-02

### 修复

- 版本常量从 v0.2.1 补齐到 v0.2.2（此前由 docker 轮发现漂移）。
- 平台 `hm` 的注入点按实盘证据校准为 `SOUL.md`。

## [0.2.1]

## [0.2.0]

## [0.1.0]

[Unreleased]: https://github.com/lxh113377/global-memory-hub/compare/v0.2.2...HEAD
[0.2.2]: https://github.com/lxh113377/global-memory-hub/compare/v0.2.1...v0.2.2
[0.2.1]: https://github.com/lxh113377/global-memory-hub/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/lxh113377/global-memory-hub/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/lxh113377/global-memory-hub/releases/tag/v0.1.0
