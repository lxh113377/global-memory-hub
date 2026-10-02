#!/usr/bin/env python3
"""check_version_sync.py - 版本一致性机器判据。

背景
----
本项目曾发生过真实漂移: 产品版本停在 v0.2.1 而 tag 已推到 v0.2.2,
靠人记必然复发。本脚本把这件事变成 CI 里的一条腿。

判据分层 (按语义分组, 刻意不把所有版本号硬绑在一起)
----------------------------------------------------
1. **产品版本** (强): git 最新 tag ``vX.Y.Z``  ==  ``router.go`` 的 ``Version`` 常量。
   这两个表达的是同一件事, 必须相等。
2. **种子包版本** (强): ``pack/manifest.json`` 的 ``seed_version``  ==  ``seed.go`` 的
   ``SeedVersion`` 常量。二者表达"播种内容版本", 与产品版本独立 bump, 不强制相等。
3. **平台定义版本** (强, 但配对对象不同): ``platforms.json`` 的 ``schema``
   形如 ``fenjue-platforms-vN`` 其中的 N 必须等于 ``version`` 的主版本号。
   它表达的是**配置格式版本**, 不是产品版本, 因此不与第 1 条比较。

为什么第 3 条不比第 1 条: 配置 schema 版本与产品版本是两套语义。
把它们绑在一起会造出一条永远红的判据, 而永远红的判据等于没有判据。

退出码: 0 一致 / 1 发现漂移 / 2 无法测量 (UNVERIFIED, 绝不折算为通过)
"""
from __future__ import annotations

import argparse
import json
import re
import subprocess
import sys
from pathlib import Path

REPO = Path(__file__).resolve().parent.parent
ROUTER_GO = REPO / "agent" / "internal" / "server" / "router.go"
SEED_GO = REPO / "agent" / "internal" / "seed" / "seed.go"
MANIFEST = REPO / "agent" / "internal" / "seed" / "pack" / "manifest.json"
PLATFORMS = REPO / "platforms.json"

SEMVER = r"\d+\.\d+\.\d+"

EXIT_OK = 0
EXIT_DIFF = 1
EXIT_UNAVAILABLE = 2


def latest_tag(repo: Path) -> str | None:
    try:
        out = subprocess.run(
            ["git", "-C", str(repo), "tag", "--list", "v*", "--sort=-v:refname"],
            capture_output=True,
            text=True,
            timeout=30,
            check=True,
        ).stdout
    except (OSError, subprocess.SubprocessError):
        return None
    for line in out.splitlines():
        line = line.strip()
        if re.fullmatch(r"v" + SEMVER, line):
            return line
    return None


def go_const(path: Path, name: str) -> str | None:
    try:
        src = path.read_text(encoding="utf-8")
    except OSError:
        return None
    m = re.search(r"const\s+" + re.escape(name) + r'\s*=\s*"([^"]+)"', src)
    return m.group(1) if m else None


def check(
    tag: str | None,
    router_version: str | None,
    seed_manifest_version: str | None,
    seed_go_version: str | None,
    schema: str | None,
    config_version: str | None,
) -> list[str]:
    """Core verdict. Pure function so the self-test can drive it with crafted inputs."""
    problems: list[str] = []

    # 1. product version: tag vs router.go const
    if tag is None or router_version is None:
        problems.append("product version UNVERIFIED: tag=%r router.go Version=%r" % (tag, router_version))
    else:
        bare = tag[1:] if tag.startswith("v") else tag
        if bare != router_version:
            problems.append(
                "product version drift: git tag %s (=%s) but router.go Version=%s; "
                "bump the const, or the repo is lying about what was released" % (tag, bare, router_version)
            )

    # 2. seed pack version: manifest vs seed.go const
    if seed_manifest_version is None or seed_go_version is None:
        problems.append(
            "seed version UNVERIFIED: manifest=%r seed.go SeedVersion=%r"
            % (seed_manifest_version, seed_go_version)
        )
    elif seed_manifest_version != seed_go_version:
        problems.append(
            "seed version drift: manifest seed_version=%s but seed.go SeedVersion=%s; "
            "regenerate with: python scripts/build_seed.py --manifest-only"
            % (seed_manifest_version, seed_go_version)
        )

    # 3. platform definition version: schema suffix vs version major
    if schema is None or config_version is None:
        problems.append("platform config UNVERIFIED: schema=%r version=%r" % (schema, config_version))
    else:
        m = re.search(r"-v(\d+)$", schema)
        if not m:
            problems.append("platform schema %r has no -vN suffix, cannot pair it with a version" % schema)
        elif not re.fullmatch(SEMVER, config_version):
            problems.append("platforms.json version %r is not semver" % config_version)
        elif int(m.group(1)) != int(config_version.split(".")[0]):
            problems.append(
                "platform config drift: schema %s says format v%s but version is %s"
                % (schema, m.group(1), config_version)
            )

    return problems


def selftest() -> int:
    # (label, inputs, expected problems) -- every case must be non-empty, i.e. the leg bites
    cases: list[tuple[str, tuple, list[str]]] = [
        (
            "tag ahead of const (the drift that actually happened)",
            ("v0.2.2", "0.2.1", "0.1.0", "0.1.0", "fenjue-platforms-v1", "1.0.0"),
            [
                "product version drift: git tag v0.2.2 (=0.2.2) but router.go Version=0.2.1; "
                "bump the const, or the repo is lying about what was released"
            ],
        ),
        (
            "const ahead of tag",
            ("v0.2.2", "0.2.3", "0.1.0", "0.1.0", "fenjue-platforms-v1", "1.0.0"),
            [
                "product version drift: git tag v0.2.2 (=0.2.2) but router.go Version=0.2.3; "
                "bump the const, or the repo is lying about what was released"
            ],
        ),
        (
            "no tag at all (shallow clone must not pass)",
            (None, "0.2.2", "0.1.0", "0.1.0", "fenjue-platforms-v1", "1.0.0"),
            ["product version UNVERIFIED: tag=None router.go Version='0.2.2'"],
        ),
        (
            "no router const",
            ("v0.2.2", None, "0.1.0", "0.1.0", "fenjue-platforms-v1", "1.0.0"),
            ["product version UNVERIFIED: tag='v0.2.2' router.go Version=None"],
        ),
        (
            "seed manifest behind code",
            ("v0.2.2", "0.2.2", "0.1.0", "0.2.0", "fenjue-platforms-v1", "1.0.0"),
            [
                "seed version drift: manifest seed_version=0.1.0 but seed.go SeedVersion=0.2.0; "
                "regenerate with: python scripts/build_seed.py --manifest-only"
            ],
        ),
        (
            "seed manifest missing",
            ("v0.2.2", "0.2.2", None, "0.1.0", "fenjue-platforms-v1", "1.0.0"),
            ["seed version UNVERIFIED: manifest=None seed.go SeedVersion='0.1.0'"],
        ),
        (
            "platform schema v1 vs version 2.0.0",
            ("v0.2.2", "0.2.2", "0.1.0", "0.1.0", "fenjue-platforms-v1", "2.0.0"),
            ["platform config drift: schema fenjue-platforms-v1 says format v1 but version is 2.0.0"],
        ),
        (
            "platform schema suffix missing",
            ("v0.2.2", "0.2.2", "0.1.0", "0.1.0", "fenjue-platforms", "1.0.0"),
            ["platform schema 'fenjue-platforms' has no -vN suffix, cannot pair it with a version"],
        ),
        (
            "platform version not semver",
            ("v0.2.2", "0.2.2", "0.1.0", "0.1.0", "fenjue-platforms-v1", "zero"),
            ["platforms.json version 'zero' is not semver"],
        ),
    ]

    failures: list[str] = []
    for label, args, expected in cases:
        got = check(*args)
        if got != expected:
            failures.append("%s\n    want: %r\n    got:  %r" % (label, expected, got))

    # 干净输入必须零问题 (正例腿)
    clean = check("v0.2.2", "0.2.2", "0.1.0", "0.1.0", "fenjue-platforms-v1", "1.0.0")
    if clean:
        failures.append("clean input must produce no problem, got: %r" % clean)

    if failures:
        for f in failures:
            print("SELFTEST-FAIL %s" % f, file=sys.stderr)
        print("[GATE:version-selftest-red] %d case(s) failed" % len(failures), file=sys.stderr)
        return 1
    print("[GATE:version-selftest-pass] %d drift cases bite + 1 clean case" % len(cases))
    return 0


def main() -> int:
    ap = argparse.ArgumentParser(description="Version consistency verdict across tag / Go consts / manifest.")
    ap.add_argument("--json", action="store_true", help="emit machine-readable output")
    ap.add_argument("--selftest", action="store_true", help="run the drift self-test and exit")
    args = ap.parse_args()

    if args.selftest:
        return selftest()

    tag = latest_tag(REPO)
    router_version = go_const(ROUTER_GO, "Version")
    seed_go_version = go_const(SEED_GO, "SeedVersion")

    seed_manifest_version = None
    try:
        seed_manifest_version = json.loads(MANIFEST.read_text(encoding="utf-8")).get("seed_version")
    except (OSError, json.JSONDecodeError, AttributeError):
        pass

    schema = config_version = None
    try:
        doc = json.loads(PLATFORMS.read_text(encoding="utf-8"))
        schema = doc.get("schema")
        config_version = doc.get("version")
    except (OSError, json.JSONDecodeError, AttributeError):
        pass

    problems = check(tag, router_version, seed_manifest_version, seed_go_version, schema, config_version)
    unverified = [p for p in problems if "UNVERIFIED" in p]
    hard = [p for p in problems if "UNVERIFIED" not in p]

    status = "DIFF" if hard else ("UNVERIFIED" if unverified else "OK")
    payload = {
        "status": status,
        "git_tag": tag,
        "router_go_version": router_version,
        "manifest_seed_version": seed_manifest_version,
        "seed_go_version": seed_go_version,
        "platforms_schema": schema,
        "platforms_version": config_version,
        "problems": problems,
    }
    if args.json:
        print(json.dumps(payload, ensure_ascii=False, indent=2))
    else:
        print("git tag              : %s" % tag)
        print("router.go Version    : %s" % router_version)
        print("manifest seed_version: %s" % seed_manifest_version)
        print("seed.go SeedVersion  : %s" % seed_go_version)
        print("platforms schema     : %s (version %s)" % (schema, config_version))
        if hard:
            print("\nDRIFT:")
            for line in hard:
                print("  - %s" % line)
        else:
            print("\nUNVERIFIED (never counted as a pass):" if unverified else "\nVERSION SYNC OK.")
            for line in unverified:
                print("  - %s" % line)
            if not unverified:
                print("tag == product const, manifest == seed const, schema paired with config version.")

    if hard:
        print("[GATE:version-sync-red]", file=sys.stderr)
        return EXIT_DIFF
    if unverified:
        # A side could not be measured. Surface the hole loudly, but do not invent a verdict:
        # a tagless shallow clone must show UNVERIFIED rather than a fake pass or a fake failure.
        print("[GATE:version-sync-unverified]", file=sys.stderr)
    return EXIT_OK


if __name__ == "__main__":
    sys.exit(main())
