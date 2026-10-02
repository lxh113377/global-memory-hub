#!/usr/bin/env python3
"""check_api_contract.py - API 文档与实现的一致性判据。

背景
----
docs/API.md 是人工写的接口描述, router.go 是路由注册表。人工文档天然会漂移:
改了路由忘了改文档, 或者文档写了从未实现的端点, 读文档的人按它发请求只会拿到 404。
本脚本把这件事变成 CI 里的一条腿。

判据分两层 (默认只跑第一层, 第二层要显式 --live)
------------------------------------------------
1. **静态对账 (双向, 默认)**: 解析 docs/API.md 里列出的端点, 与 router.go 的
   ``mux.HandleFunc`` 注册表比对。少写或多写都判为漂移。纯静态, 不需要网络与进程,
   因此可以接进阻断链。
2. **实测状态码 (--live)**: 在仓内沙箱起一个真的 serve (假平台假库根, 子进程带重定向的
   USERPROFILE/HOME), 逐个端点验证文档承诺的状态码, 重点验三类:
   免令牌的 /api/health、其余接口缺令牌必 401、非白名单 Origin 必 403。
   需要 go build, 因此 CI 侧以 advisory 腿接入。

为什么值得单独写一个脚本: 这类"文档与实现一致性"是最容易悄悄坏掉的东西之一,
而它坏掉时不会有任何测试变红。

退出码: 0 一致 / 1 发现漂移或实测不符 / 2 无法测量 (UNVERIFIED)
"""
from __future__ import annotations

import argparse
import json
import os
import re
import subprocess
import sys
import time
import urllib.error
import urllib.request
from pathlib import Path

REPO = Path(__file__).resolve().parent.parent
ROUTER_GO = REPO / "agent" / "internal" / "server" / "router.go"
API_DOC = REPO / "docs" / "API.md"
SANDBOX_ROOT = REPO / "_temp" / "apicheck"

EXIT_OK = 0
EXIT_DIFF = 1
EXIT_UNAVAILABLE = 2

# 静态页与前端资源不算 API 端点 (由 handleStatic 统一兜底);
# /api/ 是"未知 api 路径"的 404 兜底, 也不是端点。
# 注意: HANDLE_FUNC_RE 捕获的是不含引号的路径, 所以这里不能带引号。
STATIC_PREFIXES = ("/", "/api/")
HANDLE_FUNC_RE = re.compile(r'mux\.HandleFunc\("([^"]+)"')


def routes_from_router(path: Path) -> tuple[list[str], list[str]]:
    """从 router.go 提取路由。

    分两类返回, 因为 Go 的 ServeMux 语义不同:
    - exact   : 不以斜杠结尾, 精确匹配一个路径 (/api/health)
    - subtree : 以斜杠结尾, 匹配整个前缀 (/api/platforms/ 覆盖该前缀下所有路径)
    """
    try:
        src = path.read_text(encoding="utf-8")
    except OSError:
        return [], []
    exact: set[str] = set()
    subtrees: set[str] = set()
    for m in HANDLE_FUNC_RE.finditer(src):
        route = m.group(1)
        if route in STATIC_PREFIXES:
            continue
        if route.endswith("/"):
            subtrees.add(route)
        else:
            exact.add(route)
    return sorted(exact), sorted(subtrees)


def normalize_endpoint(path: str) -> str:
    """把子树端点的各种书写形态归一。

    router.go 对平台操作只注册一个子树 ``/api/platforms/``（结尾斜杠, 子树匹配）,
    而文档里会出现四种写法: ``/api/platforms/{id}/{action}``、
    ``/api/platforms/{id}/{enable|disable|restore}``、具体示例
    ``/api/platforms/hm/enable``、以及泛指的 ``/api/platforms/*``。
    不归一就会把"文档写了示例"误判成"文档写了一个不存在的独立端点"。
    """
    if path.startswith("/api/platforms/"):
        return "/api/platforms/*"
    return path


def endpoints_from_doc(path: Path) -> list[str]:
    """从 docs/API.md 提取文档声明的端点路径。

    只认 HTTP 方法加路径的写法 (GET/POST 紧跟 /api/... 出现在行内代码或标题里),
    避免把正文里提到的片段误当成一个独立端点。
    """
    try:
        text = path.read_text(encoding="utf-8")
    except OSError:
        return []
    found: set[str] = set()
    for m in re.finditer(r"`?\b(GET|POST)\s+(/api/[A-Za-z0-9_\-/{}.|]+)`?", text):
        found.add(normalize_endpoint(m.group(2)))
    return sorted(found)


def reconcile(routes: list[str], documented: list[str], subtrees: list[str] = ()) -> list[str]:
    """双向对账。返回问题列表, 空列表表示一致。"""
    problems: list[str] = []
    rset, dset = set(routes), set(documented)

    # 1) 每个精确路由都必须在文档里出现。
    for r in sorted(rset):
        hit = r in dset or any(r.startswith(d.rstrip("*")) for d in dset if d.endswith("*"))
        if not hit:
            problems.append("route %s is registered in router.go but docs/API.md never documents it" % r)

    # 2) 每棵子树也必须有文档覆盖 (用前缀形式的文档条目即可)。
    for s in subtrees:
        hit = any(d.endswith("*") and s.startswith(d.rstrip("*")) for d in dset)
        if not hit:
            problems.append("subtree %s is registered in router.go but docs/API.md does not cover it" % s)

    # 3) 每条文档不得指向不存在的实现。
    for d in sorted(dset):
        base = d.rstrip("*")
        if base in rset or d in rset:
            continue
        if any(base.startswith(s) for s in subtrees):
            continue
        problems.append("docs/API.md documents %s but no matching route is registered" % d)
    return problems


def selftest() -> int:
    """对账函数必须真的会咬人: 少写、多写、模板写法都要能判出问题。"""
    failures: list[str] = []

    clean = reconcile(
        ["/api/health", "/api/state"],
        ["/api/health", "/api/state"],
    )
    if clean:
        failures.append("identical input must produce no problem, got: %r" % clean)

    undocumented = reconcile(
        ["/api/health", "/api/state", "/api/skills"],
        ["/api/health", "/api/state"],
    )
    if len(undocumented) != 1 or "/api/skills" not in undocumented[0]:
        failures.append("a route missing from the doc must be reported, got: %r" % undocumented)

    phantom = reconcile(
        ["/api/health"],
        ["/api/health", "/api/ghost"],
    )
    if len(phantom) != 1 or "/api/ghost" not in phantom[0]:
        failures.append("a documented endpoint with no route must be reported, got: %r" % phantom)

    # 子树注册的模板写法必须能覆盖文档里的具体示例路径 (router 注册子树, 文档写泛指形式)。
    templated = reconcile(
        ["/api/platforms/hm/enable"],
        ["/api/platforms/*"],
        subtrees=["/api/platforms/"],
    )
    if templated:
        failures.append("template form /api/platforms/* must cover a concrete example path, got: %r" % templated)

    # 子树路由本身必须被文档覆盖 (否则 /api/platforms/* 全家都不进对账, 变成盲区)。
    uncovered = reconcile(["/api/health"], ["/api/health"], subtrees=["/api/platforms/"])
    if len(uncovered) != 1 or "subtree" not in uncovered[0]:
        failures.append("an uncovered subtree route must be reported, got: %r" % uncovered)

    # 子树已被覆盖时不该报。
    covered_subtree = reconcile(["/api/health"], ["/api/health", "/api/platforms/*"], subtrees=["/api/platforms/"])
    if covered_subtree:
        failures.append("a covered subtree route must not be reported, got: %r" % covered_subtree)

    if failures:
        for f in failures:
            print("SELFTEST-FAIL %s" % f, file=sys.stderr)
        print("[GATE:api-contract-selftest-red] %d case(s) failed" % len(failures), file=sys.stderr)
        return 1
    print("[GATE:api-contract-selftest-pass] missing route bites, phantom endpoint bites, template covers subtree")
    return 0


# ---------------------------------------------------------------- live 探测


def http_json(port: int, method: str, path: str, token: str | None = None,
              origin: str | None = None, host: str | None = None,
              body: dict | None = None) -> tuple[int, dict]:
    url = "http://127.0.0.1:%d%s" % (port, path)
    data = json.dumps(body).encode("utf-8") if body is not None else None
    req = urllib.request.Request(url, data=data, method=method)
    if token:
        req.add_header("X-Fenjue-Token", token)
    if origin:
        req.add_header("Origin", origin)
    if host:
        req.add_header("Host", host)
    try:
        with urllib.request.urlopen(req, timeout=10) as resp:
            raw = resp.read().decode("utf-8", "replace")
            try:
                return resp.status, json.loads(raw)
            except json.JSONDecodeError:
                return resp.status, {"raw": raw[:200]}
    except urllib.error.HTTPError as exc:
        raw = exc.read().decode("utf-8", "replace")
        try:
            return exc.code, json.loads(raw)
        except json.JSONDecodeError:
            return exc.code, {"raw": raw[:200]}
    except (urllib.error.URLError, OSError) as exc:
        return 0, {"error": "transport: %s" % exc}


def live_probe() -> tuple[list[str], list[str]]:
    """起沙箱 serve 实测状态码。返回 (problems, unverified)。"""
    sys.path.insert(0, str(REPO / "scripts"))
    problems: list[str] = []
    unverified: list[str] = []

    try:
        import bench_scale as bs  # noqa: PLC0415 - 复用同一套沙箱构造, 不重复实现
    except Exception as exc:  # noqa: BLE001
        return [], ["bench_scale.py not importable: %s" % exc]

    SANDBOX_ROOT.mkdir(parents=True, exist_ok=True)
    sandbox = SANDBOX_ROOT / ("run-%d" % int(time.time() * 1000))
    sandbox.mkdir(parents=True, exist_ok=True)

    binary = bs.SANDBOX_ROOT / ("fenjue-agent-bench%s" % (".exe" if os.name == "nt" else ""))
    if not binary.exists():
        bs.SANDBOX_ROOT.mkdir(parents=True, exist_ok=True)
        r = subprocess.run(["go", "build", "-o", str(binary), "./cmd/fenjue-agent"],
                           cwd=str(REPO / "agent"), capture_output=True, check=False)
        if r.returncode != 0:
            return [], ["go build failed: %s" % r.stderr.decode("utf-8", "replace")[-300:]]

    scale = {"name": "API", "platforms": 2, "skills": 3, "files": 10}
    try:
        cfg, fakehome, _skills_root, _n = bs.build_platforms(scale, sandbox)
    except Exception as exc:  # noqa: BLE001
        return [], ["sandbox build failed: %s" % exc]

    port = bs.pick_free_port()
    env = bs.child_env(fakehome)
    log = sandbox / "serve.log"
    fh = log.open("wb")
    proc = subprocess.Popen([str(binary), "serve", "--port", str(port), "--platforms", str(cfg)],
                            env=env, stdout=fh, stderr=subprocess.STDOUT)
    fh.close()

    try:
        ready, _ms = bs.wait_health(port, proc, log, timeout=60)
        if not ready:
            return [], ["serve never became healthy; see %s" % log]

        token = ""
        try:
            for line in log.read_bytes().decode("utf-8", "replace").splitlines():
                if "token=" in line:
                    token = line.split("token=", 1)[1].strip()
                    break
        except OSError:
            pass
        if not token:
            return [], ["could not read the token from serve output; token is required for the auth checks"]

        def call(method: str, path: str, tok: bool = True, origin: str | None = None,
                 host: str | None = None, body: dict | None = None) -> tuple[int, dict]:
            return http_json(port, method, path, token=token if tok else None,
                             origin=origin, host=host, body=body)

        checks: list[tuple[str, int, tuple]] = [
            ("GET /api/health needs no token", 200, ("GET", "/api/health", False)),
            ("GET /api/state without token is 401", 401, ("GET", "/api/state", False)),
            ("GET /api/state with token is 200", 200, ("GET", "/api/state", True)),
            ("GET /api/skills with token is 200", 200, ("GET", "/api/skills", True)),
            ("POST /api/verify with token is 200", 200, ("POST", "/api/verify", True)),
            ("GET /api/health with wrong method is 405", 405, ("POST", "/api/health", True)),
            ("GET /api/verify with wrong method is 405", 405, ("GET", "/api/verify", True)),
            ("POST /api/platforms/unknown-id/enable is 404", 404,
             ("POST", "/api/platforms/no-such-platform/enable", True)),
            ("POST /api/platforms with bad action is 404", 404,
             ("POST", "/api/platforms/bench000/bogus", True)),
            ("POST /api/platforms shape error is 404", 404,
             ("POST", "/api/platforms/bench000", True)),
            ("POST restore without backupId is 400", 400,
             ("POST", "/api/platforms/bench000/restore", True)),
            ("unknown api path is 404", 404, ("GET", "/api/ghost", True)),
        ]
        for label, want, (method, path, tok) in checks:
            got, payload = call(method, path, tok=tok, body={} if method == "POST" else None)
            if got != want:
                problems.append("%s: expected HTTP %d, got %d (%r)" % (label, want, got, payload))

        # Origin 白名单: 非白名单来源必 403。
        got, payload = call("GET", "/api/state", tok=True, origin="https://evil.example")
        if got != 403:
            problems.append("non-whitelisted Origin must be 403, got %d (%r)" % (got, payload))

        # Host 头校验: 非回环 Host 必 403。
        got, payload = call("GET", "/api/health", tok=False, host="attacker.example")
        if got != 403:
            problems.append("non-loopback Host header must be 403, got %d (%r)" % (got, payload))

        # 成功路径: enable 必须返回 ok=true 与非空 backupId。
        got, payload = call("POST", "/api/platforms/bench000/enable", tok=True, body={})
        if got != 200 or not payload.get("ok") or not payload.get("backupId"):
            problems.append("enable must return ok=true and a non-empty backupId, got %d (%r)" % (got, payload))
        else:
            bid = payload["backupId"]
            got2, payload2 = call("POST", "/api/platforms/bench000/restore", tok=True, body={"backupId": bid})
            if got2 != 200 or not payload2.get("ok"):
                problems.append("restore with the returned backupId must be 200 ok, got %d (%r)" % (got2, payload2))

        # /api/roots: 空体必 400 (文档承诺至少要给一个根)。
        got, payload = call("POST", "/api/roots", tok=True, body={})
        if got != 400:
            problems.append("POST /api/roots with neither memory nor skills must be 400, got %d (%r)" % (got, payload))
    finally:
        if proc.poll() is None:
            proc.terminate()
            try:
                proc.wait(timeout=10)
            except subprocess.TimeoutExpired:
                proc.kill()
        bs.cleanup(sandbox)
    return problems, unverified


def main() -> int:
    ap = argparse.ArgumentParser(description="Verify docs/API.md against router.go, and optionally against a live sandbox.")
    ap.add_argument("--json", action="store_true", help="emit machine-readable output")
    ap.add_argument("--selftest", action="store_true", help="run the reconciliation self-test and exit")
    ap.add_argument("--live", action="store_true", help="also boot a sandboxed serve and probe real status codes")
    args = ap.parse_args()

    if args.selftest:
        return selftest()

    if not ROUTER_GO.exists():
        print("cannot measure: %s not found (UNVERIFIED, never counted as a pass)" % ROUTER_GO, file=sys.stderr)
        print("[GATE:api-contract-unverified]", file=sys.stderr)
        return EXIT_UNAVAILABLE
    if not API_DOC.exists():
        print("cannot measure: %s not found (UNVERIFIED, never counted as a pass)" % API_DOC, file=sys.stderr)
        print("[GATE:api-contract-unverified]", file=sys.stderr)
        return EXIT_UNAVAILABLE

    routes, subtrees = routes_from_router(ROUTER_GO)
    documented = endpoints_from_doc(API_DOC)
    problems = reconcile(routes, documented, subtrees)
    unverified: list[str] = []

    live_problems: list[str] = []
    live_unverified: list[str] = []
    if args.live:
        live_problems, live_unverified = live_probe()
        problems.extend(live_problems)
        unverified.extend(live_unverified)

    payload = {
        "routes_in_router": routes,
        "documented": documented,
        "problems": problems,
        "unverified": unverified,
        "live": bool(args.live),
    }
    if args.json:
        print(json.dumps(payload, ensure_ascii=False, indent=2))
    else:
        print("router.go routes : %s" % ", ".join(routes))
        print("subtrees         : %s" % (", ".join(subtrees) if subtrees else "(none)"))
        print("documented       : %s" % ", ".join(documented))
        print("live probe       : %s"
              % ("executed, %d problem(s), %d unverified" % (len(live_problems), len(live_unverified))
                 if args.live else "SKIPPED (pass --live to boot a sandboxed serve and verify status codes)"))
        if problems:
            print("\nDRIFT:")
            for p in problems:
                print("  - %s" % p)
        else:
            print("\nAPI CONTRACT OK: every registered route is documented and every documented endpoint exists.")
        for u in unverified:
            print("UNVERIFIED (never counted as a pass): %s" % u)

    if problems:
        print("[GATE:api-contract-red]", file=sys.stderr)
        return EXIT_DIFF
    if unverified:
        print("[GATE:api-contract-unverified]", file=sys.stderr)
    return EXIT_OK


if __name__ == "__main__":
    sys.exit(main())