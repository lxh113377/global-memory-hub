# 部署手册（DEPLOY）

官网为纯静态双线部署：GitHub Pages（主，`platforms.json` 的 `site.primary`）+ Cloudflare Pages（镜像，`site.mirror`）。两线内容完全相同，互相独立，任何一线可用即服务可用。

## GitHub Pages（主站）

1. 仓库 **Settings → Pages → Build and deployment → Source 选 `GitHub Actions`**（一次性开关，需仓库管理员在网页上操作；不打开则 `pages.yml` 的部署步会失败）。
2. 之后每次 push 到 `main`，`.github/workflows/pages.yml` 自动：构建控制台 → 组装（落地页在 `/`、控制台在 `/console/`）→ 部署。
3. 线上地址：`https://lxh113377.github.io/global-memory-hub/`（与 `platforms.json` 的 `site.primary` 一致；agent 的 Origin 白名单已默认放行该域名）。

## Cloudflare Pages（镜像）

> **2026-10-14 实测校正**：本项目**不是** Git-connected。`wrangler pages project list` 回
> `global-memory-hub … Git Provider = No` ⇒ **push 到 `main` 不会自动重部署**，必须手工执行下面的步骤。
> 下面第 ① 步（Connect to Git）**从未执行过**，原文属漂移，已改写为实况。

① 认证：`wrangler whoami`（本机已持 OAuth 令牌且权限含 `pages (write)`）。

② 按 CI 同法本地独立构建并组装产物（GitHub Pages 侧 `pages.yml` 用的是同一套命令）：

```bash
cd web && npm ci && npm run build
mkdir -p site-dist/console
cp -r site/. site-dist/
cp -r web/dist/. site-dist/console/
```

③ 首次建项目（已建过可跳过）：

```bash
wrangler pages project create global-memory-hub --production-branch main
```

④ 投放：

```bash
cd site-dist && wrangler pages deploy --project-name global-memory-hub
```

⑤ **验收必须打「不存在的路径」**，不能只看首页 200 —— `site/_routes.json` 把 `not_found_handling`
设成 `404-page`，所以不存在的路径应回**真 404**；若仍回 `200` + 根页体积，说明 `_routes.json` 没被带上
（症状与修复前完全相同，最容易被误读成「已生效」）。

> ⚠ **踩过的坑：`not_found_handling: "404-page"` 必须同时存在 `404.html` 才生效。**
> 本轮第一次只加 `_routes.json` 就重投，探针显示**行为与修复前逐字节相同**（不存在的路径仍回
> `200` + 13741 B 根页）——即该配置在缺 `404.html` 时被**静默忽略**，没有任何报错或警告。
> 判别方法：投放后立刻探一个**保证不存在**的路径（例 `/totally/bogus/path`），
> 返回 200 就是没生效，别去怀疑 CDN 缓存。另注：`_routes.json` 与 `404.html` 都**不会**作为资产出现在
> 投放回显的件数里（`_routes.json` 是构建配置），所以「Uploaded N files」少于本地产物件数是正常的。

线上地址：`https://global-memory-hub.pages.dev/`（与 `platforms.json` 的 `site.mirror` 一致）。

### 为什么镜像要单独维护（`site/_routes.json` 的由来）

Cloudflare Pages 在**没有** `_routes.json` 时默认 `not_found_handling: none`，即**把任何不存在的路径回成根
`index.html` 且状态码 200**。后果：一条写错的资产 URL、一个过期 hash，都会拿到 200 —— 「URL 返 200」在本站
**不构成存在性证明**，依赖它做部署核验会一直绿。

本仓控制台是**纯 hash 路由**（`web/src` 内 `pushState` / `history.` 命中 0 次，`vite.config.ts` 用 `base: "./"`），
不依赖 history-API 深链 ⇒ 改成真 404 **没有功能代价**。该文件只对 Cloudflare 生效；GitHub Pages 不消费它，
只会把它当普通静态文件吐出（惰性无害），而它同样会进入 GitHub Pages 的产物根。

## 发布二进制（Releases）

push 形如 `v0.1.0` 的 tag 即触发 `.github/workflows/release.yml`：五目标交叉编译（windows/amd64、linux/amd64+arm64、darwin/amd64+arm64）自动传 GitHub Release。INSTALL 文档与官网下载按钮都指向 `releases/latest`。

### 校验下载到的产物

每个 release 附带一个 `SHA256SUMS`，列出该次发布的全部 5 个产物。下载后先校验再运行：

```bash
# Linux / macOS
sha256sum -c SHA256SUMS

# Windows PowerShell（无 sha256sum 时）
Get-FileHash .\fenjue-agent-windows-amd64.exe -Algorithm SHA256
# 再与 SHA256SUMS 里对应那一行比对
```

`SHA256SUMS` 在合并所有构建产物**之后**生成（`merge-multiple` 把五个 job 的产物放进同一个 `dist/`），并在创建 release 前核对条目数与文件数一致 —— 少一个文件就红，而不是发一个缺项的校验文件出去。

> 二进制**签名**仍不在此列：需要付费代码签名证书，属未完成项（见 ROADMAP）。校验和能证明下载没被改动，不能证明它来自我们。

## 本地自测部署效果

```bash
cd web && npm ci && npm run build
mkdir -p /tmp/site-dist/console
cp -r site/. /tmp/site-dist/
cp -r web/dist/. /tmp/site-dist/console/
cd /tmp/site-dist && python3 -m http.server 8080   # 浏览器打开 http://127.0.0.1:8080
```
