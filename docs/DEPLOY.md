# 部署手册（DEPLOY）

官网为纯静态双线部署：GitHub Pages（主，`platforms.json` 的 `site.primary`）+ Cloudflare Pages（镜像，`site.mirror`）。两线内容完全相同，互相独立，任何一线可用即服务可用。

## GitHub Pages（主站）

1. 仓库 **Settings → Pages → Build and deployment → Source 选 `GitHub Actions`**（一次性开关，需仓库管理员在网页上操作；不打开则 `pages.yml` 的部署步会失败）。
2. 之后每次 push 到 `main`，`.github/workflows/pages.yml` 自动：构建控制台 → 组装（落地页在 `/`、控制台在 `/console/`）→ 部署。
3. 线上地址：`https://lxh113377.github.io/global-memory-hub/`（与 `platforms.json` 的 `site.primary` 一致；agent 的 Origin 白名单已默认放行该域名）。

## Cloudflare Pages（镜像）

1. 登录 Cloudflare Dashboard → Workers & Pages → Create → Pages → Connect to Git，选本仓库 `main` 分支。
2. 构建配置：
   - Build command: `cd web && npm ci && npm run build && mkdir -p site-dist/console && cp -r site/. site-dist/ && cp -r web/dist/. site-dist/console/`
   - Build output directory: `site-dist`
3. 项目名取 `global-memory-hub`（默认域名即 `global-memory-hub.pages.dev`，与 `platforms.json` 的 `site.mirror` 一致）。
4. 首次部署完成后无需再管，push 到 `main` 自动重部署。

## 发布二进制（Releases）

push 形如 `v0.1.0` 的 tag 即触发 `.github/workflows/release.yml`：五目标交叉编译（windows/amd64、linux/amd64+arm64、darwin/amd64+arm64）自动传 GitHub Release。INSTALL 文档与官网下载按钮都指向 `releases/latest`。

## 本地自测部署效果

```bash
cd web && npm ci && npm run build
mkdir -p /tmp/site-dist/console
cp -r site/. /tmp/site-dist/
cp -r web/dist/. /tmp/site-dist/console/
cd /tmp/site-dist && python3 -m http.server 8080   # 浏览器打开 http://127.0.0.1:8080
```
