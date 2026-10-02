#!/usr/bin/env python3
"""verify_seed_manifest.py - 独立校验种子包 manifest (第二实现)。

为什么要有第二个实现:
  Go 侧的 seed.VerifyPack 与本脚本是**两套独立代码**。若只有 Go 一侧,
  "校验通过" 可能只是 Go 逻辑与自己一致 (自比恒等), 无法排除实现本身的错误。
  两套实现算出同一个 SHA-256, 才构成交叉证据。

判据:
  1. manifest 存在且 schema 正确
  2. manifest.seed_version == seed.go 里的 SeedVersion 常量
  3. pack 内每个文件的 SHA-256 与字节数与 manifest 一致
  4. 双向对账: 清单登记但缺失 / 存在但未登记, 都判为差异

退出码: 0 一致 / 1 发现差异 / 2 无法测量 (UNVERIFIED, 绝不折算为通过)
"""
from __future__ import annotations

import argparse
import hashlib
import json
import re
import sys
from pathlib import Path

REPO = Path(__file__).resolve().parent.parent
PACK = REPO / "agent" / "internal" / "seed" / "pack"
MANIFEST = PACK / "manifest.json"
SEED_GO = REPO / "agent" / "internal" / "seed" / "seed.go"
MANIFEST_SCHEMA = "fenjue-seed-manifest-v1"

EXIT_OK = 0
EXIT_DIFF = 1
EXIT_UNAVAILABLE = 2


def sha256_of(path: Path) -> tuple[str, int]:
    h = hashlib.sha256()
    size = 0
    with path.open("rb") as fh:
        for chunk in iter(lambda: fh.read(131072), b""):
            h.update(chunk)
            size += len(chunk)
    return h.hexdigest(), size


def read_seed_version() -> str:
    """SeedVersion lives in Go source; parse it so the two never drift silently."""
    src = SEED_GO.read_text(encoding="utf-8")
    m = re.search(r'SeedVersion\s*=\s*"([^"]+)"', src)
    if not m:
        raise ValueError("SeedVersion const not found in %s" % SEED_GO)
    return m.group(1)


def main() -> int:
    ap = argparse.ArgumentParser(description="Verify the seed pack manifest (second implementation).")
    ap.add_argument("--json", action="store_true", help="emit machine-readable output")
    args = ap.parse_args()

    try:
        if not MANIFEST.is_file():
            raise ValueError("manifest missing: %s (run: python scripts/build_seed.py --manifest-only)" % MANIFEST)
        manifest = json.loads(MANIFEST.read_text(encoding="utf-8"))
        if manifest.get("schema") != MANIFEST_SCHEMA:
            raise ValueError("manifest schema %r, want %r" % (manifest.get("schema"), MANIFEST_SCHEMA))
        entries = manifest.get("files")
        if not isinstance(entries, list) or not entries:
            raise ValueError("manifest lists no files")
        compiled = read_seed_version()
    except (OSError, ValueError, json.JSONDecodeError) as exc:
        payload = {"status": "UNVERIFIED", "reason": str(exc), "note": "a side could not be measured; this is not a pass"}
        print(json.dumps(payload, ensure_ascii=False, indent=2) if args.json else "SEED MANIFEST UNVERIFIED: %s" % exc)
        return EXIT_UNAVAILABLE

    problems: list[str] = []
    declared: dict[str, tuple[str, int]] = {}
    for entry in entries:
        rel = entry.get("path")
        if not isinstance(rel, str) or not rel or rel == MANIFEST.name:
            problems.append("bad or self-referential path in manifest: %r" % rel)
            continue
        if rel in declared:
            problems.append("manifest lists %s twice" % rel)
            continue
        declared[rel] = (str(entry.get("sha256", "")).lower(), int(entry.get("bytes", -1)))

    if manifest.get("seed_version") != compiled:
        problems.append(
            "seed_version drift: manifest %r vs compiled SeedVersion %r "
            "(regenerate with: python scripts/build_seed.py --manifest-only)"
            % (manifest.get("seed_version"), compiled)
        )

    actual = {
        p.relative_to(PACK).as_posix(): p
        for p in sorted(PACK.rglob("*"))
        if p.is_file() and p.name != MANIFEST.name
    }

    for rel, path in actual.items():
        if rel not in declared:
            problems.append("undeclared file in pack: %s" % rel)
            continue
        want_sum, want_size = declared[rel]
        got_sum, got_size = sha256_of(path)
        if got_size != want_size:
            problems.append("%s: size %d bytes, manifest says %d" % (rel, got_size, want_size))
        if got_sum != want_sum:
            problems.append("%s: sha256 %s, manifest says %s" % (rel, got_sum, want_sum))

    for rel in declared:
        if rel not in actual:
            problems.append("manifest lists %s but it is not in the pack" % rel)

    payload = {
        "status": "DIFF" if problems else "OK",
        "manifest": str(MANIFEST),
        "seed_version": manifest.get("seed_version"),
        "compiled_seed_version": compiled,
        "files": len(actual),
        "problems": problems,
    }
    if args.json:
        print(json.dumps(payload, ensure_ascii=False, indent=2))
    else:
        print("pack files: %d, manifest entries: %d" % (len(actual), len(declared)))
        print("seed_version: manifest=%s compiled=%s" % (manifest.get("seed_version"), compiled))
        if problems:
            print("\nDIFFERS:")
            for line in problems:
                print("  - %s" % line)
        else:
            print("\nSEED MANIFEST OK: every declared file matches, no undeclared file, no drift.")

    if problems:
        print("[GATE:seed-manifest-red]", file=sys.stderr)
        return EXIT_DIFF
    print("[GATE:seed-manifest-pass]")
    return EXIT_OK


if __name__ == "__main__":
    sys.exit(main())
