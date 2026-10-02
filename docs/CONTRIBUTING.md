# Contributing to 焚诀 Global

感谢你愿意参与。本项目刻意保持很小，但有几条约定必须守住，否则会破坏它对用户承诺的性质。

## 它承诺了什么

三句话，贡献时不要违背：

1. **不联网、不上传、不放后台。** 任何新代码都不得引入出网请求。
2. **改一处、N 端同步生效。** 权威库只有一份，其余端都是链接。
3. **关闭是可逆的。** 任何写操作前先备份，默认软关闭（保留链接、只停注入）。

## 开发流程

### 构建顺序不能颠倒

Go 侧把前端产物内嵌进二进制，所以**前端必须先构建**：

```bash
cd web
npm ci
npm run build
cd ..
```

Windows 用 `pwsh .\agent\build.ps1`，macOS / Linux 见 README。

### 提交前必跑

```bash
cd agent && go vet ./... && go test ./...
cd .. && python3 scripts/build_seed.py --selftest
cd .. && python3 scripts/check_version_sync.py --selftest
cd .. && python3 scripts/check_version_sync.py
cd .. && python3 scripts/verify_seed_manifest.py
```

后四条在 CI 里同样是**阻断型**判据，本地红了就别提交。

其中 `--selftest` 两条不是形式：它们验证「判据本身会咬人」。一条没有反例腿的判据等于没有判据，所以新增或修改任何判据时，**必须同时补上会让它变红的反例**。

### 版本号改哪个

| 文件 | 含义 | 什么时候 bump |
|---|---|---|
| `agent/internal/server/router.go` 的 `Version` | 产品版本 | 发版时，必须与 git tag 一致 |
| `agent/internal/seed/seed.go` 的 `SeedVersion` | 种子包内容版本 | 改动 pack 内容时，与产品版本独立 |
| `platforms.json` 的 `version` | **配置格式**版本 | 改平台定义格式时，与 `schema` 后缀配套 |
| `pack/manifest.json` 的 `seed_version` | 播种内容版本（由脚本生成） | 不要手改，跑 `--manifest-only` 重算 |

改完 pack 内容后**必须**重算 manifest：

```bash
python3 scripts/build_seed.py --manifest-only
```

漏了这一步，`seed.VerifyPack` 会在播种前 fail-closed 拒绝（这是设计意图，不是 bug）。

## 种子包改动规约

`agent/internal/seed/pack/` 里的内容会被**打进二进制分发给所有用户**，因此门槛高于普通代码：

- 只放原创或可再分发的通用方法论内容，判定标准见 [SKILLS-TIERS.md](SKILLS-TIERS.md)
- 不含任何个人标识（姓名、学号、用户名、绝对路径）
- 不含凭据形态、不含危险命令
- 上述两条由 `build_seed.py` 门禁强制，且**终检 fail-closed**

需要更高精度的个人化规则时，写进 `scripts/sanitize.local.json`（该文件已 gitignore，不入库），
**不要**把字面量写回 `build_seed.py` —— 那个文件随仓库公开，字面量本身就是泄漏面。

## 新增一个 Agent 平台

只改 `platforms.json`，不要改代码。规约：

- 路径只用 `~` / `%VAR%` / `$VAR`，**禁止个人绝对路径**
- `support` 填 `ga`（已实测）或 `beta`（已接入但消费行为未验证）
- 不确定消费行为就填 `beta` 并在 `notes` 写明，别提前宣称 `ga`
- 改完跑 `python3 scripts/platform_parity.py`（需要私有权威文件，缺侧会报 UNVERIFIED，属正常）

## 代码风格

- Go：`gofmt`，注释用中文，**代码与注释里不要用中文引号**
- Python：`from __future__ import annotations`，UTF-8 + LF，注释用中文
- 注释写**为什么**，不写**做了什么**

## 提交与 PR

- 一个逻辑单元一次提交
- PR 描述里说明：改了什么、为什么、怎么验证的
- 涉及 pack 的改动，PR 里必须贴出 `--selftest` 与 `go test` 的输出

## 禁入内容（会被门禁拦下）

| 类别 | 例子 |
|---|---|
| 个人标识 | 真实姓名、学号、工号、用户名、绝对路径 |
| 凭据 | 任何形态的 token / 私钥 / 密钥赋值 |
| 危险命令 | `curl ... \| sh`、`rm -rf /`、`Remove-Item ... -Recurse` 之类 |

## 行为准则

参与即视为同意：保持尊重、就事论事、对代码质量负责。欢迎指出问题，也欢迎被指出问题。
