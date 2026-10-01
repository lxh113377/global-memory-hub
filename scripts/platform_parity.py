#!/usr/bin/env python3
"""Cross-repo platform parity (advisory).

Compares this repository's ``platforms.json`` against a private authority's
``truth_constants.json`` on the faces that are actually derivable on both
sides:

  1. the set of active platform ids
  2. per platform, whether the skills face is a *link* and whether the memory
     face is a *link* (a 2-bit shape signature)

Path fields are deliberately exempt. The public side expresses locations as
``~`` / ``%VAR%`` / ``$VAR`` while the private side carries absolute paths, so
comparing them would report the same guaranteed-unequal difference on every
run and bury the real signal.

Deriving the shape on each side
-------------------------------
public  ``platforms.json``   a mount whose ``to`` names the skills root and
                             whose ``kind`` is ``link`` counts as linked
private ``junction_paths``   a non-null ``skills`` / ``memory`` entry counts as
                             linked (``null`` means the face is not a directory
                             link)

Known limitation
----------------
The private authority records ``skills: null`` for both the physical-mirror
platform and the per-skill-tree platform, so it cannot distinguish those two
shapes. This leg therefore compares only the faces above and prints the
underivable distinction as a note rather than pretending to verify it. Closing
that gap requires the private authority to record the mount kind explicitly.

Exit codes
----------
0  both faces agree
1  a difference was found (ids and/or shapes)
2  a side is missing, unreadable or unparseable -- reported as UNVERIFIED,
   never folded into a pass
"""

from __future__ import annotations

import argparse
import json
import os
import sys
from pathlib import Path

EXIT_OK = 0
EXIT_DIFF = 1
EXIT_UNAVAILABLE = 2

SKILLS_TOKEN = "<skills>"
MEMORY_TOKEN = "<memory>"
LINK_KIND = "link"


class Unavailable(Exception):
    """A side could not be measured. Never means 'equal'."""


def read_json(path: Path, *, bom_tolerant: bool = False) -> dict:
    encoding = "utf-8-sig" if bom_tolerant else "utf-8"
    try:
        with path.open("r", encoding=encoding) as handle:
            return json.load(handle)
    except FileNotFoundError as exc:
        raise Unavailable("not found: %s" % path) from exc
    except json.JSONDecodeError as exc:
        raise Unavailable("not valid JSON: %s (%s)" % (path, exc)) from exc
    except OSError as exc:
        raise Unavailable("cannot read %s (%s)" % (path, exc)) from exc


def classify_face(target: str) -> str | None:
    """Map a mount's ``to`` value onto the face it feeds."""
    if not isinstance(target, str):
        return None
    if SKILLS_TOKEN in target:
        return "skills"
    if MEMORY_TOKEN in target:
        return "memory"
    return None


def public_face(doc: dict) -> tuple[set[str], dict[str, tuple[bool, bool]], dict[str, list[str]]]:
    platforms = doc.get("platforms")
    if not isinstance(platforms, list):
        raise Unavailable("public document has no 'platforms' list")

    ids: set[str] = set()
    shapes: dict[str, tuple[bool, bool]] = {}
    kinds: dict[str, list[str]] = {}

    for entry in platforms:
        if not isinstance(entry, dict):
            raise Unavailable("public platform entry is not an object")
        pid = entry.get("id")
        if not isinstance(pid, str) or not pid:
            raise Unavailable("public platform entry without a usable 'id'")
        ids.add(pid)

        linked = {"skills": False, "memory": False}
        observed: list[str] = []
        for mount in entry.get("mounts") or []:
            if not isinstance(mount, dict):
                continue
            kind = mount.get("kind")
            if isinstance(kind, str):
                observed.append(kind)
            face = classify_face(mount.get("to"))
            if face and kind == LINK_KIND:
                linked[face] = True

        shapes[pid] = (linked["skills"], linked["memory"])
        kinds[pid] = sorted(observed)

    return ids, shapes, kinds


def private_face(doc: dict) -> tuple[set[str], dict[str, tuple[bool, bool]]]:
    endpoints = doc.get("endpoints")
    if not isinstance(endpoints, dict):
        raise Unavailable("private document has no 'endpoints' object")
    active = endpoints.get("active")
    if not isinstance(active, list) or not active:
        raise Unavailable("private document has no usable 'endpoints.active'")

    ids = {pid for pid in active if isinstance(pid, str) and pid}

    paths = doc.get("junction_paths")
    if not isinstance(paths, dict):
        raise Unavailable("private document has no 'junction_paths' object")

    shapes: dict[str, tuple[bool, bool]] = {}
    for pid in ids:
        entry = paths.get(pid)
        if not isinstance(entry, dict):
            # A registered endpoint with no junction record is a difference in
            # its own right; surface it rather than defaulting to "linked".
            shapes[pid] = (False, False)
            continue
        shapes[pid] = (bool(entry.get("skills")), bool(entry.get("memory")))

    return ids, shapes


def locate_private_authority(explicit: str | None) -> Path:
    """Resolve the private truth_constants.json without hardcoding a name."""
    if explicit:
        candidate = Path(explicit).expanduser()
        return candidate if candidate.name.endswith(".json") else candidate / "eval" / "truth_constants.json"

    from_env = os.environ.get("FENJUE_PRIVATE")
    if from_env:
        candidate = Path(from_env).expanduser()
        return candidate if candidate.name.endswith(".json") else candidate / "eval" / "truth_constants.json"

    # Generic fallback: look for a sibling directory that carries the file.
    # No directory name is hardcoded, and an ambiguous result is an error
    # rather than a guess.
    try:
        siblings = sorted(Path("..").resolve().iterdir())
    except OSError as exc:
        raise Unavailable("cannot list siblings to locate the private authority (%s)" % exc) from exc

    found = [
        sib / "eval" / "truth_constants.json"
        for sib in siblings
        if sib.is_dir() and (sib / "eval" / "truth_constants.json").is_file()
    ]
    if len(found) == 1:
        return found[0]
    if not found:
        raise Unavailable(
            "no private authority found beside this repository; pass --private <path> "
            "or set FENJUE_PRIVATE"
        )
    raise Unavailable(
        "ambiguous: %d candidates found; pass --private <path> explicitly" % len(found)
    )


def compare(pub_ids, pub_shapes, priv_ids, priv_shapes) -> list[str]:
    problems: list[str] = []

    only_public = sorted(pub_ids - priv_ids)
    only_private = sorted(priv_ids - pub_ids)
    if only_public:
        problems.append("in public but not in private authority: %s" % ", ".join(only_public))
    if only_private:
        problems.append("in private authority but not in public: %s" % ", ".join(only_private))

    for pid in sorted(pub_ids & priv_ids):
        pub = pub_shapes.get(pid)
        priv = priv_shapes.get(pid)
        if pub != priv:
            problems.append(
                "%-4s shape differs: public (skills_linked=%s, memory_linked=%s) "
                "vs private (%s, %s)"
                % (pid, pub[0], pub[1], priv[0], priv[1])
            )

    return problems


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(
        description="Advisory cross-repo platform parity check (ids and mount shapes).",
    )
    parser.add_argument("--hub", default=None, help="this repository's root (default: parent of scripts/)")
    parser.add_argument("--private", default=None, help="private authority root or truth_constants.json path")
    parser.add_argument("--json", action="store_true", help="emit machine-readable output")
    args = parser.parse_args(argv)

    script_dir = Path(__file__).resolve().parent

    try:
        hub_root = Path(args.hub).expanduser().resolve() if args.hub else script_dir.parent
        pub_path = hub_root / "platforms.json"
        priv_path = locate_private_authority(args.private)

        pub_ids, pub_shapes, pub_kinds = public_face(read_json(pub_path))
        priv_ids, priv_shapes = private_face(read_json(priv_path, bom_tolerant=True))
    except Unavailable as exc:
        payload = {
            "status": "UNVERIFIED",
            "reason": str(exc),
            "note": "a side could not be measured; this is not a pass",
        }
        if args.json:
            print(json.dumps(payload, ensure_ascii=False, indent=2))
        else:
            print("PARITY UNVERIFIED: %s" % exc)
            print("NOTE: an unmeasurable side is never reported as equal.")
        return EXIT_UNAVAILABLE

    problems = compare(pub_ids, pub_shapes, priv_ids, priv_shapes)

    # Surfaces the public side distinguishes but the private authority cannot.
    underivable = sorted(
        pid for pid, ks in pub_kinds.items() if "mirror" in ks or "per-skill" in ks
    )

    payload = {
        "status": "DIFF" if problems else "OK",
        "public": str(pub_path),
        "private": str(priv_path),
        "public_ids": sorted(pub_ids),
        "private_ids": sorted(priv_ids),
        "shapes": {pid: {"public": pub_shapes.get(pid), "private": priv_shapes.get(pid)} for pid in sorted(pub_ids | priv_ids)},
        "problems": problems,
        "underivable_shape_distinction": underivable,
        "advisory": True,
    }

    if args.json:
        print(json.dumps(payload, ensure_ascii=False, indent=2))
    else:
        print("public  ids (%d): %s" % (len(pub_ids), ", ".join(sorted(pub_ids))))
        print("private ids (%d): %s" % (len(priv_ids), ", ".join(sorted(priv_ids))))
        if problems:
            print("\nDIFFERS:")
            for line in problems:
                print("  - %s" % line)
        else:
            print("\nPARITY OK: id sets and mount shapes agree.")
        if underivable:
            print(
                "\nNOTE (not a failure): the private authority records no mount kind, so it cannot "
                "distinguish a physical mirror from a per-skill tree. These platforms carry a shape "
                "the public side distinguishes but this leg cannot verify: %s"
                % ", ".join(underivable)
            )
        print("\nADVISORY: this leg reports only; it is not wired into any blocking chain.")

    return EXIT_DIFF if problems else EXIT_OK


if __name__ == "__main__":
    sys.exit(main())
