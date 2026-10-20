#!/usr/bin/env python3
"""build_seed.py - 从本机技能库精选通用技能, 消毒后移植进 agent/internal/seed/pack/skills/。

用法:
  python scripts/build_seed.py [--source <dir>] [--dry-run]
  python scripts/build_seed.py --manifest-only   # 只读重算 pack/manifest.json, 不碰内容
  python scripts/build_seed.py --selftest        # 跑全部规则的正反例, 不依赖 --source

设计铁律
--------
1. 本文件随仓库公开, 因此**禁止写入任何具体个人的字面量**(姓名/学号/用户名)。
   消毒规则一律走**通用启发式**(正则形态), 而不是针对特定个人的硬编码字符串。
   需要更高精度时, 把字面量清单放进 **本地不入库文件** ``scripts/sanitize.local.json``
   (已在 .gitignore 中), 由本脚本自动合并。
2. 身份消毒命中 => 整个技能丢弃; 路径消毒后仍有残留 => SKILL.md 命中丢整技能,
   references 命中只丢该文件; 凭据/危险模式命中 => 命中即丢(供应链安全)。
3. 终检对整包再扫一遍, 任何残留即非零退出 (fail-closed)。
4. 播种内容写入前会生成 ``pack/manifest.json`` (路径 + SHA-256 + 字节数),
   供运行时播种前 fail-closed 校验; manifest 不在 pack/skills 下, 不受重建影响。
"""
from __future__ import annotations

import argparse
import hashlib
import json
import re
import shutil
import sys
from pathlib import Path

REPO = Path(__file__).resolve().parent.parent
DEFAULT_SOURCE = Path("D:/global_skills")
DEST = REPO / "agent" / "internal" / "seed" / "pack" / "skills"
PACK = REPO / "agent" / "internal" / "seed" / "pack"
MANIFEST = PACK / "manifest.json"
LOCAL_RULES = Path(__file__).resolve().parent / "sanitize.local.json"
MAX_FILE_BYTES = 100 * 1024

MANIFEST_SCHEMA = "fenjue-seed-manifest-v1"

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
    # doc-coauthoring 于 2026-10-20 移出：正文与 anthropics/skills 专有件逐字节相同
    # （sha256 2e47d788…，40 字符滑窗 15071/15071=100%），其 LICENSE.txt 明文禁止复制/衍生/分发；
    # 属"引用不分发"档，见 docs/SKILLS-TIERS.md 上游引用档。
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

# ---------------------------------------------------------------------------
# 1. 路径消毒: 正则形态, 对任何用户名/私有根名生效 (不写死字面量)
# ---------------------------------------------------------------------------

# 任意用户主目录 => ~
USER_HOME_PATTERNS = [
    re.compile(r"[A-Za-z]:[\\/]Users[\\/][^\\/\s\"'`]+"),
    re.compile(r"/(?:home|Users)/[^\s/\"'`]+"),
]
# 本项目私有库根 => 产品内的中立位置
PRIVATE_ROOT_MAP = {
    "skills": "~/.fenjue/skills",
    "memory": "~/.fenjue/memory",
}
PRIVATE_ROOT_RE = re.compile(
    r"[A-Za-z]:[\\/](?:[^\s\"'`]+[\\/])*global_(skills|memory)(?=[\\/]|\b)",
    re.IGNORECASE,
)

# ---------------------------------------------------------------------------
# 2. 身份启发式: 只在有明确标识词时才判定为身份信息, 避免误杀版本号/日期/端口
# ---------------------------------------------------------------------------

IDENTITY_KEYWORD = r"(?:学号|工号|证件号|身份证|student\s*id|student\s*number|emp(?:loyee)?\s*id)"
IDENTITY_RE = re.compile(IDENTITY_KEYWORD + r"[^\n]{0,24}?\b\d{6,14}\b", re.IGNORECASE)
# 独立成行的长号码 (单独出现时视为身份信息)
IDENTITY_BARE_RE = re.compile(r"^\s*\d{9,14}\s*$", re.MULTILINE)

# ---------------------------------------------------------------------------
# 3. 残留路径: 消毒后仍存在的绝对路径一律视为残留
# ---------------------------------------------------------------------------

RESIDUE_RE = re.compile(
    # 盘符路径: 斜杠后不得再跟斜杠, 否则 https:// 里的 "s:/" 会被误判成盘符;
    # file:/// 是固定 8 字符前缀, 用等宽否定环视把它整段排除。
    r"(?<!file:///)[A-Za-z]:[\\/](?![\\/])"
    # POSIX 主目录: 前一个字符不得是域名/路径字符, 否则 https://x.com/home/ 会误判。
    r"|(?<![A-Za-z0-9.-])/(?:home|Users|private|var)/"
    r"|(?<![A-Za-z0-9.-])\\\\[A-Za-z0-9_.-]+\\"
)

# ---------------------------------------------------------------------------
# 4. 凭据形态 (供应链安全): 命中即丢
# ---------------------------------------------------------------------------

SECRET_RES = [
    re.compile(r"\bsk-[A-Za-z0-9]{16,}"),
    re.compile(r"\bgh[pousr]_[A-Za-z0-9]{20,}"),
    re.compile(r"\bAKIA[0-9A-Z]{16}\b"),
    re.compile(r"\bAIza[0-9A-Za-z_-]{30,}"),
    re.compile(r"-----BEGIN [A-Z ]*PRIVATE KEY-----"),
    re.compile(r"\bxox[baprs]-[A-Za-z0-9-]{10,}"),
    re.compile(r"\beyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}"),
    re.compile(
        r"(?i)\b(?:api[_-]?key|secret|passwd|password|token|access[_-]?key)\b\s*[:=]\s*[\"']?[A-Za-z0-9_/+=-]{16,}"
    ),
]

# ---------------------------------------------------------------------------
# 5. 危险模式 (供应链安全): 命中即丢
# ---------------------------------------------------------------------------

DANGER_RES = [
    re.compile(r"curl[^\n|]{0,80}\|\s*(?:ba)?sh"),
    re.compile(r"wget[^\n|]{0,80}\|\s*(?:ba)?sh"),
    re.compile(r"rm\s+-[rRf]{1,3}\s+/(\s|$)"),
    re.compile(r"(?i)\b(?:sudo|doas)\s+rm\s+-[rRf]"),
    re.compile(r"(?i)\beval\s*\(\s*(?:base64|atob|Buffer\.from)"),
    re.compile(r"(?i)base64\s+(?:-d|--decode)[^\n|]{0,60}\|\s*(?:ba)?sh"),
    re.compile(r"(?i)\bos\.system\s*\(\s*[\"'][^\"']*rm\s+-rf"),
    re.compile(r"(?i)Remove-Item[^\n]{0,80}-Recurse"),
    re.compile(r"(?i)[A-Za-z]:\\\*[^\n]{0,40}-Recurse[^\n]{0,40}-Force"),
]

RULE_SETS = {
    "identity": [IDENTITY_RE, IDENTITY_BARE_RE],
    "residue": [RESIDUE_RE],
    "secret": SECRET_RES,
    "danger": DANGER_RES,
}


# ---------------------------------------------------------------------------
# 本地增强 (不入库): 允许维护者补字面量以提高精度, 不污染公开面
# ---------------------------------------------------------------------------


def load_local_rules() -> dict:
    if not LOCAL_RULES.is_file():
        return {}
    try:
        data = json.loads(LOCAL_RULES.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as exc:
        print("WARN: ignore unreadable %s (%s)" % (LOCAL_RULES, exc), file=sys.stderr)
        return {}
    return data if isinstance(data, dict) else {}


def effective_literals(local: dict, key: str) -> list[str]:
    out = list(local.get(key) or [])
    return [str(x) for x in out if isinstance(x, (str, int)) and str(x)]


def effective_extra_regex(local: dict, key: str) -> list[re.Pattern]:
    out = []
    for pat in local.get(key) or []:
        try:
            out.append(re.compile(str(pat), re.IGNORECASE))
        except re.error as exc:
            print("WARN: ignore bad regex %r (%s)" % (pat, exc), file=sys.stderr)
    return out


def match_rules(text: str, regexes, literals) -> str | None:
    for rx in regexes:
        m = rx.search(text)
        if m:
            return m.group(0)[:60]
    for lit in literals:
        if lit in text:
            return lit
    return None


def sanitize_text(text: str) -> str:
    for rx in USER_HOME_PATTERNS:
        text = rx.sub("~", text)
    text = PRIVATE_ROOT_RE.sub(lambda m: PRIVATE_ROOT_MAP[m.group(1).lower()], text)
    # 消毒后统一分隔符: 保留反斜杠会产出 ~\notes.md 这种混合形态, 判据与人都难读。
    text = text.replace("~\\", "~/")
    return text


def scan(text: str, local: dict | None = None) -> dict[str, str]:
    """Return {rule_set: matched_snippet} for every rule set that fires."""
    local = local or {}
    hits: dict[str, str] = {}
    for name, regexes in RULE_SETS.items():
        extra = effective_extra_regex(local, "extra_" + name + "_regex")
        found = match_rules(text, list(regexes) + extra, effective_literals(local, name + "_literals"))
        if found is not None:
            hits[name] = found
    return hits


# ---------------------------------------------------------------------------
# manifest (SHA-256): 播种内容的完整性凭据
# ---------------------------------------------------------------------------


def sha256_of(path: Path) -> str:
    h = hashlib.sha256()
    with path.open("rb") as fh:
        for chunk in iter(lambda: fh.read(131072), b""):
            h.update(chunk)
    return h.hexdigest()


def collect_pack_files(pack: Path) -> list[Path]:
    return sorted((p for p in pack.rglob("*") if p.is_file() and p.name != MANIFEST.name), key=lambda p: p.relative_to(pack).as_posix())


def build_manifest(pack: Path, seed_version: str) -> dict:
    files = []
    for p in collect_pack_files(pack):
        files.append(
            {
                "path": p.relative_to(pack).as_posix(),
                "sha256": sha256_of(p),
                "bytes": p.stat().st_size,
            }
        )
    return {
        "schema": MANIFEST_SCHEMA,
        "seed_version": seed_version,
        "files": files,
    }


def write_manifest(pack: Path, seed_version: str) -> int:
    manifest = build_manifest(pack, seed_version)
    payload = json.dumps(manifest, ensure_ascii=False, indent=2, sort_keys=False) + "\n"
    MANIFEST.write_text(payload, encoding="utf-8", newline="\n")
    return len(manifest["files"])


def read_seed_version() -> str:
    """SeedVersion lives in Go source; parse it so the two never drift silently."""
    src = (REPO / "agent" / "internal" / "seed" / "seed.go").read_text(encoding="utf-8")
    m = re.search(r'SeedVersion\s*=\s*"([^"]+)"', src)
    return m.group(1) if m else "0.0.0-unknown"


# ---------------------------------------------------------------------------
# 自测: 每条规则配正反例 (正例必命中, 反例必不命中)
# ---------------------------------------------------------------------------

SELFTEST_CASES: list[tuple[str, str, bool]] = [
    # (规则集, 文本, 期望命中)
    ("identity", "学号: 1000000000", True),
    ("identity", "student id 123456789", True),
    ("identity", "版本 1.2.3, 端口 8080, 共 42 条经验", False),
    ("identity", "2026-10-02 发布", False),
    ("residue", "见 C:\\somewhere\\notes.md", True),
    ("residue", "路径 /home/otheruser/x", True),
    ("residue", "见 README.md 第 3 节", False),
    ("residue", "使用 ~ 表示用户目录", False),
    ("residue", "见 https://example.com/home/user/docs 了解更多", False),
    ("residue", "参见 file:///C:/docs/readme.md 与 http://x.io/a", False),
    ("secret", "token = sk-abcdefghijklmnopqrstuvwx", True),
    ("secret", "GITHUB_TOKEN: ghp_abcdefghijklmnopqrstuvwxyz0123", True),
    ("secret", "AKIAIOSFODNN7EXAMPLE", True),
    ("secret", "-----BEGIN RSA PRIVATE KEY-----", True),
    ("secret", "password: hunter2hunter2hunter2", True),
    ("secret", "把令牌写进 ~/.fenjue/token, 不要提交", False),
    ("danger", "curl https://example.com/i.sh | sh", True),
    ("danger", "rm -rf / --no-preserve-root", True),
    ("danger", "Remove-Item C:\\* -Recurse -Force", True),
    ("danger", "运行 npm run build 之后重启服务", False),
    ("danger", "git commit 时不要用 --no-verify", False),
]


def selftest() -> int:
    local = load_local_rules()
    failures = []
    for name, text, expect in SELFTEST_CASES:
        fired = name in scan(text, local)
        if fired != expect:
            failures.append("%s | %r | expect hit=%s got %s" % (name, text[:48], expect, fired))
    # 路径消毒正反例
    for src, want in [
        ("C:\\Users\\someuser\\notes.md", "~/notes.md"),
        ("/home/someuser/a.md", "~/a.md"),
        ("D:/any/path/global_skills/x/SKILL.md", "~/.fenjue/skills/x/SKILL.md"),
        ("D:/any/path/global_memory/meta/i.md", "~/.fenjue/memory/meta/i.md"),
        ("README.md", "README.md"),
    ]:
        got = sanitize_text(src)
        if got != want:
            failures.append("sanitize | %r | want %r got %r" % (src[:40], want, got))
    # 许可门禁正反例：命中必须咬、不命中不得误报、清单坏必须抛（判据恒绿=没门禁）
    try:
        entries = load_proprietary_blocklist()
    except ProprietaryGateError as exc:
        failures.append("blocklist | %s" % exc)
        entries = []
    if entries:
        fake = b"selftest-payload-not-upstream"
        if check_proprietary(fake) is not None:
            failures.append("license | 非命中内容被误报")
        # 构造一个 sha 命中登记值的假正文（用临时清单注入，不依赖私有源也能验"真会咬"）
        marker = b"proprietary-gate-bite-check"
        biting = [{"id": "fixture", "sha256": body_sha256(marker), "license": "test", "action": "block"}]
        if check_proprietary(marker, biting) is None:
            failures.append("license | 已知专有指纹未被识别（比对逻辑失效）")
        # 反例：只改 frontmatter 不得改变指纹（否则整文件哈希会漏判）
        if body_sha256(b"---\nversion: 1.0.0\n---\nBODY\n") != body_sha256(b"---\nversion: 9.9.9\nlast_updated: x\n---\nBODY\n"):
            failures.append("license | frontmatter 参与了指纹（改版本号即可绕过门禁）")
    else:
        failures.append("blocklist | entries 为空")
    try:
        load_proprietary_blocklist(REPO / "scripts" / "no-such-blocklist.json")
        failures.append("blocklist | 缺失清单被判通过（应抛 ProprietaryGateError）")
    except ProprietaryGateError:
        pass
    if failures:
        for f in failures:
            print("SELFTEST-FAIL %s" % f, file=sys.stderr)
        print("[GATE:seed-selftest-red] %d case(s) failed" % len(failures), file=sys.stderr)
        return 1
    print("[GATE:seed-selftest-pass] %d rule cases + 5 sanitize cases" % len(SELFTEST_CASES))
    return 0


# ---------------------------------------------------------------------------
# 主流程
# ---------------------------------------------------------------------------


def collect_files(skill_dir: Path) -> list[Path]:
    files = [skill_dir / "SKILL.md"]
    refs = skill_dir / "references"
    if refs.is_dir():
        files += sorted(p for p in refs.rglob("*.md") if p.is_file())
    return [p for p in files if p.exists()]


# ---------------------------------------------------------------------------
# 4.5 许可门禁（2026-10-20 立）：专有逐字内容不得进种子包
# ---------------------------------------------------------------------------
# 存在理由：pack/ 随 go:embed 打进 public 仓的发布二进制。此前的门禁只扫身份/凭据/路径，
# 没有任何一条看**许可**，于是 doc-coauthoring 正文（与 Anthropic 专有件逐字节相同）一路绿着
# 进了 CI 与 5 个 release 资产。LICENSE.txt 明文禁止：复制、创建衍生作品、分发/再许可、
# 在 Services 外留存副本。
PROPRIETARY_LIST = REPO / "scripts" / "proprietary_hashes.json"
PROPRIETARY_SCHEMA = "fenjue-proprietary-hash-v1"


class ProprietaryGateError(RuntimeError):
    """判据面不可用（清单缺失/坏/schema 不符/条目空）——绝不折算为通过（R247）。"""


def load_proprietary_blocklist(path: Path = PROPRIETARY_LIST) -> list[dict]:
    try:
        doc = json.loads(Path(path).read_text(encoding="utf-8"))
    except FileNotFoundError as exc:
        raise ProprietaryGateError("proprietary blocklist missing: %s" % path) from exc
    except (OSError, ValueError) as exc:
        raise ProprietaryGateError("proprietary blocklist unreadable: %s (%s)" % (path, exc)) from exc
    if doc.get("schema") != PROPRIETARY_SCHEMA:
        raise ProprietaryGateError("blocklist schema %r, want %r" % (doc.get("schema"), PROPRIETARY_SCHEMA))
    entries = doc.get("entries") or []
    if not entries:
        raise ProprietaryGateError("proprietary blocklist lists no entries (gate would be a no-op)")
    for e in entries:
        if not re.fullmatch(r"[0-9a-f]{64}", str(e.get("sha256") or "")):
            raise ProprietaryGateError("blocklist entry %r has a non-sha256 digest" % e.get("id"))
        if e.get("action", "block") != "block":
            raise ProprietaryGateError("blocklist entry %r has unsupported action %r" % (e.get("id"), e.get("action")))
    return entries


PROPRIETARY_CACHE: list[dict] | None = None


def proprietary_entries(force: bool = False) -> list[dict]:
    """懒加载缓存：任何调用路径（main / selftest / 外部 import）都拿到同一份清单，
    取不到就抛 ProprietaryGateError —— 绝不因未初始化而静默放行。"""
    global PROPRIETARY_CACHE
    if PROPRIETARY_CACHE is None or force:
        PROPRIETARY_CACHE = load_proprietary_blocklist()
    return PROPRIETARY_CACHE


def body_sha256(payload: bytes) -> str:
    """正文指纹口径：去 frontmatter + 统一行尾后取 sha256（与登记侧同一函数，防两把尺）。"""
    return hashlib.sha256(strip_frontmatter_for_hash(payload)).hexdigest()


def check_proprietary(payload: bytes, entries: list[dict] | None = None) -> dict | None:
    """命中返回该条目，否则 None。

    哈希在函数内部自算（调用方传不进"配好的 sha"，否则判据可被喂假 sha 绕过）；
    比对按内容不按文件名——改名/挪路径都绕不过。
    """
    want = body_sha256(payload)
    for e in (entries if entries is not None else proprietary_entries()):
        if e["sha256"].lower() == want:
            return e
    return None


def scan_pack_proprietary(pack_dir: Path, blocklist_path: Path = PROPRIETARY_LIST) -> list[str]:
    """整包扫（含 manifest 自身以外的所有文件），返回命中的 '路径 == 条目id' 列表。"""
    entries = load_proprietary_blocklist(blocklist_path)
    offenders: list[str] = []
    for p in sorted(Path(pack_dir).rglob("*")):
        if not p.is_file() or p.name == "manifest.json":
            continue
        hit = check_proprietary(p.read_bytes(), entries)
        _digest = None
        if hit:
            offenders.append("%s == %s (%s)" % (p.relative_to(pack_dir), hit["id"], hit["license"]))
    return offenders


def strip_frontmatter_for_hash(raw: bytes) -> bytes:
    """按正文（去掉 YAML frontmatter、统一行尾与空白）取指纹。

    为什么不能整文件哈希：本地件与上游只差 frontmatter（version/last_updated），
    整文件哈希会因此漏判。这里与实测口径（40 字符滑窗）同向——比的是正文本体。
    """
    s = raw.decode("utf-8", errors="replace")
    m = re.match(r"^\s*---\s*\n.*?\n---\s*\n", s, re.S)
    body = s[m.end():] if m else s
    body = body.replace("\r\n", "\n").replace("\r", "\n").replace("\ufeff", "")
    return body.strip().lstrip("\ufeff").encode("utf-8")


def stage_skill(name: str, skill_dir: Path, local: dict) -> tuple[list[tuple[Path, str]], str | None]:
    staged: list[tuple[Path, str]] = []
    for f in collect_files(skill_dir):
        try:
            raw = f.read_bytes()
        except OSError as exc:
            return [], "read %s: %s" % (f.name, exc)
        if len(raw) > MAX_FILE_BYTES:
            if f.name == "SKILL.md":
                return [], "SKILL.md too large (%dB)" % len(raw)
            continue
        try:
            text = raw.decode("utf-8")
        except UnicodeDecodeError:
            if f.name == "SKILL.md":
                return [], "SKILL.md not utf-8"
            continue
        # 许可门禁：正文级 sha256 命中专有清单 ⇒ 整件丢弃（改名/改路径都绕不过内容比对）
        phit = check_proprietary(raw)
        if phit is not None:
            return [], "proprietary-verbatim (%s): %s" % (phit["id"], phit["license"])
        hits = scan(text, local)
        if hits:
            reason = ", ".join("%s=%r" % (k, v) for k, v in sorted(hits.items()))
            if f.name == "SKILL.md" or set(hits) & {"identity", "secret", "danger"}:
                return [], reason + " in " + f.name
            continue
        staged.append((f, sanitize_text(text)))
    return staged, None


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--source", default=str(DEFAULT_SOURCE))
    ap.add_argument("--dry-run", action="store_true")
    ap.add_argument("--selftest", action="store_true")
    ap.add_argument("--manifest-only", action="store_true")
    args = ap.parse_args()

    if args.selftest:
        return selftest()

    seed_version = read_seed_version()

    if args.manifest_only:
        if not PACK.is_dir():
            print("pack dir missing: %s" % PACK, file=sys.stderr)
            return 1
        n = write_manifest(PACK, seed_version)
        print("manifest written: %d files, seed_version=%s" % (n, seed_version))
        return 0

    source = Path(args.source)
    local = load_local_rules()
    if not DEST.exists() and not args.dry_run:
        DEST.mkdir(parents=True)

    picked, dropped = [], []
    proprietary_entries(force=True)  # 取不到即抛 ProprietaryGateError（不静默放行）
    for name in CANDIDATES:
        skill_dir = source / name
        if not (skill_dir / "SKILL.md").exists():
            dropped.append((name, "not found in source"))
            continue
        staged, reason = stage_skill(name, skill_dir, local)
        if reason is not None:
            dropped.append((name, reason))
            continue
        picked.append((name, staged))

    if args.dry_run:
        print("picked=%d dropped=%d" % (len(picked), len(dropped)))
        for n, r in dropped:
            print("  DROP %s: %s" % (n, r))
        return 0

    if DEST.exists():
        shutil.rmtree(DEST)
    DEST.mkdir(parents=True)

    for name, staged in picked:
        out_dir = DEST / name
        for src_file, text in staged:
            target = out_dir / src_file.relative_to(source / name)
            target.parent.mkdir(parents=True, exist_ok=True)
            target.write_text(text, encoding="utf-8", newline="\n")

    print("seeded %d skills into %s" % (len(picked), DEST))
    for n, _ in picked:
        cnt = sum(1 for _ in (DEST / n).rglob("*.md"))
        print("  OK %s (%d files)" % (n, cnt))
    for n, r in dropped:
        print("  DROP %s: %s" % (n, r))

    # 终检: 整包扫描, 任何残留即失败退出 (fail-closed)
    bad = []
    for p in PACK.rglob("*.md"):
        text = p.read_text(encoding="utf-8", errors="replace")
        hits = scan(text, local)
        if hits:
            bad.append((p.relative_to(PACK).as_posix(), hits))
    if bad:
        for rel, hits in bad:
            print("RESIDUE %s: %s" % (rel, hits), file=sys.stderr)
        return 1
    print("final scan: 0 residue hits [GATE:seed-clean]")

    n = write_manifest(PACK, seed_version)
    print("manifest written: %d files, seed_version=%s" % (n, seed_version))
    return 0


if __name__ == "__main__":
    sys.exit(main())
