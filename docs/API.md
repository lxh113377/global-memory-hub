# HTTP API · 2026-10-02

> 本文件是 `fenjue-agent serve` 暴露的**全部** HTTP 接口。内容以源码为唯一真相源提取
> （`agent/internal/server/{router,skills,roots,middleware}.go` 与 `agent/internal/state/`），
> 不含任何未实现的接口。
>
> **先读这一段再发请求**：服务只监听 `127.0.0.1`，且每个非 `/api/health` 的接口都要令牌。
> 四道安全闸中有一道（Origin 白名单）只对**带 Origin 头的请求**生效 —— 这一点对命令行客户端
> 很重要，见 §2。

---

## 1. 基本约定

| 项 | 值 |
|---|---|
| 基址 | `http://127.0.0.1:7799`（端口由 `--port` 改；**只绑定回环**，不监听 `0.0.0.0`） |
| 鉴权头 | `X-Fenjue-Token: <本次启动的令牌>` —— **不是** `Authorization` |
| 令牌来源 | 启动日志里的握手链接 `http://127.0.0.1:7799/console#token=<hex>`，URL 片段不进请求 |
| 令牌文件 | `~/.fenjue/token`（每次启动重新生成，权限 0600 / Windows 上另加 ACL） |
| 请求体上限 | 1 MiB（`io.LimitReader(r.Body, 1<<20)`）；空体与纯空白体按零值处理 |
| 响应类型 | 成功与失败**都是** `application/json; charset=utf-8` |
| 错误体 | `{"ok": false, "error": "<可读原因>"}` |
| 操作结果体 | `{"ok": true, "backupId": "<备份号或空>", "changes": ["<逐条变更说明>"]}` |
| 版本字段 | `/api/health` 的 `version` 即 `server.Version` 常量，与 git tag 同步 |

`enable` / `disable` / `restore` / `sync` 都返回同一个 `OpResult` 结构（`state.OpResult`），
**`changes` 是给人看的变更清单**，要判断成功看 `ok`；`backupId` 用于后续 `restore`。

---

## 2. 四道安全闸对调用方的实际影响

| 闸 | 触发条件 | 调用方会看到 |
|---|---|---|
| **Host 头校验** | 所有请求 | `403 {"error":"forbidden Host header ...; use 127.0.0.1:7799 or localhost:7799"}`。用 `--port` 改了端口后，Host 里的端口必须同步改，否则全站 403 |
| **Origin 白名单 + CORS** | **仅当请求带 `Origin` 头** | 非白名单来源 `403 forbidden Origin ...; add it via --origin`。白名单来源会收到 `Access-Control-Allow-Origin` / `-Headers: X-Fenjue-Token, Content-Type` / `-Methods: GET, POST, OPTIONS` |
| **OPTIONS 预检** | `OPTIONS` 请求 | 直接 `204`，**在令牌校验之前**。因此预检永远不会被 401 卡住 |
| **令牌校验** | 路径以 `/api/` 开头且**不等于** `/api/health` | `401` + `WWW-Authenticate: X-Fenjue-Token`。静态页 `/` 与 `/console` 免令牌（握手链接要把片段里的令牌交给前端，因此页面本身必须可达） |

**由此得出两条给自动化工具的结论**：

1. **命令行客户端不要伪造 `Origin` 头**。不带 `Origin` 就绕过白名单这一闸（这是本机回环服务的正常形态，
   不是绕过漏洞：没有 Origin 的请求无法被浏览器发起）。带上反而可能被 403。
2. **`--port` 改了端口，curl 也要用同一个端口**。`Host` 头由 curl 按 URL 自动生成，
   但如果你手写 `-H "Host: ..."`，必须写成 `127.0.0.1:<port>` 或 `localhost:<port>`。

比较令牌用 `crypto/subtle.ConstantTimeCompare`，避免时序侧信道。

---

## 3. 端点

### 3.1 `GET /api/health` —— 免令牌

唯一不需要令牌的接口，用作「服务是否就绪」的探针。

```bash
curl -s http://127.0.0.1:7799/api/health
```

```json
{ "ok": true, "version": "0.2.2", "os": "windows", "token_required": true }
```

| 字段 | 说明 |
|---|---|
| `version` | 与 git tag 同步的产品版本（由 `check_version_sync.py` 判据保证不漂移） |
| `os` | Go 的 `runtime.GOOS` |
| `token_required` | 恒为 `true`，提示其余接口需要令牌 |

`POST` 等其他方法 → `405 {"error":"method not allowed; use GET"}`。

---

### 3.2 `GET /api/state` —— 需令牌

全量探活快照（`state.BuildSnapshot`）。**这是本项目对应「统一库现状视图」的功能模块**：
一份共享库被多少端接入、每端挂载点是否健康、注入块状态，全部一次返回。

```bash
curl -s -H "X-Fenjue-Token: $TOKEN" http://127.0.0.1:7799/api/state
```

```json
{
  "roots": { "memory": "/home/you/.fenjue/memory", "skills": "/home/you/.fenjue/skills" },
  "platforms": [
    {
      "id": "hm", "label": "Hermes Agent", "support": "ga", "status": "OK",
      "mounts": [
        { "from": "...", "to": "...", "kind": "link",     "status": "OK" },
        { "from": "...", "to": "...", "kind": "per-skill","status": "OK" }
      ],
      "inject": [ { "path": "...", "status": "OK" } ]
    }
  ]
}
```

`status` 是**四态 + SKIP**（`state/verify.go`），不是布尔：

| 值 | 含义 |
|---|---|
| `OK` | 正常 |
| `MISMATCH` | 挂载点存在但类型或目标对不上（例如本该是链接却成了真实目录） |
| `BROKEN` | 链接在，但目标不可达 |
| `MISSING` | 挂载点不存在 |
| `SKIP` | 该端在本机不可用（home 不存在或平台不支持当前系统），**不计入异常** |

---

### 3.3 `POST /api/verify` —— 需令牌

把 `state` 的四态汇总成计数（`state.Verify`）。**注意它只统计，不动作**：不会尝试修复。

```bash
curl -s -X POST -H "X-Fenjue-Token: $TOKEN" http://127.0.0.1:7799/api/verify
```

```json
{ "total": 27, "ok": 27, "broken": [], "mismatch": [], "missing": [] }
```

| 字段 | 说明 |
|---|---|
| `total` | 参与统计的条目数（`SKIP` 不计入） |
| `ok` | 正常数 |
| `broken` / `mismatch` / `missing` | 异常条目，每条是 `平台id: from -> to (kind)` 或 `平台id: inject <path>` |

**复杂度**：`O(平台数 + 技能条数)` 次系统调用。`per-skill` 挂载会 `ReadDir` 技能根并逐条探活，
`memory` 根只被 `stat` 一次 ⇒ **校验成本不随记忆库文件数增长**。实测数据见
[PERFORMANCE.md](PERFORMANCE.md) §4.4。用 `GET` 调本接口 → `405`。

---

### 3.4 `GET /api/skills` —— 需令牌

实时扫描技能根，读每个技能的 `SKILL.md` frontmatter 取 `name` / `description`。
控制台用它渲染技能列表。

```bash
curl -s -H "X-Fenjue-Token: $TOKEN" http://127.0.0.1:7799/api/skills
```

```json
{
  "ok": true,
  "roots": { "memory": "...", "skills": "..." },
  "skills": [ { "id": "brainstorming", "name": "brainstorming", "description": "..." } ],
  "total": 18,
  "note": ""
}
```

| 字段 | 说明 |
|---|---|
| `skills[].id` | 目录名（稳定标识） |
| `skills[].name` / `description` | 来自 `SKILL.md` 的 YAML frontmatter，去掉外层引号；**缺 `name` 时回退成目录名** |
| `note` | 技能根不可达时的原因（正常为空字符串）。**这不是错误**：HTTP 仍是 200，`total` 为 0 |

只有目录、且含 `SKILL.md` 的条目会被列出；`SKILL.md` 缺失或不可读的条目被静默跳过。
复杂度 `O(技能条数)`，每次调用都重新扫盘，没有缓存。

---

### 3.5 `POST /api/roots` —— 需令牌

持久化自定义库根到 `~/.fenjue/state/roots.json`。

```bash
curl -s -X POST -H "X-Fenjue-Token: $TOKEN" -H "Content-Type: application/json" \
  -d '{"memory":"~/my-memory","skills":"~/my-skills"}' \
  http://127.0.0.1:7799/api/roots
```

```json
{ "ok": true, "applied": false, "note": "saved; restart fenjue-agent to take effect",
  "roots": { "memory": "/home/you/my-memory", "skills": "/home/you/my-skills" } }
```

请求体字段 `memory` / `skills`，**至少要给一个**，否则 `400`。路径支持 `~` / `%VAR%` /
`$VAR` 展开；含 Windows 保留设备名（`CON`、`NUL` 等）或展开后为空 → `400`。

> **`applied` 恒为 `false`**：保存即返回，但运行中的配置已解析完毕，**必须重启 `fenjue-agent` 才生效**。
> 重启后 `platform.loadRootsOverride()` 读取该文件；文件缺失或损坏时**静默回落**到 `platforms.json`
> 的默认库根（不报错）。

---

### 3.6 `POST /api/platforms/{id}/{action}` —— 需令牌

四个动作共用一个路径形状，`{action}` 为 `enable` / `disable` / `restore` / `sync`。
路径形状不对 → `404 expected /api/platforms/{id}/{enable|disable|restore}`；
动作不认识 → `404 unknown action <x>; use enable|disable|restore|sync`。
非 `POST` → `405`。**所有动作都先写备份再改动**（`restore` 除外，它本身就是回滚）。

#### 3.6.1 `enable`

```bash
curl -s -X POST -H "X-Fenjue-Token: $TOKEN" -H "Content-Type: application/json" \
  -d '{}' http://127.0.0.1:7799/api/platforms/hm/enable
```

请求体可选字段：

| 字段 | 说明 |
|---|---|
| `roots` | `{"memory": "...", "skills": "..."}`，**仅本次调用生效**的库根覆盖，不落盘（落盘请用 `POST /api/roots`） |

行为：建端内 home 目录 → 引导统一库（目录不存在则建、空库播种，已有库**零覆盖**）→ 写备份 →
按挂载类型建链/同步/批量建技能链接 → 写注入激活标记块。**幂等**：已 OK 的挂载点走 no-op，
不会破坏已有链接。未知 `id` → `404`。

#### 3.6.2 `disable`

```bash
curl -s -X POST -H "X-Fenjue-Token: $TOKEN" -H "Content-Type: application/json" \
  -d '{"soft": true}' http://127.0.0.1:7799/api/platforms/hm/disable
```

| 字段 | 默认 | 说明 |
|---|---|---|
| `soft` | `true` | `true` = 摘链接 + 注入块改写为停用短壳（可一键恢复）；`false` = 摘链接 + 删除注入块（彻底） |

`mirror` 类型在停用时只写「软关闭标记」并停止同步，**不删除已同步过去的文件**。

#### 3.6.3 `restore`

```bash
curl -s -X POST -H "X-Fenjue-Token: $TOKEN" -H "Content-Type: application/json" \
  -d '{"backupId":"20261002-153000-ab12cd"}' http://127.0.0.1:7799/api/platforms/hm/restore
```

请求体 `backupId` **必填**，空或缺失 → `400 backupId is required in body`。
备份号来自 `enable` / `disable` 响应的 `backupId`，备份在 `~/.fenjue/trash/<备份号>/`。
还原是整文件级回滚：链接按记录的目标重建，文件/目录按备份内容恢复。

#### 3.6.4 `sync`

```bash
curl -s -X POST -H "X-Fenjue-Token: $TOKEN" http://127.0.0.1:7799/api/platforms/cx/sync
```

手动触发 `mirror` 型挂载的单向同步（`state.SyncMirror`）。**无请求体**（请求体会被忽略）。
该端没有任何 `mirror` 挂载时 → `500 no mirror mounts`（这是唯一用 500 表达的业务分支，
因为它是状态冲突而非请求错误）。

---

### 3.7 静态页与前端资源 —— 免令牌

| 路径 | 返回 |
|---|---|
| `/` 与 `/console` | 内嵌前端的 `index.html` |
| 其他路径 | 按静态文件返回；未命中回落到 `index.html`（前端路由） |

`Content-Type` 按扩展名判定（`.html` / `.js` / `.css` / `.svg` / `.json`，其余 `application/octet-stream`）。
若二进制里没有前端产物（未跑 `npm run build` 构建）→ `404 frontend dist missing; run build.ps1`。

**这些路径免令牌是设计使然**：握手链接把令牌放在 URL 片段里交给前端，页面本身必须先可达。
令牌不会随静态请求上到任何地方（片段不会随 HTTP 请求发送）。

---

## 4. 状态码总表

| 码 | 触发 | 体 |
|---|---|---|
| `200` | 成功 | 见各端点 |
| `204` | `OPTIONS` 预检（令牌校验之前） | 空 |
| `400` | JSON 非法 / `backupId` 缺失 / 库根非法 / `roots` 两个都没给 | `{"ok":false,"error":...}` |
| `401` | 缺令牌或令牌不对（仅 `/api/*` 除 `/api/health`） | `{"ok":false,"error":...}` + `WWW-Authenticate` |
| `403` | Host 头不是回环，或 Origin 不在白名单 | `{"ok":false,"error":...}` |
| `404` | 未知 api 路径 / 路径形状错 / 未知动作 / 未知平台 id / 前端产物缺失 | `{"ok":false,"error":...}` |
| `405` | 方法不对（错误体会提示正确方法） | `{"ok":false,"error":...}` |
| `500` | 操作失败（备份失败、链接失败、无 mirror 挂载）、内部 panic | `{"ok":false,"error":...}` |

panic 会被中间件兜住并返回 `500 internal panic; see agent log`，细节只进 `~/.fenjue/logs/agent.log`。

---

## 5. 调用示例（完整流程）

```bash
# 1) 起服务并从日志里取令牌
fenjue-agent serve --port 7799
#    日志里会打印: Handshake: http://127.0.0.1:7799/console#token=<hex>
TOKEN=<hex>

# 2) 免令牌探活
curl -s http://127.0.0.1:7799/api/health

# 3) 带令牌查状态与技能（命令行客户端不要加 Origin 头）
curl -s -H "X-Fenjue-Token: $TOKEN" http://127.0.0.1:7799/api/state
curl -s -H "X-Fenjue-Token: $TOKEN" http://127.0.0.1:7799/api/skills

# 4) 启用一端并记下 backupId 以便回滚
curl -s -X POST -H "X-Fenjue-Token: $TOKEN" -d '{}' \
  http://127.0.0.1:7799/api/platforms/hm/enable

# 5) 全量校验
curl -s -X POST -H "X-Fenjue-Token: $TOKEN" http://127.0.0.1:7799/api/verify

# 6) 软关闭 / 从备份还原
curl -s -X POST -H "X-Fenjue-Token: $TOKEN" -d '{"soft":true}' \
  http://127.0.0.1:7799/api/platforms/hm/disable
curl -s -X POST -H "X-Fenjue-Token: $TOKEN" -d '{"backupId":"<上一步的 backupId>"}' \
  http://127.0.0.1:7799/api/platforms/hm/restore
```

---

## 6. 本文件未覆盖的内容（如实声明）

| 项 | 状态 |
|---|---|
| OpenAPI / Swagger 文件 | **不存在**。本文件是当前唯一的人工 API 描述 |
| WebSocket / SSE | 不存在 |
| 并发与限流 | **无**。没有请求频率限制，也没有并发上限声明；本轮未测并发 QPS（见 [PERFORMANCE.md](PERFORMANCE.md) §5） |
| 版本化 API（`/api/v1/...`） | 不存在。路径不带版本，字段若有变更靠程序版本对齐 |
| `sync` 的请求体 | 实现上**不解析**请求体，传什么都不报错 |
| 认证方式扩展 | 只有 `X-Fenjue-Token` 一种。不接受 `Authorization`，也不接受 query 参数传令牌 |