#!/usr/bin/env python3
"""bench_scale.py - 规模压测: 把性能维度从"没有数据"变成"可复算的数据"。

背景
----
对标报告 (docs/BENCHMARK-2026-10-02.md) 的维度 4 长期是空的: 报告在 §5.3 明确声明
"hub 未做规模压测, 本报告不做任何性能数字断言"。没有数字, 就无法回答三个问题:
挂载端数上去以后会不会卡? 技能库变大以后 verify 会不会变慢? 常驻内存能不能放进
一台小机器? 本脚本用一次可复算的实测回答它们。

沙箱隔离 (硬要求)
----------------
公开仓 + 私有记忆源是同一台机器。任何一次 enable 都会写备份到 ~/.fenjue/trash,
serve 每次启动都会改写 ~/.fenjue/token。若直接跑, 会污染用户真实的统一库与令牌。
因此本脚本:

1. 自造假的 platforms.json (假平台 id、假库根), 一切写入都在仓内 _temp/bench/ 下;
2. 给子进程注入 USERPROFILE 与 HOME (Go 侧 safeio.HomeDir 走 os.UserHomeDir, 即读这两个
   环境变量), 于是 hub 以为的 ~/.fenjue 落在沙箱里, 真实的 ~/.fenjue 一个字节都不碰;
3. 库根与平台 home 全部落在沙箱内, 结束后清理 (--keep 可保留)。

复杂度: 造库 O(files + skills), 挂载 O(platforms + skills) 次系统调用,
verify O(platforms + skills) 次 stat (与库文件数无关, 见下面 EXPLAIN)。

退出码: 0 至少一档跑完 / 1 全档失败或 UNVERIFIED 之外有硬错 / 2 无法测量 (UNVERIFIED)
"""
from __future__ import annotations

import argparse
import json
import os
import statistics
import subprocess
import sys
import tempfile
import time
import urllib.error
import urllib.request
from pathlib import Path

REPO = Path(__file__).resolve().parent.parent
AGENT_DIR = REPO / "agent"
SANDBOX_ROOT = REPO / "_temp" / "bench"

# 三档规模。platforms 只挂 1 条 link (单端成本), 技能数只喂给 1 个 per-skill 平台,
# 否则 200 平台 x 1000 技能 = 20 万条链接, 测的是磁盘而不是程序。
SCALES = [
    {"name": "S1", "platforms": 9, "skills": 50, "files": 500},
    {"name": "S2", "platforms": 50, "skills": 300, "files": 3000},
    {"name": "S3", "platforms": 200, "skills": 1000, "files": 10000},
]

EXIT_OK = 0
EXIT_HARD = 1
EXIT_UNAVAILABLE = 2

UNVERIFIED = "UNVERIFIED"


# ---------------------------------------------------------------- 度量原语


def median_ms(samples: list[float]) -> float | None:
    """中位数。样本为空返回 None (测不到), 绝不返回 0 充数。"""
    if not samples:
        return None
    return round(statistics.median(samples), 1)


def links_per_sec(links: int, ms: float | None) -> float | None:
    if not ms or ms <= 0 or links <= 0:
        return None
    return round(links / (ms / 1000.0), 1)


def rss_peak_mb(samples: list[float]) -> float | None:
    """峰值 RSS (MB)。空样本 => None => 调用方标 UNVERIFIED, 不折算 0。"""
    if not samples:
        return None
    return round(max(samples), 1)


# ---------------------------------------------------------------- 沙箱构造


def build_platforms(scale: dict, sandbox: Path) -> tuple[Path, Path, Path, int]:
    """造假的 platforms.json 与库。返回 (platforms_json, fakehome, skills_root, 技能条目数)。

    home 用绝对路径写在运行时生成的 JSON 里 (该文件落在 _temp/ 且不入库), 因此不含
    任何个人标识, 与仓库的"禁个人绝对路径"口径不冲突。
    """
    library = sandbox / "library"
    memory_root = library / "memory"
    skills_root = library / "skills"
    fakehome = sandbox / "fakehome"
    for d in (memory_root, skills_root, fakehome):
        d.mkdir(parents=True, exist_ok=True)

    # 库文件: 观测 verify 是否随库规模变化。结论是"不变", 这个数组就是证据。
    for i in range(scale["files"]):
        bucket = memory_root / ("bucket%02d" % (i % 16))
        bucket.mkdir(parents=True, exist_ok=True)
        (bucket / ("note%05d.md" % i)).write_text("bench payload %d\n" % i, encoding="utf-8")
        if (i + 1) % 2000 == 0:
            progress("  %s: %d/%d library files written" % (scale["name"], i + 1, scale["files"]))

    for i in range(scale["skills"]):
        sd = skills_root / ("bench-skill-%04d" % i)
        sd.mkdir(parents=True, exist_ok=True)
        (sd / "SKILL.md").write_text(
            "---\nname: bench-skill-%04d\ndescription: benchmark fixture\n---\n\nbody\n" % i,
            encoding="utf-8",
        )
        if (i + 1) % 250 == 0:
            progress("  %s: %d/%d skill dirs written" % (scale["name"], i + 1, scale["skills"]))

    platforms = []
    for i in range(scale["platforms"]):
        home = fakehome / ("bench%03d" % i)
        platforms.append(
            {
                "id": "bench%03d" % i,
                "label": "Bench%03d" % i,
                "support": "ga",
                "os": ["windows", "darwin", "linux"],
                "home": str(home),
                "mounts": [{"kind": "link", "from": str(home / "memory"), "to": "<memory>"}],
                "inject": [{"path": str(home / "AGENTS.md"), "mode": "marker"}],
            }
        )
    # per-skill 平台造 3 个: per-skill 挂载是幂等的, 对同一个平台再 enable 只会走
    # no-op 分支, 测出来的"第二次耗时"是空操作。只有多个独立目标才能取中位数。
    for k in range(3):
        ps_home = fakehome / ("benchperskill%d" % k)
        platforms.append(
            {
                "id": "benchperskill%d" % k,
                "label": "BenchPerSkill%d" % k,
                "support": "ga",
                "os": ["windows", "darwin", "linux"],
                "home": str(ps_home),
                "mounts": [{"kind": "per-skill", "from": str(ps_home / "skills"), "to": "<skills>"}],
                "inject": [{"path": str(ps_home / "AGENTS.md"), "mode": "marker"}],
            }
        )

    doc = {
        "schema": "fenjue-platforms-v1",
        "version": "0.0.0-bench",
        "roots": {"memory": str(memory_root), "skills": str(skills_root)},
        "site": {"primary": "https://example.invalid", "mirror": "", "local": "http://127.0.0.1:7799"},
        "platforms": platforms,
    }
    cfg_path = sandbox / "platforms.json"
    cfg_path.write_text(json.dumps(doc, ensure_ascii=False, indent=2), encoding="utf-8")
    skills_now = sum(1 for p in skills_root.iterdir() if p.is_dir())
    return cfg_path, fakehome, skills_root, skills_now


def child_env(fakehome: Path) -> dict:
    """让子进程以为 home 在沙箱里。USERPROFILE (Windows) 与 HOME (Unix) 都设。"""
    env = dict(os.environ)
    env["USERPROFILE"] = str(fakehome)
    env["HOME"] = str(fakehome)
    return env


def cleanup(sandbox: Path) -> None:
    """清沙箱用系统原生命令, 不用 shutil.rmtree。

    沙箱里有 junction (平台目录指向库根)。原生 rd / rm -rf 对 reparse point 不
    递归, 而 shutil.rmtree 在 Windows 上会跟进去: 被指向的库根同时也在遍历树里,
    于是重复删除、抛 FileNotFoundError、整段清理退化成重试。表现为"压测跑完就卡住"。
    """
    if not sandbox.exists():
        return
    try:
        if os.name == "nt":
            subprocess.run(["cmd", "/c", "rd", "/s", "/q", str(sandbox)],
                           capture_output=True, timeout=300, check=False)
        else:
            subprocess.run(["rm", "-rf", str(sandbox)], capture_output=True, timeout=300, check=False)
    except (OSError, subprocess.TimeoutExpired):
        pass


# ---------------------------------------------------------------- 采样


def sample_rss(pid: int, times: int = 15, interval: float = 0.1) -> list[float]:
    """同步采 RSS (MB)。

    刻意不用后台线程: 线程会在异常路径上静默提前结束 (psutil 在进程刚启动的窗口里
    可能抛错), 于是采样条数变得不确定, 同一个二进制的数字能差 4 倍 (实测 11.2MB vs
    51.4MB)。同步采样条数固定, 数字才可复算。没有 psutil 就返回空列表, 调用方标
    UNVERIFIED, 绝不填 0。
    """
    try:
        import psutil  # noqa: PLC0415
    except Exception:  # noqa: BLE001 - 没有 psutil 就明确不报数
        return []
    samples: list[float] = []
    try:
        proc = psutil.Process(pid)
        for _ in range(times):
            try:
                samples.append(proc.memory_info().rss / (1024.0 * 1024.0))
            except Exception:  # noqa: BLE001 - 进程没了就停, 不是错误
                break
            time.sleep(interval)
    except Exception:  # noqa: BLE001 - 拿不到进程就明确不报数
        return samples
    return samples


def pick_free_port() -> int:
    """向系统要一个当前空闲的回环端口。

    固定端口 (7811-7813) 会在两种真实场景下炸: 上一次跑被中断后 serve 残留,
    或同一台机器上并行跑两个压测。压测脚本因为端口占用失败, 会被误读成
    "程序慢" 或 "程序有问题", 因此这里必须动态取端口。
    """
    import socket  # noqa: PLC0415

    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
        s.bind(("127.0.0.1", 0))
        return int(s.getsockname()[1])


def wait_health(port: int, proc: subprocess.Popen, log_path: Path, timeout: float) -> tuple[bool, float]:
    """轮询 /api/health (免令牌)。返回 (是否就绪, 从调用到 200 的毫秒)。

    每 5 秒打一行进度: serve 起不来的典型原因是端口占用或配置解析失败, 静默等满
    超时只会得到一个没有线索的 False。
    """
    url = "http://127.0.0.1:%d/api/health" % port
    start = time.perf_counter()
    last_note = 0.0
    while time.perf_counter() - start < timeout:
        if proc.poll() is not None:
            progress("serve exited early with rc=%s" % proc.returncode)
            return False, round((time.perf_counter() - start) * 1000.0, 1)
        try:
            with urllib.request.urlopen(url, timeout=2) as resp:
                if resp.status == 200:
                    return True, round((time.perf_counter() - start) * 1000.0, 1)
        except (urllib.error.URLError, OSError):
            pass
        elapsed = time.perf_counter() - start
        if elapsed - last_note >= 5.0:
            last_note = elapsed
            tail = ""
            try:
                tail = "\n".join(log_path.read_bytes().decode("utf-8", "replace").splitlines()[-6:])
            except OSError:
                pass
            progress("waiting for /api/health (%.1fs)%s" % (elapsed, (" | serve output: " + tail) if tail else ""))
        time.sleep(0.02)
    return False, round((time.perf_counter() - start) * 1000.0, 1)


def timed(cmd: list[str], env: dict, timeout: float = 300.0, label: str = "") -> tuple[float, bool, str]:
    """跑一条子命令并计时。

    输出落临时文件而不是 PIPE: verify 的输出随平台数增长 (200 平台约 40KB),
    远超 Windows 管道 4KB 缓冲, 用 PIPE 会把子进程永久卡死在写 stdout 上,
    表现为"压测挂住"。同时每 5 秒打一行进度, 避免分钟级静默被当成挂死。
    失败时回传输出尾部, 否则只剩一个非零退出码, 无法归因。
    """
    t0 = time.perf_counter()
    fd, log_path = tempfile.mkstemp(prefix="bench_cmd_", suffix=".log")
    logf = os.fdopen(fd, "wb")
    try:
        proc = subprocess.Popen(cmd, env=env, stdout=logf, stderr=subprocess.STDOUT)
    except OSError as exc:
        logf.close()
        os.unlink(log_path)
        return round((time.perf_counter() - t0) * 1000.0, 1), False, "spawn failed: %s" % exc
    timed_out = False
    while True:
        try:
            proc.wait(timeout=5)
            break
        except subprocess.TimeoutExpired:
            if time.perf_counter() - t0 > timeout:
                proc.kill()
                timed_out = True
                break
            if label:
                progress("  %s still running (%.1fs)" % (label, time.perf_counter() - t0))
    ms = round((time.perf_counter() - t0) * 1000.0, 1)
    logf.close()
    tail = ""
    try:
        data = Path(log_path).read_bytes()
        tail = data.decode("utf-8", "replace").strip().splitlines()[-12:]
        tail = "\n".join(tail)
    except OSError:
        pass
    try:
        os.unlink(log_path)
    except OSError:
        pass
    if timed_out:
        return ms, False, "timed out after %.0fs; tail:\n%s" % (timeout, tail)
    return ms, proc.returncode == 0, tail


# ---------------------------------------------------------------- 一档


def progress(msg: str) -> None:
    """进度写 stdout 并立刻 flush。

    刻意不用 stderr: 进度是给操作者看的正常输出, 而在命令被嵌套 (例如套一层
    powershell -Command) 时 stderr 常常不被转发, 于是外部看不到任何输出, 把一次
    正在正常跑的压测误判成挂死而掐掉。真正的错误仍然走 stderr。
    """
    print("[bench] %s" % msg, flush=True)


def run_scale(binary: Path, scale: dict, keep: bool, rss_repeats: int) -> dict:
    sandbox = SANDBOX_ROOT / ("%s-%d" % (scale["name"], int(time.time() * 1000)))
    sandbox.mkdir(parents=True, exist_ok=True)
    progress("%s: building sandbox (platforms=%d skills=%d files=%d)"
             % (scale["name"], scale["platforms"], scale["skills"], scale["files"]))
    result: dict = {"scale": scale["name"], "platforms": scale["platforms"], "skills_requested": scale["skills"],
                    "library_files": scale["files"], "unverified": [], "errors": []}
    proc = None
    try:
        cfg, fakehome, skills_root, skills_now = build_platforms(scale, sandbox)
        env = child_env(fakehome)
        result["skills_actual"] = skills_now
        progress("%s: sandbox ready, actual skill dirs=%d" % (scale["name"], skills_now))

        # 1. 冷启动: 进程起来 -> /api/health 200。含首次建根 + 空库播种 + manifest 校验。
        rss_samples: list[float] = []
        port = pick_free_port()
        result["port"] = port
        serve_log = sandbox / "serve.log"
        serve_fh = serve_log.open("wb")
        proc = subprocess.Popen(
            [str(binary), "serve", "--port", str(port), "--platforms", str(cfg)],
            env=env, stdout=serve_fh, stderr=subprocess.STDOUT,
        )
        serve_fh.close()
        ok, cold_ms = wait_health(port, proc, serve_log, timeout=60)
        result["cold_start_ms"] = cold_ms
        result["serve_ready"] = ok
        if not ok:
            tail = ""
            try:
                tail = "\n".join(serve_log.read_bytes().decode("utf-8", "replace").splitlines()[-8:])
            except OSError:
                pass
            result["errors"].append("serve did not become healthy on port %d: %s" % (port, tail))
            return result
        # 稳态内存: serve 起来后同步采 1.5 秒, 覆盖挂载请求到达前的真实占用。
        rss_samples = sample_rss(proc.pid)
        progress("%s: serve healthy in %s ms, rss samples=%d" % (scale["name"], cold_ms, len(rss_samples)))

        # 2. link 类平台: 逐端冷启用 (每次都是一次真实进程启动, 端到端成本)。
        link_samples: list[float] = []
        for i in range(scale["platforms"]):
            ms, good, tail = timed(
                [str(binary), "enable", "bench%03d" % i, "--platforms", str(cfg)], env,
                label="%s enable bench%03d" % (scale["name"], i),
            )
            if not good:
                result["errors"].append("enable bench%03d failed: %s" % (i, tail))
                break
            link_samples.append(ms)
            if (i + 1) % 50 == 0:
                progress("%s: enabled %d/%d link platforms" % (scale["name"], i + 1, scale["platforms"]))
        result["enable_link_ms_median"] = median_ms(link_samples)
        result["enable_link_samples"] = len(link_samples)

        # 3. per-skill 平台: 每个平台建 N 条技能链接, 挂载吞吐的真实瓶颈在这里。
        #    3 个独立平台取中位数 (单个平台重跑只会命中幂等 no-op)。
        ps_samples: list[float] = []
        ps_error = ""
        for k in range(3):
            progress("%s: enabling per-skill platform #%d (%d links)" % (scale["name"], k, skills_now))
            ps_ms, ps_ok, ps_tail = timed(
                [str(binary), "enable", "benchperskill%d" % k, "--platforms", str(cfg)], env,
                label="%s enable benchperskill%d" % (scale["name"], k), timeout=600,
            )
            if not ps_ok:
                ps_error = "enable benchperskill%d failed: %s" % (k, ps_tail)
                break
            ps_samples.append(ps_ms)
        ps_median = median_ms(ps_samples)
        result["enable_perskill_ms"] = ps_median
        result["perskill_samples"] = ps_samples
        result["perskill_links"] = skills_now if ps_median else None
        result["mount_links_per_sec"] = links_per_sec(skills_now, ps_median)
        if ps_error:
            result["errors"].append(ps_error)

        # 4. verify: 3 次取中位数。代码依据: statusPerSkill 遍历技能根逐条探活,
        #    而 memory 根只被 stat 一次 => 成本随 (平台数 + 技能数) 走, 不随库文件数走。
        v_samples: list[float] = []
        for n in range(3):
            ms, good, v_tail = timed(
                [str(binary), "verify", "--platforms", str(cfg)], env,
                label="%s verify pass %d" % (scale["name"], n + 1), timeout=600,
            )
            if not good:
                result["errors"].append("verify failed: %s" % v_tail)
                break
            v_samples.append(ms)
            progress("%s: verify pass %d/3 = %s ms" % (scale["name"], n + 1, ms))
        result["verify_ms_median"] = median_ms(v_samples)

        # 5. 内存两段: 刚起来的稳态, 以及全部挂载+verify 之后的操作后值。
        #    分两段是因为两者不是一个量级 (稳态只加载了配置, 操作后才吃到分配峰值),
        #    只报一个数会掩盖这个差异。
        after = sample_rss(proc.pid)
        result["rss_steady_mb"] = rss_peak_mb(rss_samples)
        result["rss_after_ops_mb"] = rss_peak_mb(after)
        result["rss_peak_mb"] = rss_peak_mb(rss_samples + after)
        if not rss_samples and not after:
            result["unverified"].append("rss: psutil unavailable on this machine (never counted as 0)")
        progress("%s: rss steady=%s MB after_ops=%s MB"
                 % (scale["name"], result["rss_steady_mb"], result["rss_after_ops_mb"]))
        return result
    except Exception as exc:  # noqa: BLE001 - 任何一档崩了都要留下原因, 不是静默跳过
        result["errors"].append("%s: %s" % (type(exc).__name__, exc))
        return result
    finally:
        if proc is not None and proc.poll() is None:
            proc.terminate()
            try:
                proc.wait(timeout=10)
            except subprocess.TimeoutExpired:
                proc.kill()
        if not keep and sandbox.exists():
            cleanup(sandbox)


# ---------------------------------------------------------------- 自测


def selftest() -> int:
    """度量函数必须满足: 空样本 => None (=> UNVERIFIED), 绝不是 0。"""
    failures: list[str] = []

    if median_ms([]) is not None:
        failures.append("median_ms([]) must be None (UNVERIFIED), got %r" % median_ms([]))
    if median_ms([10.0, 20.0, 30.0]) != 20.0:
        failures.append("median of an odd-length sorted sample must be the middle one")
    if links_per_sec(100, 0) is not None:
        failures.append("links_per_sec with 0 ms must be None, not a division crash and not 0")
    if links_per_sec(100, None) is not None:
        failures.append("links_per_sec(None) must be None (UNVERIFIED)")
    if links_per_sec(200, 1000.0) != 200.0:
        failures.append("links_per_sec(200 links, 1000 ms) must be 200.0/s, got %r" % links_per_sec(200, 1000.0))
    if rss_peak_mb([]) is not None:
        failures.append("rss_peak_mb([]) must be None (no psutil => UNVERIFIED), not 0.0")
    if rss_peak_mb([1.0, 9.0, 3.0]) != 9.0:
        failures.append("rss_peak_mb must take the max sample")
    # 端口必须是真正可用的: 拿到就 bind 一次, 否则固定端口/残留进程会让压测假失败。
    import socket  # noqa: PLC0415

    port = pick_free_port()
    if not (1024 < port < 65536):
        failures.append("pick_free_port returned an implausible port: %r" % port)
    else:
        try:
            with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
                s.bind(("127.0.0.1", port))
        except OSError as exc:
            failures.append("pick_free_port returned %d which cannot be bound: %s" % (port, exc))

    if failures:
        for f in failures:
            print("SELFTEST-FAIL %s" % f, file=sys.stderr)
        print("[GATE:bench-selftest-red] %d case(s) failed" % len(failures), file=sys.stderr)
        return 1
    print("[GATE:bench-selftest-pass] empty samples stay UNVERIFIED, throughput math bites")
    return 0


# ---------------------------------------------------------------- main


def explain() -> str:
    """复杂度归因: 每个数字对应代码里的一处, 不靠猜。"""
    return (
        "挂载阶段 O(平台数 x 每端挂载数 + 技能数) 次系统调用。Windows 走 junction (单次系统调用, 免提权),\n"
        "  瓶颈在 per-skill 的并发上限 (mount/perskill.go) 而非磁盘 IO。\n"
        "verify 阶段 O(平台数 + 技能数) 次 stat: statusPerSkill (state/verify.go) 会 ReadDir 技能根并逐条探活,\n"
        "  而 memory 根只被 stat 一次 => verify 成本不随记忆库文件数增长 (这是代码结论, 由 S1/S2/S3 三档实测复核)。\n"
        "冷启动含 seed.Bootstrap 建根 + 首次播种 29 个内嵌文件 + SHA-256 全量校验 (seed.VerifyPack)。"
    )


def main() -> int:
    ap = argparse.ArgumentParser(description="Scale benchmark for the hub agent (sandboxed, never touches the real ~/.fenjue).")
    ap.add_argument("--json", action="store_true", help="emit machine-readable output")
    ap.add_argument("--selftest", action="store_true", help="run the metric self-test and exit")
    ap.add_argument("--scale", choices=[s["name"] for s in SCALES], help="run one scale only")
    ap.add_argument("--keep", action="store_true", help="keep sandboxes for inspection")
    args = ap.parse_args()

    if args.selftest:
        return selftest()

    SANDBOX_ROOT.mkdir(parents=True, exist_ok=True)
    t0 = time.perf_counter()
    binary = SANDBOX_ROOT / ("fenjue-agent-bench%s" % (".exe" if os.name == "nt" else ""))
    # cwd 已由 __main__ 切到 agent/ (go build 需要 module 目录); 其余路径一律绝对路径。
    build_ms, built, build_tail = timed(
        ["go", "build", "-o", str(binary), "./cmd/fenjue-agent"], dict(os.environ), timeout=600, label="go build",
    )
    if not built:
        print("error: go build failed; cannot benchmark", file=sys.stderr)
        print(build_tail, file=sys.stderr)
        print("[GATE:bench-unavailable]", file=sys.stderr)
        return EXIT_UNAVAILABLE

    payload = {
        "go_build_ms": build_ms,
        "os": os.name,
        "python": sys.version.split()[0],
        "total_ms": None,
        "scales": [],
        "explain": explain(),
    }
    scales = [s for s in SCALES if not args.scale or s["name"] == args.scale]
    for scale in scales:
        payload["scales"].append(run_scale(binary, scale, args.keep, 1))
    payload["total_ms"] = round((time.perf_counter() - t0) * 1000.0, 1)

    hard = [r for r in payload["scales"] if r.get("errors")]
    unverified = [u for r in payload["scales"] for u in r.get("unverified", [])]

    # 产物落地 (_temp/ 由 .gitignore 命中, 复算命令见输出末行)。
    latest = SANDBOX_ROOT / "latest.json"
    latest.write_text(json.dumps(payload, ensure_ascii=False, indent=2), encoding="utf-8")
    payload["latest_json"] = str(latest)

    if args.json:
        print(json.dumps(payload, ensure_ascii=False, indent=2))
    else:
        print("go build: %.1f ms   total: %d ms   os: %s   python: %s"
              % (payload["go_build_ms"], payload["total_ms"], payload["os"], payload["python"]))
        print("")
        head = "%-4s %9s %8s %10s %11s %10s %12s %10s" % (
            "scale", "platforms", "skills", "cold_ms", "link_ms", "perskill", "links/s", "verify_ms")
        print(head)
        print("-" * len(head))
        for r in payload["scales"]:
            print("%-4s %9d %8s %10s %11s %10s %12s %10s" % (
                r["scale"], r["platforms"], r.get("skills_actual", "?"),
                r.get("cold_start_ms", "?"),
                r.get("enable_link_ms_median", "?"),
                r.get("enable_perskill_ms", "?"),
                r.get("mount_links_per_sec", "?"),
                r.get("verify_ms_median", "?"),
            ))
        print("")
        for r in payload["scales"]:
            if r.get("rss_steady_mb") is not None:
                print("rss (%s): steady=%s MB  after_ops=%s MB  peak=%s MB"
                      % (r["scale"], r["rss_steady_mb"], r.get("rss_after_ops_mb"), r.get("rss_peak_mb")))
        for u in unverified:
            print("UNVERIFIED (never counted as a pass): %s" % u)
        for r in payload["scales"]:
            for e in r.get("errors", []):
                print("ERROR [%s] %s" % (r["scale"], e))
        print("")
        print(payload["explain"])
        print("")
        print("recompute: python scripts/bench_scale.py --json")

    if hard:
        print("[GATE:bench-red] %d scale(s) reported errors" % len(hard), file=sys.stderr)
        return EXIT_HARD
    if unverified:
        print("[GATE:bench-unverified]", file=sys.stderr)
    return EXIT_OK


if __name__ == "__main__":
    os.chdir(AGENT_DIR)  # go build 需要 module 目录; 其余路径一律用绝对路径
    sys.exit(main())