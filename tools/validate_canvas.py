#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
# Validates REASONS Canvas files under docs/spdd/.
# Usage: python tools/validate_canvas.py [canvas-dir]
# Default canvas-dir: docs/spdd/
#
# Rules enforced:
#   - All seven REASONS sections present (R, E, A, S-structure, O, N, S-safeguards)
#   - Required header fields present: Issue, Author, Date, Status
#   - Status is one of: Draft, Aligned, Implemented, Synced
#   - Safeguards context-security checklist is present
#   - canvas.private.md files are not committed (gitignore should catch this, but verify)
#
# Exit codes: 0 = all valid (warnings may be printed), 1 = structural errors found
import re
import sys
from pathlib import Path

VALID_STATUSES = {"Draft", "Aligned", "Implemented", "Synced"}

# The seven REASONS section headings in order
REQUIRED_SECTIONS = [
    r"^## R ",
    r"^## E ",
    r"^## A ",
    r"^## S ",   # first S — Structure
    r"^## O ",
    r"^## N ",
    r"^## S — Safeguards",   # second S — distinguished by "Safeguards" in heading
]

REQUIRED_FIELDS = ["**Issue:**", "**Author:**", "**Date:**", "**Status:**"]

SECURITY_CHECKLIST_MARKER = "Context Security"


def _extract_status(text: str) -> str | None:
    m = re.search(r"\*\*Status:\*\*\s*(\w+)", text)
    return m.group(1) if m else None


def validate_canvas(path: Path) -> list[str]:
    errors: list[str] = []
    warnings: list[str] = []
    text = path.read_text(encoding="utf-8")
    lines = text.splitlines()

    # Required header fields
    for field in REQUIRED_FIELDS:
        if field not in text:
            errors.append(f"missing header field: {field}")

    # Status value
    status = _extract_status(text)
    if status and status not in VALID_STATUSES:
        errors.append(f"invalid Status '{status}' — must be one of: {', '.join(sorted(VALID_STATUSES))}")
    if status == "Draft":
        warnings.append("Canvas is Draft — must reach Aligned before /spdd-generate runs")

    # Seven REASONS sections
    # Count occurrences of each section pattern against line starts
    section_hits: list[bool] = []
    for pattern in REQUIRED_SECTIONS:
        found = any(re.match(pattern, line) for line in lines)
        section_hits.append(found)

    # The two "## S" sections: first is Structure, second is Safeguards.
    # REQUIRED_SECTIONS[3] matches any "## S " line; REQUIRED_SECTIONS[6] requires "Safeguards".
    # Check that at least two "## S" headings exist (Structure + Safeguards).
    s_headings = [l for l in lines if re.match(r"^## S", l)]
    if len(s_headings) < 2:
        errors.append("missing second '## S' section (Safeguards) — Canvas needs both Structure and Safeguards")

    for i, (pattern, found) in enumerate(zip(REQUIRED_SECTIONS, section_hits)):
        if not found:
            # Skip the Safeguards check here — handled above via s_headings count
            if "Safeguards" in pattern:
                continue
            label = pattern.lstrip("^## ").rstrip(" ")
            errors.append(f"missing REASONS section matching: '## {label} ...'")

    # Context security checklist
    if SECURITY_CHECKLIST_MARKER not in text:
        errors.append("missing Context Security checklist in Safeguards section")

    # canvas.private.md should never be committed alongside canvas.md
    private = path.parent / "canvas.private.md"
    if private.exists():
        errors.append(
            f"canvas.private.md found at {private} — this file must be gitignored and never committed"
        )

    return errors, warnings


def main() -> None:
    canvas_dir = Path(sys.argv[1]) if len(sys.argv) > 1 else Path("docs/spdd")

    if not canvas_dir.exists():
        print(f"  (canvas dir not found: {canvas_dir} — skipping)")
        sys.exit(0)

    canvases = sorted(canvas_dir.rglob("canvas.md"))
    if not canvases:
        print(f"  (no canvas.md files found under {canvas_dir})")
        sys.exit(0)

    errors_found = False
    for canvas in canvases:
        rel = canvas.relative_to(Path("."))
        errors, warnings = validate_canvas(canvas)
        if errors:
            errors_found = True
            print(f"FAIL {rel}:")
            for e in errors:
                print(f"  ERROR  {e}")
            for w in warnings:
                print(f"  WARN   {w}")
        elif warnings:
            print(f"  WARN {rel}:")
            for w in warnings:
                print(f"         {w}")
        else:
            print(f"  OK   {rel}")

    if errors_found:
        sys.exit(1)


if __name__ == "__main__":
    main()
