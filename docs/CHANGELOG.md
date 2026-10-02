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

### 变更

- 种子构建脚本的消毒规则由**针对特定个人的字面量**改为**通用启发式**（正则形态）。
  原写法把个人标识写进了随仓库公开的脚本里，字面量本身即泄漏面；
  改后规则对任何用户名与私有根名都生效，精度靠本地不入库文件 `scripts/sanitize.local.json` 补齐。
- 路径消毒后统一分隔符，避免产出混合形态路径。

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
