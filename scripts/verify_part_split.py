#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""verify_part_split.py — 分卷编号归属一致性判据（只读）

背景
----
本仓 ROADMAP 分三件：`ROADMAP.md`（权威面）+ `ROADMAP.part1.md`（§6 取证证据）
+ `ROADMAP.part2.md`（§1 已闭合 30 行）。分卷的代价是「一个编号可以落进两处，或两处都没有」。
r187 就因此让 G0 同时存在于主壳与 part2 —— 那次不报错：命令成功、行数对、字节接近。

设计上的三条硬约束（都来自踩过的坑）
----------------------------------------
1. **边界必须显式声明，不许用「剩余集」。**
   第一版按「整份文件里所有 `| G<数字> ` 行」取编号，结果把 §5 判据表里的 `| G0 |`
   也算成编号（主壳「33 个」= §1 六行 + §5 二十七行），把 §6 证据里**提到** G 编号的
   part1 误判成持有者 —— 一片假警报。所以：主壳只扫 `--shell-section` 指定的那一节；
   每个 part 用 `--owner` 显式声明它**持有**编号，默认为「不持有」。**提到 ≠ 持有。**

2. **期望集不许自指。**
   第一版拿「并集里的最大编号」当期望集，于是删掉最大那行时期望集跟着缩，**永远查不出缺失**。
   所以缺失检查只在显式给了 `--expect-range` 时才做；不给就明确跳过并说明，不装作查过了。

3. **判据自己也要有反例腿。**
   `--selftest` 含「重复 / 缺失 / 表头被当数据 / 提到≠持有」四类变异腿，全红才算 selftest 过。
   否则它就是个恒不命中的假判据。

它**不**查什么（明写，避免被当成「搬移已验证」的证据）
--------------------------------------------------------
- 不校验行内容是否被改写（需要逐行哈希基线，超出本判据范围）。
- 不校验 part 之间的引用是否成环。
- 不比较字节数 —— 分卷会让总量上升，那是预期的。

用法
----
    python scripts/verify_part_split.py --owner ROADMAP.part2.md --expect-range 0-35
    python scripts/verify_part_split.py --selftest
    python scripts/verify_part_split.py --json ...

退出码：0 = PASS，1 = FAIL，2 = 用法/取文件失败。与仓内其余判据同形态。
"""

from __future__ import annotations

import argparse
import json
import pathlib
import re
import sys

ROW_RE = re.compile(r"^\|\s*(G\d+)\s")
HEADER_RE = re.compile(r"^\|\s*编号\s*\|")


def _rows(text: str) -> list[str]:
    """取出数据行里的编号，**按出现顺序保留重复**（重复本身是信号，不能先收成 set）。"""
    out = []
    for line in text.splitlines():
        if HEADER_RE.match(line):
            continue
        m = ROW_RE.match(line)
        if m:
            out.append(m.group(1))
    return out


def section_of(text: str, start: str, end: str) -> str:
    """按标题字面切出一节。切不到就报错 —— 宁可失败，也不要悄悄扫全文。"""
    lines = text.splitlines()
    try:
        i = next(n for n, l in enumerate(lines) if l.startswith(start))
    except StopIteration:
        raise SystemExit(f"shell section start not found: {start!r}")
    try:
        j = next(n for n, l in enumerate(lines[i + 1:], i + 1) if l.startswith(end))
    except StopIteration:
        raise SystemExit(f"shell section end not found: {end!r}")
    return "\n".join(lines[i:j])


def check(shell_section_text: str, owners: dict[str, list[str]], expect: set[str] | None) -> dict:
    shell_rows = _rows(shell_section_text)

    dup_same_file: dict[str, list[str]] = {}
    counts: dict[str, int] = {}
    for gid in shell_rows:
        counts[gid] = counts.get(gid, 0) + 1
    for gid, n in counts.items():
        if n > 1:
            dup_same_file[gid] = ["ROADMAP.md"] * n

    across: dict[str, list[str]] = {}
    for gid in shell_rows:
        across.setdefault(gid, []).append("ROADMAP.md")
    for name, rows in owners.items():
        for gid in rows:
            across.setdefault(gid, []).append(name)
    duplicates = {g: w for g, w in across.items() if len(w) > 1}

    union = set(across)
    missing = sorted(expect - union, key=lambda g: int(g[1:])) if expect else []

    return {
        "ok": not duplicates and not missing,
        "shell_count": len(set(shell_rows)),
        "owner_counts": {n: len(set(r)) for n, r in owners.items()},
        "union_count": len(union),
        "expect_checked": expect is not None,
        "expect_count": len(expect) if expect else None,
        "duplicates": dict(sorted(duplicates.items())),
        "dup_within_one_file": dict(sorted(dup_same_file.items())),
        "missing": missing,
    }


def _parse_range(spec: str) -> set[str]:
    lo, hi = (int(x) for x in spec.split("-", 1))
    if hi < lo:
        raise SystemExit(f"bad --expect-range: {spec!r}")
    return {f"G{i}" for i in range(lo, hi + 1)}


def selftest() -> int:
    shell_sec = "## 1. 台账\n\n| 编号 | 任务 |\n|---|---|\n| G0 | a |\n| G1 | b |\n"
    legs = []

    # 正例：G0/G1 归主壳，G2/G3 归 owner，无重复无缺失。
    r = check(shell_sec, {"p2.md": ["G2", "G3"]}, {"G0", "G1", "G2", "G3"})
    legs.append(("clean-split-bites", r["ok"], r))

    # 变异 1：同一编号出现在两处（r187 的原病）。
    r = check(shell_sec, {"p2.md": ["G2", "G3", "G0"]}, {"G0", "G1", "G2", "G3"})
    legs.append(("duplicate-across-bites", (not r["ok"]) and "G0" in r["duplicates"], r))

    # 变异 2：删掉一条 => 缺失。期望集由调用方显式给出，不靠最大编号自推。
    r = check(shell_sec, {"p2.md": ["G2"]}, {"G0", "G1", "G2", "G3"})
    legs.append(("missing-bites", (not r["ok"]) and "G3" in r["missing"], r))

    # 变异 3：同一文件里同一编号出现两次（收成 set 就会漏掉）。
    sec2 = shell_sec + "| G1 | dup |\n"
    r = check(sec2, {"p2.md": ["G2", "G3"]}, {"G0", "G1", "G2", "G3"})
    legs.append(("duplicate-within-file-bites", (not r["ok"]) and "G1" in r["duplicates"], r))

    # 变异 4：表头行不得被当成数据。表头写成 `| G编号 |` 这种形状最容易被误吞。
    hdr = "## 1. 台账\n\n| G编号 | 任务 |\n|---|---|\n"
    r = check(hdr, {"p2.md": ["G2"]}, {"G2"})
    legs.append(("header-not-data-bites", r["ok"] and r["shell_count"] == 0, r))

    # 变异 5：part 只「提到」编号不算持有 —— 不传 --owner 时它不参与。
    r = check(shell_sec, {}, {"G0", "G1"})
    legs.append(("mention-is-not-owning-bites", r["ok"], r))

    failures = 0
    for name, passed, r in legs:
        print(f"  [{'PASS' if passed else 'FAIL'}] {name}")
        if not passed:
            failures += 1
            print(f"      dup={r['duplicates']} missing={r['missing']} shell={r['shell_count']}")
    print(f"[GATE:part-split-selftest-{'pass' if failures == 0 else 'fail'}] "
          f"{len(legs) - failures} clean / {failures} failing")
    return 0 if failures == 0 else 1


def main() -> int:
    ap = argparse.ArgumentParser(description="分卷编号归属一致性只读判据")
    ap.add_argument("--root", default=str(pathlib.Path(__file__).resolve().parent.parent))
    ap.add_argument("--shell", default="ROADMAP.md", help="主壳文件名（相对 docs/）")
    ap.add_argument("--shell-section", default="## 1.", help="主壳里持有编号的那一节的标题前缀")
    ap.add_argument("--shell-section-end", default="## 2.", help="该节的结束标题前缀")
    ap.add_argument("--owner", action="append", default=[],
                    help="持有 G 编号的 part 文件（相对 docs/，可重复；只被『提到』的不算）")
    ap.add_argument("--expect-range", default="", help="期望编号区间，形如 0-35；不给则跳过缺失检查")
    ap.add_argument("--json", action="store_true")
    ap.add_argument("--selftest", action="store_true")
    args = ap.parse_args()

    if args.selftest:
        return selftest()

    root = pathlib.Path(args.root)
    docs = root / "docs"

    def read(name: str) -> str:
        f = docs / name
        if not f.exists():
            print(f"missing file: {f}", file=sys.stderr)
            sys.exit(2)
        return f.read_text(encoding="utf-8", errors="replace")

    shell_text = read(args.shell)
    sec = section_of(shell_text, args.shell_section, args.shell_section_end)
    owners = {name: _rows(read(name)) for name in args.owner}
    expect = _parse_range(args.expect_range) if args.expect_range else None

    rep = check(sec, owners, expect)

    if args.json:
        print(json.dumps(rep, ensure_ascii=False, indent=2))
    else:
        print(f"主壳 {args.shell} 的 {args.shell_section!r} 一节: {rep['shell_count']} 个编号")
        for n, c in rep["owner_counts"].items():
            print(f"  owner {n}: {c} 个")
        print(f"并集: {rep['union_count']} 个")
        if rep["expect_checked"]:
            print(f"期望: {rep['expect_count']} 个（{args.expect_range}）")
        else:
            print("期望: 未提供 --expect-range ⇒ **本次不做缺失检查**")
        if rep["duplicates"]:
            print("重复编号:")
            for gid, where in rep["duplicates"].items():
                print(f"  {gid} -> {', '.join(where)}")
        if rep["missing"]:
            print(f"缺失编号: {', '.join(rep['missing'])}")
        print(f"[GATE:part-split-{'pass' if rep['ok'] else 'fail'}]"
              + (" 编号完整且唯一；未校验内容是否被改写（见脚本头部声明）" if rep["ok"] else " 详见上方"))

    return 0 if rep["ok"] else 1


if __name__ == "__main__":
    sys.exit(main())