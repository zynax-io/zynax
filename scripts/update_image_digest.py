#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Upsert one image digest in images/images.yaml (ADR-024 source of truth).

Used by the release.yml retag-on-merge job and the tools-image.yml digest
sync (ADR-027 atomic digest commit). Line-based edit — comments and
formatting in images.yaml are preserved, which a YAML round-trip would not
guarantee.

  update_image_digest.py --name api-gateway \\
      --ref ghcr.io/zynax-io/zynax/api-gateway --digest sha256:<64 hex>

If --name has no entry yet, a new one is appended (requires --ref) with an
empty consumers list, so first-time promotions (e.g. a brand-new service)
self-register.

Exit code 0 on success (changed or already current), non-zero on bad input.
"""

from __future__ import annotations

import argparse
import pathlib
import re
import sys

DIGEST_RE = re.compile(r"^sha256:[0-9a-f]{64}$")
NAME_LINE_RE = re.compile(r"^\s*-\s+name:\s+(\S+)\s*$")
DIGEST_LINE_RE = re.compile(r"^(\s*digest:\s*)sha256:[0-9a-f]{64}\s*$")


def upsert(text: str, name: str, ref: str | None, digest: str) -> tuple[str, str]:
    """Return (new_text, action) where action is updated|unchanged|added."""
    lines = text.splitlines(keepends=True)
    out: list[str] = []
    current = None
    found = False
    changed = False
    for line in lines:
        m = NAME_LINE_RE.match(line)
        if m:
            current = m.group(1)
        if current == name:
            found = True
            dm = DIGEST_LINE_RE.match(line)
            if dm:
                new_line = f"{dm.group(1)}{digest}\n"
                if new_line != line:
                    changed = True
                line = new_line
        out.append(line)

    if found:
        return "".join(out), ("updated" if changed else "unchanged")

    if not ref:
        sys.exit(f"error: no entry named {name!r} and --ref not provided")
    if out and not out[-1].endswith("\n"):
        out[-1] += "\n"
    out.append(
        f"\n  - name: {name}\n    ref: {ref}\n    digest: {digest}\n    consumers: []\n"
    )
    return "".join(out), "added"


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--name", required=True, help="entry name (images.yaml key)")
    parser.add_argument("--ref", help="image ref without tag/digest (for new entries)")
    parser.add_argument("--digest", required=True, help="sha256:<64 hex>")
    parser.add_argument("--file", default="images/images.yaml")
    args = parser.parse_args()

    if not DIGEST_RE.match(args.digest):
        sys.exit(f"error: invalid digest {args.digest!r} (want sha256:<64 hex>)")

    path = pathlib.Path(args.file)
    new_text, action = upsert(path.read_text(), args.name, args.ref, args.digest)
    if action != "unchanged":
        path.write_text(new_text)
    print(f"{action}: {args.name} -> {args.digest}")


if __name__ == "__main__":
    main()
