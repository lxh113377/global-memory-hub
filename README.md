# 焚诀 Global（global-memory-hub）

[![CI](https://github.com/lxh113377/global-memory-hub/actions/workflows/ci.yml/badge.svg)](https://github.com/lxh113377/global-memory-hub/actions/workflows/ci.yml)
[![Release](https://github.com/lxh113377/global-memory-hub/actions/workflows/release.yml/badge.svg)](https://github.com/lxh113377/global-memory-hub/releases)
[![License: MIT](https://img.shields.io/badge/license-MIT-2ea44f.svg)](LICENSE)
[![Platforms](https://img.shields.io/badge/platforms-9-8A2BE2)](platforms.json)
[![Go](https://img.shields.io/badge/go-1.x-00ADD8?logo=go&logoColor=white)](agent/go.mod)

> English version: [README.en.md](README.en.md)

让多个 AI Agent 共享**同一份本地记忆库与技能库**，改一处、N 端同步生效。

不联网、不上传、不放后台。所有内容存在你自己的机器上，程序只监听本机回环地址。

---

## 它解决什么问题

同时用两个以上 AI 编程助手的人都撞过同一堵墙：

- **记忆分裂** —— 你在 A 里教会它的东西，B 完全不知道；同一份知识被反复重写。
- **技能散落** —— 同一个技能在 A 的目录下装了一份，在 B 的目录下又装了一份，改了一个忘了另一个。
- **无法统一** —— 各家程序把记忆和技能放在各自的私有目录里，互相不可见。

`焚诀 Global` 的做法：在你本机建**一处**权威的记忆库与技能库，再让各个 Agent 程序通过目录链接指向它。此后无论从哪个 Agent 修改，其余 Agent 立刻看到同一份内容。

关闭某个 Agent 的接入是**可逆的**：默认走「软关闭」，保留链接、只停用注入，随时可一键恢复。

---

## 三步上手

### 1. 拿到程序

预编译的三平台单文件程序发布在本仓库的 Releases 页。若尚未发布或你偏好自行构建，见下方[从源码构建](#从源码构建)。

### 2. 运行

```bash
./fenjue-agent serve
```

程序会在本机回环地址 `127.0.0.1:7799` 上启动一个控制台，并在终端打印一条**握手链接**，形如：

```
Handshake: http://127.0.0.1:7799/console#token=<一串随机十六进制>
```

点开这条链接即可打开控制台。链接里带的是本次启动随机生成的访问令牌；令牌只在你本机生成，每次启动都会换新。

### 3. 打开开关

控制台会列出内建支持的 Agent 平台。每个平台一张卡片，显示它当前是**已接入**、**未接入**还是**连接异常**，并列出实际的挂载路径。需要哪个就打开哪个。

想关掉的时候再点一次即可。默认的关闭方式不会删除任何东西，原始文件在操作前已做备份。

---

## 内建支持的平台

| 标识 | 平台 | 记忆挂载 | 技能挂载 |
|---|---|---|---|
| `wb` | WorkBuddy | 目录链接 | 目录链接 |
| `tr` | TRAE SOLO CN | 目录链接 | 目录链接 |
| `cx` | Codex CLI | 目录链接 | 物理镜像（一份真实拷贝，非链接） |
| `hm` | Hermes Agent | 目录链接 | 逐技能链接（每条技能一个链接） |
| `zc` | ZCode | 目录链接 | 目录链接 |
| `oc` | OpenCode | 目录链接 | 目录链接 |
| `qd` | Qoder CN | 目录链接 | 目录链接 |
| `mm` | MiniMax Code | 目录链接 | 目录链接 |
| `ds` | DeepSeek Harness Desktop | 目录链接 | 目录链接 |

**关于三种挂载形态。** 上表里 `cx` 与 `hm` 与其余几个**行为不同**，控制台会用不同徽标标注：

- **目录链接**：一个目录指向另一个目录。开关语义最直接。
- **物理镜像**（`cx`）：真实复制而非链接，因此「关闭」的含义是**停止同步**，而不是断开一个链接。已同步过去的文件不会被自动删除。
- **逐技能链接**（`hm`）：约一百六十条独立链接。启用与关闭是批量操作，程序做了并发上限与失败回滚，不会留下半成品。

平台定义集中在仓库根的 `platforms.json`。新增一个平台只需要改这个文件，不需要改代码。

---

## 命令行

```
fenjue-agent serve    [--port 7799] [--platforms <path>] [--origin <url>]...
fenjue-agent verify   [--platforms <path>]
fenjue-agent enable   <platform-id>  [--platforms <path>]
fenjue-agent disable  <platform-id>  [--soft=true] [--platforms <path>]
fenjue-agent version
```

| 参数 | 说明 |
|---|---|
| `--port` | 监听端口，默认 `7799`。**只绑定 `127.0.0.1`，不接受外部连接。** |
| `--platforms` | 平台定义文件路径。默认从仓库根自动定位。 |
| `--origin` | 追加允许访问的网页来源，可重复。指向你部署控制台的域名。 |
| `--soft` | 关闭方式，默认 `true`（软关闭，保留链接只停注入）。 |

`verify` 子命令不带 `--platforms` 也能用，适合放进脚本里做例行体检。

---

## 安全设计

本地程序一开，**任何网页**理论上都能向你本机的回环端口发请求。为此程序强制四道闸，任一不满足直接拒绝：

| 闸 | 作用 |
|---|---|
| **Host 头校验** | 只接受 `127.0.0.1:<port>` 与 `localhost:<port>`，封死 DNS 重绑定类攻击 |
| **来源白名单** | 只放行本站域名与你显式通过 `--origin` 添加的来源；其余一律拒绝 |
| **CORS 预检应答** | 对白名单来源正确应答跨域预检，让公网页面调用本地服务在浏览器侧可行 |
| **访问令牌** | 除健康检查外所有接口都要带令牌；比较使用恒定时间实现，避免时序侧信道 |

另外：

- 进程**只绑定回环地址**，不监听 `0.0.0.0`，局域网内其他机器无法直接访问。
- 令牌每次启动重新生成，写在本机受限文件里，并在终端打印的握手链接中通过 URL 片段携带（片段不会随请求发给服务器，也不会进入 Referer）。
- 日志不打印令牌明文。

---

## 数据与备份

程序自身的目录都在 `~/.fenjue/` 下：

| 路径 | 内容 |
|---|---|
| `~/.fenjue/token` | 本次启动的访问令牌 |
| `~/.fenjue/trash/<备份号>/` | 改动前的原始文件备份（含 `manifest.json` 变更清单） |
| `~/.fenjue/logs/agent.log` | 运行日志 |
| `~/.fenjue/memory` | 默认记忆库根 |
| `~/.fenjue/skills` | 默认技能库根 |

**任何写操作前都会先备份。** 响应里回传备份号，控制台据此提供一键还原。备份只增不改，还原只读备份不读旧状态。

记忆库根与技能库根都可以改（在控制台的设置页，或直接编辑 `platforms.json` 的 `roots`）。

---

## 从源码构建

需要 Go 与 Node.js。**构建顺序不能颠倒**：Go 侧会把前端产物内嵌进二进制，所以前端必须先构建。

### Windows

```powershell
git clone <repo-url> global-memory-hub
cd global-memory-hub\web
npm ci
npm run build
cd ..\agent
pwsh .\build.ps1
```

`build.ps1` 会先把 `web\dist` 同步到内嵌目录再编译。若 `go` 不在 `PATH` 上，脚本会回退到默认安装路径；若你的安装位置不同，请直接把 `go` 的绝对路径写进脚本或加进 `PATH`。

### macOS / Linux

```bash
git clone <repo-url> global-memory-hub
cd global-memory-hub/web
npm ci
npm run build
cd ../agent
mkdir -p cmd/fenjue-agent/dist
cp -r ../web/dist/. cmd/fenjue-agent/dist/
go build -o fenjue-agent ./cmd/fenjue-agent
```

**为什么内嵌目录里必须至少有一个文件。** Go 的内嵌指令要求目标目录非空，否则**编译期**直接报错。仓库里为此保留了一个占位文件；上面的 `cp` 步骤会把真实产物覆盖进去。若你跳过了前端构建，程序仍能编译，但打开控制台会提示前端产物缺失。

---

## 平台兼容性

| 平台 | 链接实现 | 是否需要提权 |
|---|---|---|
| Windows | 目录联接（junction） | **不需要**管理员权限 |
| macOS | 符号链接 | 不需要 |
| Linux | 符号链接 | 不需要 |

Windows 上刻意不采用需要开发者模式或提权的符号链接形式。

---

## 和其他方案的区别

同样在解决「多个 Agent 共用一份配置/技能」的问题，但选择不同：

| 方案 | 覆盖端数 | 共享的是什么 | 形态 | 差别在哪 |
|---|---|---|---|---|
| **焚诀 Global（本项目）** | 9 | **记忆 + 技能** | 单文件程序 + 浏览器控制台 | 目前同类里**把记忆库也纳入共享范围**的方案；可逆软关闭 + 全量回滚 |
| [vercel-labs/skills](https://github.com/vercel-labs/skills) | 79 | 技能 | Node CLI | 生态量大得多；只管技能，不管记忆；面向分发而非本地共享 |
| [xingkongliang/skills-manager](https://github.com/xingkongliang/skills-manager) | 50+ | 技能 | 桌面应用 | 有市场、预设、多设备同步；不做记忆 |
| [dyoshikawa/rulesync](https://github.com/dyoshikawa/rulesync) | — | 规则文件 | CLI | 面向各家的规则配置文件，不管理记忆/技能库内容 |
| 各 Agent 自带配置 | 1 | 只有它自己 | 各家程序 | 天然分裂，改一处只改一处 |

一句话概括差异：**别人主要在管「技能怎么装」，本项目管「记忆和技能怎么在你自己的机器上保持同一份」。**
所以本项目刻意不做技能市场、不做多设备云同步 —— 那需要联网，与「不联网」的承诺冲突。

更完整的八维对比（含逐项差距量化与改进清单）见 [对标报告](docs/BENCHMARK-2026-10-02.md)。

---

## 相关文档

- [安装说明 · Windows](docs/INSTALL-windows.md)
- [安装说明 · macOS](docs/INSTALL-macos.md)
- [安装说明 · Linux](docs/INSTALL-linux.md)
- [架构说明](docs/ARCHITECTURE.md) —— 全局视图、三种挂载形态、令牌在请求链路上的位置
- [常见问题](docs/FAQ.md)
- [安全策略](docs/SECURITY.md) —— 四道安全闸、已知限制、漏洞报告方式
- [贡献指南](docs/CONTRIBUTING.md) —— 构建顺序、判据清单、种子包规约
- [变更日志](docs/CHANGELOG.md)
- [技能分档说明](docs/SKILLS-TIERS.md) —— 哪些技能可以公开分发，为什么
- [路线图：未完成任务与后续规划](docs/ROADMAP.md)
- [HTTP API](docs/API.md) —— 全部端点的鉴权、请求/响应、状态码与四道安全闸对调用方的影响
- [性能实测](docs/PERFORMANCE.md) —— 三档规模实测数据、复杂度归因与边界声明
- [对标报告 · 2026-10-02](docs/BENCHMARK-2026-10-02.md) —— 同类项目八维对比

---

## 许可

MIT。详见 [LICENSE](LICENSE)。
