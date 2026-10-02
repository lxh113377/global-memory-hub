#!/usr/bin/env python3
"""check_preset_config.py - 预设配置判据。

背景
----
命名预设 (platforms.json 的 presets 段) 是一次动多个端的动作, 因此**配置写错的后果
比单个端大得多**: 少写一个端 id, 用户会以为全开了; 写错一个 id, 那一端在批量动作里
静默缺席。本脚本把这类错误变成 CI 里的一条腿。

判据分层
--------
1. **平台段 (基础)**: id 唯一、id 非空。预设的成员引用必须落在这里, 所以先查。
2. **预设段 (结构)**: name 非空且唯一; platforms 非空; 成员 id 必须存在于平台段;
   同一预设内成员不重复。
3. **提示级 (不判红)**: 成员里含 support=beta 的端, 或 label 缺省。这两条都不是错误,
   但值得让人看见, 所以打印出来而不影响退出码。

为什么与 Go 侧的实现分开写: platform.parsePresets 已经做了同样的校验, 这里用 Python
独立再写一遍, 两边任一漏掉都能被另一边的测试抓到 (自比恒等不算验证)。

退出码: 0 通过 (可能有提示) / 1 配置有问题 / 2 无法测量 (UNVERIFIED)
"""
from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

REPO = Path(__file__).resolve().parent.parent
PLATFORMS = REPO / "platforms.json"

EXIT_OK = 0
EXIT_DIFF = 1
EXIT_UNAVAILABLE = 2

MAX_PRESET_MEMBERS = 64


def check(doc: dict) -> tuple[list[str], list[str]]:
    """纯函数, 便于自测驱动。返回 (problems, notes)。"""
    problems: list[str] = []
    notes: list[str] = []

    raw_platforms = doc.get("platforms")
    if not isinstance(raw_platforms, list) or not raw_platforms:
        return ["platforms must be a non-empty array"], notes

    # 1. 平台段
    ids: list[str] = []
    for i, p in enumerate(raw_platforms):
        if not isinstance(p, dict):
            problems.append("platforms[%d] is not an object" % i)
            continue
        pid = str(p.get("id", "")).strip()
        if not pid:
            problems.append("platforms[%d] has an empty id" % i)
            continue
        if pid in ids:
            problems.append("duplicate platform id '%s' (a preset referencing it would be ambiguous)" % pid)
        ids.append(pid)
    if problems:
        # 平台段本身就坏了, 继续查预设只会产生噪声。
        return problems, notes

    idset = set(ids)
    support = {str(p.get("id", "")).strip(): str(p.get("support", "")) for p in raw_platforms}

    # 2. 预设段。缺省 (没有 presets) 是合法状态, 不是错误。
    presets = doc.get("presets")
    if presets is None:
        notes.append("no presets section: batch toggle is unavailable, which is a valid configuration")
        return problems, notes
    if not isinstance(presets, list):
        return problems + ["presets must be an array when present"], notes

    seen_names: set[str] = set()
    for i, pr in enumerate(presets):
        if not isinstance(pr, dict):
            problems.append("presets[%d] is not an object" % i)
            continue
        name = str(pr.get("name", "")).strip()
        if not name:
            problems.append("presets[%d] has an empty name" % i)
            continue
        if name in seen_names:
            problems.append("presets[%d]: duplicate preset name '%s'" % (i, name))
        seen_names.add(name)

        members = pr.get("platforms")
        if not isinstance(members, list) or not members:
            problems.append("preset '%s': platforms must be a non-empty array" % name)
            continue
        if len(members) > MAX_PRESET_MEMBERS:
            problems.append("preset '%s': %d members exceeds the sanity limit of %d; a batch this large is probably a mistake"
                            % (name, len(members), MAX_PRESET_MEMBERS))

        inner: set[str] = set()
        for raw_id in members:
            pid = str(raw_id).strip()
            if not pid:
                problems.append("preset '%s': contains an empty platform id" % name)
                continue
            if pid not in idset:
                problems.append("preset '%s' references unknown platform '%s'; a batch action would silently skip it"
                                % (name, pid))
                continue
            if pid in inner:
                problems.append("preset '%s' lists '%s' twice" % (name, pid))
                continue
            inner.add(pid)
            if support.get(pid) == "beta":
                notes.append("preset '%s' includes beta platform '%s' (consumption behavior not verified)" % (name, pid))
        if not str(pr.get("label", "")).strip():
            notes.append("preset '%s' has no label; the console will show the raw name" % name)
        if not str(pr.get("note", "")).strip():
            notes.append("preset '%s' has no note; batch actions deserve a one-line reason" % name)

    return problems, notes


def selftest() -> int:
    """每条规则都要有一个真会红的反例, 外加一个干净正例。"""
    base = {"schema": "fenjue-platforms-v1", "version": "1.1.0",
            "platforms": [{"id": "a", "support": "ga"}, {"id": "b", "support": "beta"}]}

    def doc_with(presets):
        d = dict(base)
        d["presets"] = presets
        return d

    failures: list[str] = []

    clean_problems, clean_notes = check(doc_with([
        {"name": "core", "label": "Core", "note": "why", "platforms": ["a", "b"]},
    ]))
    if clean_problems:
        failures.append("a valid config must produce no problem, got: %r" % clean_problems)
    if not any("beta platform 'b'" in n for n in clean_notes):
        failures.append("including a beta platform must produce a note (not a silent pass), got: %r" % clean_notes)

    cases: list[tuple[str, list[str]]] = [
        ("unknown member",
         check(doc_with([{"name": "p", "platforms": ["a", "ghost"]}]))[0]),
        ("duplicate member",
         check(doc_with([{"name": "p", "platforms": ["a", "a"]}]))[0]),
        ("empty name",
         check(doc_with([{"platforms": ["a"]}]))[0]),
        ("duplicate preset name",
         check(doc_with([{"name": "p", "platforms": ["a"]}, {"name": "p", "platforms": ["b"]}]))[0]),
        ("empty member list",
         check(doc_with([{"name": "p", "platforms": []}]))[0]),
        ("empty member id",
         check(doc_with([{"name": "p", "platforms": [" "]}]))[0]),
        ("oversized batch",
         check(doc_with([{"name": "p", "platforms": ["a", "b"] * 40}]))[0]),
        ("duplicate platform id",
         check({"platforms": [{"id": "a"}, {"id": "a"}]})[0]),
        ("platforms not an array",
         check({"platforms": {}})[0]),
        ("presets not an array",
         check(doc_with({"name": "p"}))[0]),
    ]
    for label, got in cases:
        if not got:
            failures.append("rule must bite but produced nothing: %s" % label)

    # 缺省 presets 是合法配置, 且要给出提示。
    no_presets_problems, no_presets_notes = check(base)
    if no_presets_problems:
        failures.append("a config without presets must stay valid, got: %r" % no_presets_problems)
    if not no_presets_notes:
        failures.append("a config without presets must say so in the notes")

    if failures:
        for f in failures:
            print("SELFTEST-FAIL %s" % f, file=sys.stderr)
        print("[GATE:preset-selftest-red] %d case(s) failed" % len(failures), file=sys.stderr)
        return 1
    print("[GATE:preset-selftest-pass] 10 rules bite + 2 clean cases (with presets / without presets)")
    return 0


def main() -> int:
    ap = argparse.ArgumentParser(description="Validate the presets section of platforms.json.")
    ap.add_argument("--json", action="store_true", help="emit machine-readable output")
    ap.add_argument("--selftest", action="store_true", help="run the rule self-test and exit")
    args = ap.parse_args()

    if args.selftest:
        return selftest()

    if not PLATFORMS.exists():
        print("cannot measure: %s not found (UNVERIFIED, never counted as a pass)" % PLATFORMS, file=sys.stderr)
        print("[GATE:preset-unverified]", file=sys.stderr)
        return EXIT_UNAVAILABLE
    try:
        doc = json.loads(PLATFORMS.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as exc:
        print("cannot measure: cannot parse %s: %s (UNVERIFIED, never counted as a pass)" % (PLATFORMS, exc),
              file=sys.stderr)
        print("[GATE:preset-unverified]", file=sys.stderr)
        return EXIT_UNAVAILABLE
    if not isinstance(doc, dict):
        print("cannot measure: %s is not a JSON object (UNVERIFIED)" % PLATFORMS, file=sys.stderr)
        print("[GATE:preset-unverified]", file=sys.stderr)
        return EXIT_UNAVAILABLE

    problems, notes = check(doc)
    payload = {
        "status": "DIFF" if problems else "OK",
        "platforms": len(doc.get("platforms") or []),
        "presets": len(doc.get("presets") or []),
        "problems": problems,
        "notes": notes,
    }
    if args.json:
        print(json.dumps(payload, ensure_ascii=False, indent=2))
    else:
        print("platforms: %d   presets: %d" % (payload["platforms"], payload["presets"]))
        for n in notes:
            print("  NOTE  %s" % n)
        if problems:
            print("\nPROBLEMS:")
            for p in problems:
                print("  - %s" % p)
        else:
            print("\nPRESET CONFIG OK: every preset resolves to existing platforms with no duplicate members.")

    if problems:
        print("[GATE:preset-red]", file=sys.stderr)
        return EXIT_DIFF
    return EXIT_OK


if __name__ == "__main__":
    sys.exit(main())