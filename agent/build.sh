#!/usr/bin/env bash
# build.sh - mac/linux 构建入口 (对齐 agent/build.ps1): 前端 -> embed -> go build。
# 用法: cd agent && ./build.sh   (可选: GO=/path/to/go ./build.sh)
set -euo pipefail
cd "$(dirname "$0")"

# 定位 go: PATH -> GO 环境变量 -> 常见安装路径, 不假设 PATH 里有 go。
GO_BIN="${GO:-}"
if [ -z "$GO_BIN" ]; then
  if command -v go >/dev/null 2>&1; then
    GO_BIN="go"
  else
    for c in /usr/local/go/bin/go /opt/homebrew/bin/go "$HOME/go/bin/go" /usr/lib/go/bin/go "/c/Program Files/Go/bin/go.exe"; do
      if [ -x "$c" ]; then GO_BIN="$c"; break; fi
    done
  fi
fi
if [ -z "$GO_BIN" ]; then
  echo "error: go not found; install Go 1.27+ or set GO=/path/to/go" >&2
  exit 1
fi

# 前端产物缺失时现建 (产物不入库, embed 需要真 dist)。
if [ ! -f ../web/dist/index.html ]; then
  echo "[build] web dist missing, building frontend..."
  (cd ../web && npm ci && npm run build)
fi

mkdir -p cmd/fenjue-agent/dist
cp -r ../web/dist/. cmd/fenjue-agent/dist/

echo "[build] using go: $("$GO_BIN" version)"
# CGO_ENABLED=0: 静态链接, linux 产物在 alpine/musl 也能跑 (docker 实测 2026-10-02)
CGO_ENABLED=0 "$GO_BIN" build -trimpath -ldflags "-s -w" -o fenjue-agent ./cmd/fenjue-agent
echo "[build] done: agent/fenjue-agent"
