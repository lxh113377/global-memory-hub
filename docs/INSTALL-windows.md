# 安装说明 · Windows

> 适用：Windows 10 / 11。全程**不需要管理员权限**。
> 约定：`<repo>` 指本仓库的克隆路径；`%USERPROFILE%` 由系统展开为你的用户主目录。

---

## 0. 前提

- 本机已安装至少一个要接入的 Agent 程序（未安装的平台在控制台里会显示为「未接入」，属正常）。
- 若要自行构建：需要 Go 与 Node.js。只跑预编译包则不需要。

---

## 1. 拿到程序

从本仓库 Releases 页下载 Windows 版本的单文件程序 `fenjue-agent.exe`，放到一个你方便调用的位置，例如：

```powershell
mkdir "%USERPROFILE%\fenjue"
# 把 fenjue-agent.exe 移动进去
```

> 不要放在被系统清理的临时目录里。程序会在 `%USERPROFILE%\.fenjue\` 下保存令牌、日志与备份，程序本体放在哪不影响这些。

**首次运行会被 SmartScreen 拦一次**，因为文件没有付费签名证书。在弹窗里选择「更多信息」→「仍要运行」即可。这是免费分发且未做代码签名的必然结果，不是文件损坏。

---

## 2. 首次运行

```powershell
"%USERPROFILE%\fenjue\fenjue-agent.exe" serve
```

正常输出形如：

```
fenjue-agent 0.1.0 (windows/amd64)
platforms: <repo>\platforms.json
token file: %USERPROFILE%\.fenjue\token
Handshake: http://127.0.0.1:7799/console#token=<随机十六进制>
listening on 127.0.0.1:7799
```

把 `Handshake` 那一行整条**复制到浏览器**打开——注意**不要只复制到 `#` 之前**，`#` 后面那一段是访问令牌，丢了就打不开控制台。

程序只监听 `127.0.0.1`，局域网里其他机器访问不到。保持这个终端窗口开着；关掉窗口即停止服务。

---

## 3. 接入平台

### 方式一：在控制台里点

控制台列出全部内建平台。找到目标平台，打开开关。卡片会实时显示状态与真实挂载路径。

### 方式二：命令行

```powershell
"%USERPROFILE%\fenjue\fenjue-agent.exe" enable wb     # 接入
"%USERPROFILE%\fenjue\fenjue-agent.exe" disable wb    # 软关闭（默认）
```

平台标识见 [README](../README.md#内建支持的平台)。

---

## 4. 验证

```powershell
"%USERPROFILE%\fenjue\fenjue-agent.exe" verify
```

逐条输出每个平台、每条挂载、每个注入点的状态。状态含义：

| 状态 | 含义 | 处置 |
|---|---|---|
| `OK` | 链接存在、指向预期目标、且目标可达 | 无需处理 |
| `MISSING` | 挂载点不存在 | 该平台尚未接入，或在控制台里开启即可 |
| `MISMATCH` | 挂载点存在但不是链接，或指向了别处 | 说明该路径被一个普通目录占用，先确认里面没有你要留的东西，再重建 |
| `BROKEN` | 是链接，但目标不可达 | 目标盘未挂载或目标目录被删，先恢复目标再重跑 |

`verify` 不需要 `--platforms` 参数也能跑，适合放进计划任务做例行体检。

---

## 5. 关闭与还原

- **软关闭（默认）**：保留链接，只停用注入。随时可原样恢复，不丢失任何东西。
- **还原到操作前**：任何写操作都先在 `%USERPROFILE%\.fenjue\trash\<备份号>\` 下留备份。控制台的备份列表可一键还原；命令行方式则需要使用接口回传的备份号。

---

## 6. 排错

| 现象 | 原因与处置 |
|---|---|
| 端口被占用，启动即退出 | 换端口：`serve --port 7800`。注意换了端口，握手链接里的端口也要跟着改 |
| 浏览器打开握手链接显示未授权 | `#` 后面的令牌丢了。回终端重新整条复制 |
| 控制台一直显示「未检测到本地程序」 | 服务没在跑，或端口不一致。看终端窗口还在不在 |
| 页面由公网域名打开、连不上本地 | 需要给程序加来源白名单：`serve --origin https://你的域名`。可重复传多次 |
| 某个平台显示 `MISMATCH` | 该挂载点被普通目录占了。确认目录内容后可删除重建，或改用不同的记忆库/技能库根 |
| 某个平台显示 `BROKEN` | 链接目标不可达。常见于目标在外接盘或另一分区、该盘未挂载 |
| `verify` 报 Hermes 端相关异常 | 该端依赖环境变量 `HERMES_HOME`。变量缺失时程序会显式告警而不是静默跳过；请先设好该变量 |

---

## 7. 自行构建（可选）

```powershell
cd <repo>\web
npm ci
npm run build
cd ..\agent
pwsh .\build.ps1
```

**顺序不可颠倒**：Go 侧内嵌前端产物，前端必须先构建。

若 `go` 不在 `PATH` 上，`build.ps1` 会回退到默认安装路径。你的安装位置若不同，请把绝对路径写进脚本或加进 `PATH`。

产物为 `agent\fenjue-agent.exe`。
