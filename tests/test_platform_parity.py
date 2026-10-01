#!/usr/bin/env python3
"""Mutation legs for scripts/platform_parity.py.

Runs against two *synthetic* documents built inside a temporary directory.
Nothing here reads the real repositories, the network, or any machine path, so
the fixture behaves identically on a developer box and on a clean CI runner.

The point of the fixture is direction: a parity leg that has never been seen to
fail proves nothing, so each injected difference must turn the leg red, and
removing it must turn the leg green again.

Run directly:

    python tests/test_platform_parity.py

or under pytest (a single test case drives every leg).
"""

from __future__ import annotations

import json
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[1]
LEG = REPO_ROOT / "scripts" / "platform_parity.py"

EXIT_OK = 0
EXIT_DIFF = 1
EXIT_UNAVAILABLE = 2

HUB_DOC = {
    "schema": "fenjue-platforms-v1",
    "version": "0.0.0-fixture",
    "roots": {"memory": "~/.fenjue/memory", "skills": "~/.fenjue/skills"},
    "platforms": [
        {
            "id": "aa",
            "label": "Synthetic Linked",
            "mounts": [
                {"kind": "link", "from": "~/aa/skills", "to": "<skills>"},
                {"kind": "link", "from": "~/aa/memory", "to": "<memory>"},
            ],
        },
        {
            "id": "bb",
            "label": "Synthetic Mirror",
            "mounts": [
                {"kind": "mirror", "from": "~/bb/skills", "to": "<skills>"},
                {"kind": "link", "from": "~/bb/memory", "to": "<memory>"},
            ],
        },
    ],
}

PRIV_DOC = {
    "_meta": "synthetic",
    "endpoints": {"active": ["aa", "bb"]},
    "junction_paths": {
        "aa": {"skills": "/synthetic/aa/skills", "memory": "/synthetic/aa/memory"},
        "bb": {"skills": None, "memory": "/synthetic/bb/memory"},
    },
}


def _write(path: Path, doc) -> None:
    path.write_text(json.dumps(doc, ensure_ascii=False, indent=2), encoding="utf-8")


def _run(hub_dir: Path, private_arg: str):
    proc = subprocess.run(
        [sys.executable, str(LEG), "--hub", str(hub_dir), "--private", private_arg],
        capture_output=True,
        text=True,
    )
    return proc.returncode, (proc.stdout or "") + (proc.stderr or "")


def _mutate_drop_platform(doc):
    doc["platforms"] = [p for p in doc["platforms"] if p["id"] != "bb"]


def _mutate_flip_skills_kind(doc):
    for p in doc["platforms"]:
        if p["id"] == "aa":
            for m in p["mounts"]:
                if "<skills>" in m.get("to", ""):
                    m["kind"] = "mirror"


def _mutate_drop_junction_record(doc):
    doc["junction_paths"].pop("aa", None)


CASES = [
    # name, hub mutation, private mutation, expected exit code
    ("control-unchanged", None, None, EXIT_OK),
    ("hub-missing-platform", _mutate_drop_platform, None, EXIT_DIFF),
    ("hub-skills-kind-flipped", _mutate_flip_skills_kind, None, EXIT_DIFF),
    ("private-missing-junction-record", None, _mutate_drop_junction_record, EXIT_DIFF),
]


def run_all() -> int:
    work = Path(tempfile.mkdtemp(prefix="parity-fixture-"))
    failures = 0
    total = 0
    try:
        private_dir = work / "private"
        (private_dir / "eval").mkdir(parents=True)

        for name, hub_mut, priv_mut, expect in CASES:
            total += 1
            hub_doc = json.loads(json.dumps(HUB_DOC))
            priv_doc = json.loads(json.dumps(PRIV_DOC))
            if hub_mut:
                hub_mut(hub_doc)
            if priv_mut:
                priv_mut(priv_doc)

            hub_dir = work / name
            hub_dir.mkdir(parents=True)
            _write(hub_dir / "platforms.json", hub_doc)
            _write(private_dir / "eval" / "truth_constants.json", priv_doc)

            rc, out = _run(hub_dir, str(private_dir))
            ok = rc == expect
            failures += 0 if ok else 1
            print("%-34s expect rc=%d  got rc=%d  %s" % (name, expect, rc, "PASS" if ok else "FAIL"))
            if not ok:
                print("    output: %s" % out.strip().replace("\n", "\n    "))

        # An unmeasurable side must never be folded into a pass.
        total += 1
        hub_dir = work / "unavailable"
        hub_dir.mkdir(parents=True)
        _write(hub_dir / "platforms.json", HUB_DOC)
        rc, out = _run(hub_dir, str(work / "no-such-private"))
        ok = rc == EXIT_UNAVAILABLE
        failures += 0 if ok else 1
        print("%-34s expect rc=%d  got rc=%d  %s"
              % ("unavailable-side", EXIT_UNAVAILABLE, rc, "PASS" if ok else "FAIL"))
        if not ok:
            print("    output: %s" % out.strip().replace("\n", "\n    "))
    finally:
        shutil.rmtree(work, ignore_errors=True)

    print("\n%d legs, %d failed" % (total, failures))
    return 0 if failures == 0 else 1


def test_platform_parity_legs():
    assert run_all() == 0


if __name__ == "__main__":
    sys.exit(run_all())
