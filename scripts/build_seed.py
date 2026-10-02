#!/usr/bin/env python3
"""build_seed.py - 从本机技能库精选通用技能, 消毒后移植进 agent/internal/seed/pack/skills/。

用法: python scripts/build_seed.py [--source D:/global_skills] [--dry-run]
规则:
  - 只复制 SKILL.md 与 references/**/*.md (跳过 scripts/ assets/ 二进制, 供给链安全)
  - 路径消毒: C:\\Users\\37533 -> ~ ; D:\\global_memory -> ~/.fenjue/memory ; D:\\global_skills -> ~/.fenjue/skills
  - 身份消毒命中(姓名/学号/称呼) => 整个技能丢弃并记录
  - 消毒后残留个人路径模式 => 该文件丢弃; SKILL.md 被丢 => 整个技能丢弃
  - 单文件 >100KB 丢弃; 目标已存在时先清空重建(本脚本是种子包唯一写入口)
"""
import argparse
import re
import shutil
import sys
from pathlib import Path

REPO = Path(__file__).resolve().parent.parent
DEFAULT_SOURCE = Path("D:/global_skills")
DEST = REPO / "agent" / "internal" / "seed" / "pack" / "skills"
MAX_FILE_BYTES = 100 * 1024

# 精选候选: 通用方法论类, 与具体端/个人工作流无关。命中身份消毒即被丢弃, 属预期筛选。
CANDIDATES = [
    "brainstorming",
    "first-principles-decomposer",
    "verification-before-completion",
    "test-driven-development",
    "refactoring",
    "debugging-fixing",
    "code-review",
    "writing-plans",
    "doc-coauthoring",
    "knowledge-capture",
    "data-analysis",
    "research-documentation",
    "spec-to-implementation",
    "scope-creep-detector",
    "consulting-analysis",
    "prompt-consolidation",
    "diagram-maker",
    "frontend-design",
]

# 路径消毒映射 (先后有序: 长路径在前)
PATH_RULES = [
    ("C:\\Users\\37533", "~"),
    ("C:/Users/37533", "~"),
    ("D:\\global_memory", "~/.fenjue/memory"),
    ("D:/global_memory", "~/.fenjue/memory"),
    ("D:\\global_skills", "~/.fenjue/skills"),
    ("D:/global_skills", "~/.fenjue/skills"),
]

# 身份消毒命中 => 整技能丢弃
IDENTITY_PATTERNS = [
    "刘星辉", "202599010324", "37533", "辉哥", "龙虾",
]

# 消毒后残留 => 单文件丢弃
RESIDUE_PATTERNS = [
    "C:\\Users", "C:/Users", "D:\\global", "D:/global",
]


def sanitize_text(text: str) -> str:
    for old, new in PATH_RULES:
        text = text.replace(old, new)
    return text


def hit_any(text: str, patterns) -> str | None:
    for p in patterns:
        if p in text:
            return p
    return None


def collect_files(skill_dir: Path) -> list[Path]:
    files = [skill_dir / "SKILL.md"]
    refs = skill_dir / "references"
    if refs.is_dir():
        files += sorted(p for p in refs.rglob("*.md") if p.is_file())
    return [p for p in files if p.exists()]


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--source", default=str(DEFAULT_SOURCE))
    ap.add_argument("--dry-run", action="store_true")
    args = ap.parse_args()
    source = Path(args.source)

    if not DEST.exists() and not args.dry_run:
        DEST.mkdir(parents=True)

    picked, dropped = [], []
    for name in CANDIDATES:
        skill_dir = source / name
        if not (skill_dir / "SKILL.md").exists():
            dropped.append((name, "not found in source"))
            continue
        reason = None
        staged: list[tuple[Path, str]] = []
        for f in collect_files(skill_dir):
            try:
                raw = f.read_bytes()
            except OSError as e:
                reason = f"read {f.name}: {e}"
                break
            if len(raw) > MAX_FILE_BYTES:
                if f.name == "SKILL.md":
                    reason = f"SKILL.md too large ({len(raw)}B)"
                    break
                continue  # 大引用文件直接跳过
            try:
                text = raw.decode("utf-8")
            except UnicodeDecodeError:
                if f.name == "SKILL.md":
                    reason = f"SKILL.md not utf-8"
                    break
                continue
            ident = hit_any(text, IDENTITY_PATTERNS)
            if ident is not None:
                reason = f"identity pattern {ident!r} in {f.name}"
                break
            text = sanitize_text(text)
            residue = hit_any(text, RESIDUE_PATTERNS)
            if residue is not None:
                if f.name == "SKILL.md":
                    reason = f"residue {residue!r} in SKILL.md"
                    break
                continue  # 引用文件残留则丢该文件
            staged.append((f, text))
        if reason is not None:
            dropped.append((name, reason))
            continue
        picked.append((name, staged))

    if args.dry_run:
        print(f"picked={len(picked)} dropped={len(dropped)}")
        for n, r in dropped:
            print(f"  DROP {n}: {r}")
        return 0

    if DEST.exists():
        shutil.rmtree(DEST)
    DEST.mkdir(parents=True)

    for name, staged in picked:
        out_dir = DEST / name
        for src_file, text in staged:
            rel = src_file.relative_to(source / name)
            target = out_dir / rel
            target.parent.mkdir(parents=True, exist_ok=True)
            target.write_text(text, encoding="utf-8", newline="\n")

    print(f"seeded {len(picked)} skills into {DEST}")
    for n, _ in picked:
        cnt = sum(1 for _ in (DEST / n).rglob("*.md"))
        print(f"  OK {n} ({cnt} files)")
    for n, r in dropped:
        print(f"  DROP {n}: {r}")

    # 终检: 整包扫描, 任何残留即失败退出 (fail-closed)
    bad = []
    for p in DEST.rglob("*.md"):
        text = p.read_text(encoding="utf-8", errors="replace")
        pat = hit_any(text, IDENTITY_PATTERNS + RESIDUE_PATTERNS)
        if pat:
            bad.append((str(p), pat))
    if bad:
        for p, pat in bad:
            print(f"RESIDUE {p}: {pat}", file=sys.stderr)
        return 1
    print("final scan: 0 residue hits [GATE:seed-clean]")
    return 0


if __name__ == "__main__":
    sys.exit(main())
