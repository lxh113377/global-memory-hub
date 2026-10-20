#!/usr/bin/env python3
"""test_proprietary_gate.py — 种子包许可门禁的 TDD 证据。

判据存在理由：agent/internal/seed/pack/ 随 go:embed 打进**公开仓的发布二进制**。
2026-10-20 实测 pack 内 doc-coauthoring/SKILL.md 正文与 Anthropic 专有件逐字节相同
（正文 sha256 三侧一致；40 字符滑窗 15071/15071 = 100%），而其 LICENSE.txt 明文禁止
复制 / 创建衍生作品 / 分发再许可 / 在 Services 外留存副本。此前门禁只扫身份、凭据、
路径残留，**没有任何一条看许可**，所以它能一路绿着进 CI、进 5 个 release 资产。

本文件锁三件事：
  1. 门禁正反例都有（只测正例=判据恒绿，等价于没有门禁）；
  2. 指纹口径不能被绕过（改 frontmatter、调用方自喂 sha 都不行）；
  3. 现网 pack 干净 + 被登记件不得留在 CANDIDATES 里蒙混。
"""
import sys
import unittest
from pathlib import Path

REPO = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(REPO / "scripts"))

import build_seed  # noqa: E402

LIST = REPO / "scripts" / "proprietary_hashes.json"


def _fixture_entry(payload: bytes) -> dict:
    return {"id": "fixture", "sha256": build_seed.body_sha256(payload),
            "license": "test fixture", "action": "block"}


class GateBehaviour(unittest.TestCase):
    def test_blocklist_shape(self):
        entries = build_seed.load_proprietary_blocklist(LIST)
        self.assertGreaterEqual(len(entries), 1)
        for e in entries:
            self.assertRegex(e["sha256"], r"^[0-9a-f]{64}$")
            self.assertEqual(e.get("action"), "block")
            self.assertTrue(e.get("license"), "条目必须写明为什么不能分发")

    def test_known_digest_bites(self):
        """正例：登记过的正文必须被识别（含 scan_pack 真咬一次，防函数只比不报）。"""
        payload = "GATE-MUST-BITE 专有正文样本\n".encode("utf-8")
        entry = _fixture_entry(payload)
        self.assertIsNotNone(build_seed.check_proprietary(payload, [entry]))
        tmp = Path(__file__).parent / "_tmp_pack"
        (tmp / "skills" / "x").mkdir(parents=True, exist_ok=True)
        (tmp / "skills" / "x" / "SKILL.md").write_bytes(payload)
        bad_list = tmp / "blocklist.json"
        bad_list.write_text('{"schema": "fenjue-proprietary-hash-v1", "entries": ['
                            + __import__("json").dumps(entry, ensure_ascii=False) + "]}")
        try:
            offenders = build_seed.scan_pack_proprietary(tmp, bad_list)
            self.assertEqual(len(offenders), 1, offenders)
        finally:
            import shutil
            shutil.rmtree(tmp, ignore_errors=True)

    def test_non_matching_payload_not_flagged(self):
        self.assertIsNone(build_seed.check_proprietary(b"totally our own text"))

    def test_frontmatter_edit_cannot_escape(self):
        """绕过测试①：只改 frontmatter（版本号/日期）与行尾不得改变指纹。"""
        a = b"---\nversion: 1.0.0\n---\nBODY LINE\n"
        b = b"---\nversion: 9.9.9\nlast_updated: 2099-01-01\n---\nBODY LINE\r\n"
        self.assertEqual(build_seed.body_sha256(a), build_seed.body_sha256(b))

    def test_hash_is_computed_internally(self):
        """绕过测试②：调用方不能自己传 sha（否则喂个假 sha 就绿了）。"""
        import inspect
        sig = list(inspect.signature(build_seed.check_proprietary).parameters)
        self.assertEqual(sig, ["payload", "entries"], sig)

    def test_missing_blocklist_fails_closed(self):
        with self.assertRaises(build_seed.ProprietaryGateError):
            build_seed.load_proprietary_blocklist(REPO / "scripts" / "no-such.json")

    def test_empty_blocklist_fails_closed(self):
        bad = Path(__file__).parent / "_tmp_empty_blocklist.json"
        bad.write_text('{"schema": "fenjue-proprietary-hash-v1", "entries": []}',
                       encoding="utf-8")
        try:
            with self.assertRaises(build_seed.ProprietaryGateError):
                build_seed.load_proprietary_blocklist(bad)
        finally:
            bad.unlink()

    def test_wrong_schema_fails_closed(self):
        bad = Path(__file__).parent / "_tmp_wrong_schema.json"
        bad.write_text('{"schema": "other", "entries": [{"sha256": "' + "a" * 64 + '"}]}',
                       encoding="utf-8")
        try:
            with self.assertRaises(build_seed.ProprietaryGateError):
                build_seed.load_proprietary_blocklist(bad)
        finally:
            bad.unlink()


class CurrentStateIsClean(unittest.TestCase):
    def test_pack_has_no_blocked_bytes(self):
        offenders = build_seed.scan_pack_proprietary(
            REPO / "agent" / "internal" / "seed" / "pack", LIST)
        self.assertEqual(offenders, [], "种子包内含专有逐字内容: %s" % offenders)

    def test_blocked_skill_not_in_candidates(self):
        for e in build_seed.load_proprietary_blocklist(LIST):
            self.assertNotIn("doc-coauthoring", build_seed.CANDIDATES,
                             "doc-coauthoring 已登记为专有件，却仍在 CANDIDATES 里")

    def test_upstream_reference_tier_documents_it(self):
        doc = (REPO / "docs" / "SKILLS-TIERS.md").read_text(encoding="utf-8")
        self.assertIn("doc-coauthoring", doc, "移出后必须在上游引用档登记来源")


if __name__ == "__main__":
    unittest.main(verbosity=2)
