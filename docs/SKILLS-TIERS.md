# 技能三档分发登记（G7）

> 口径依据：本机技能注册表实测 **169 条**，其中原创 **17 条（10.1%）**、上游引入 **152 条**。
> 「全量打包开源」在版权与隐私两侧都不可行，故分三档（详见 ROADMAP §1.1）。

## 三档定义

| 档 | 语义 | 分发方式 |
|---|---|---|
| **可开源档** | 原创或可再分发的通用方法论技能，过消毒门禁后随产品分发 | 打进种子包（`agent/internal/seed/pack/skills/`），`go:embed` 随二进制离线可用 |
| **上游引用档** | 他方作品，不复制文件，只登记来源与获取方式 | 本文件登记来源清单；用户按上游渠道自行获取 |
| **不公开档** | 含个人标识/私有项目痕迹，永不出库 | 不登记内容，只登记条数 |

## 可开源档（当前种子包，18 条）

由 `scripts/build_seed.py` 从私有库精选生成，全部过身份/路径消毒 + 终检零残留门禁：

brainstorming, first-principles-decomposer, verification-before-completion,
test-driven-development, refactoring, debugging-fixing, code-review,
writing-plans, doc-coauthoring, knowledge-capture, data-analysis,
research-documentation, spec-to-implementation, scope-creep-detector,
consulting-analysis, prompt-consolidation, diagram-maker, frontend-design

复算：`python scripts/build_seed.py --dry-run`（picked=18）；终检 `[GATE:seed-clean]`。

## 上游引用档（152 条，逐条来源登记待补）

政策：**不随产品分发任何文件**。每条只登记「技能 id → 上游项目与获取渠道」。此登记表为增量工作，
按需补录（用户启用技能库后按请求补齐高频项）；在补齐之前，本档一律视为「引用未登记」，不提供分发。

## 不公开档

条数：实测 0 条被分发（种子包构建门禁保证：命中身份模式的技能在生成期即被丢弃）。
私有库中含个人标识/商业项目痕迹的条目永远不会进入 `agent/internal/seed/pack/`。

## 门禁

1. 种子包生成：`scripts/build_seed.py` —— 身份模式（姓名/学号/称呼）命中即整技能丢弃；
   路径消毒（`C:\Users\*`、`D:\global_*` → `~`/`~/.fenjue/*`）后残留即丢该文件；终检零残留才输出 `[GATE:seed-clean]`。
2. 只复制 SKILL.md 与 `references/**/*.md`，跳过 scripts/assets/二进制（供给链安全）。
3. 单文件 >100KB 丢弃。
