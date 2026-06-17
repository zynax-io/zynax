# SPDX-License-Identifier: Apache-2.0
"""Drift guard for the authoring <-> runtime expert mapping (ADR-033).

EPIC X (#1170), step X.5 (#1205). Reconciles the machine-readable mapping
(automation/experts/runtime_mapping.yaml) against three surfaces and fails the
build on any divergence:

1. Declared mapping is mandatory. Every authoring expert
   (.claude/commands/experts/<slug>.md) MUST appear exactly once in the mapping
   file, and its ``runtime_mapping`` MUST be either a runtime AgentDef name or
   the literal ``authoring-only``. A missing or empty value is a hard failure.
2. Runtime reference must resolve. When ``runtime_mapping`` names an AgentDef,
   that AgentDef MUST exist as a registerable expert under ``agents/examples/``.
   A dangling reference fails.
3. Table reconciliation. The mapping file MUST stay identical to ADR-033's
   mapping table (the human-readable mirror). Any unlisted expert, stale row, or
   changed counterpart fails.

The .claude/** tree is read-only here (it is CODEOWNERS-gated): the script never
writes to it. Run as ``python automation/scripts/check_expert_mapping.py`` from
the repo root, or via ``make check-expert-mapping``.
"""

from __future__ import annotations

import re
import sys
from pathlib import Path

import yaml

REPO_ROOT = Path(__file__).resolve().parents[2]
MAPPING_PATH = REPO_ROOT / "automation/experts/runtime_mapping.yaml"
AUTHORING_DIR = REPO_ROOT / ".claude/commands/experts"
EXAMPLES_DIR = REPO_ROOT / "agents/examples"
ADR_PATH = REPO_ROOT / "docs/adr/ADR-033-expert-agent-substrate.md"

AUTHORING_ONLY = "authoring-only"


def load_mapping() -> list[dict[str, str]]:
    """Return the ``experts`` list from the mapping manifest."""
    doc = yaml.safe_load(MAPPING_PATH.read_text())
    experts = doc.get("experts")
    if not isinstance(experts, list):
        raise SystemExit(f"{MAPPING_PATH}: missing or invalid 'experts' list")
    return experts


def discover_authoring_experts() -> set[str]:
    """Slugs of every authoring expert under .claude/commands/experts/ (read-only)."""
    return {p.stem for p in AUTHORING_DIR.glob("*.md")}


def discover_runtime_agents() -> set[str]:
    """Names of registerable runtime agents under agents/examples/."""
    return {
        p.name
        for p in EXAMPLES_DIR.iterdir()
        if p.is_dir() and (p / "pyproject.toml").exists()
    }


def parse_adr_table() -> dict[str, str]:
    """Parse ADR-033's mapping table into {authoring_slug: runtime_mapping}."""
    rows: dict[str, str] = {}
    row_re = re.compile(r"^\|\s*`([^`]+)`\s*\|\s*`([^`]+)`\s*\|")
    for line in ADR_PATH.read_text().splitlines():
        m = row_re.match(line)
        if m:
            rows[m.group(1)] = m.group(2)
    return rows


def check() -> list[str]:
    """Run all three drift-guard rules; return a list of error strings."""
    errors: list[str] = []
    experts = load_mapping()

    # Rule 1 — declared mapping is mandatory and well-formed.
    declared: dict[str, str] = {}
    for i, entry in enumerate(experts):
        slug = entry.get("authoring")
        if not slug:
            errors.append(f"mapping entry #{i}: missing 'authoring' slug")
            continue
        if slug in declared:
            errors.append(f"{slug}: declared more than once in the mapping file")
        rm = entry.get("runtime_mapping")
        if not rm:
            errors.append(f"{slug}: empty or missing 'runtime_mapping' (ADR-033)")
        declared[slug] = rm or ""

    authoring = discover_authoring_experts()
    missing = sorted(authoring - declared.keys())
    if missing:
        errors.append(
            "authoring experts with no runtime_mapping declaration "
            f"(ADR-033 rule 1): {missing}"
        )
    extra = sorted(declared.keys() - authoring)
    if extra:
        errors.append(
            f"mapping lists experts with no .claude/commands/experts file: {extra}"
        )

    # Rule 2 — runtime references must resolve to agents/examples/.
    runtime_agents = discover_runtime_agents()
    for slug, rm in declared.items():
        if rm and rm != AUTHORING_ONLY and rm not in runtime_agents:
            errors.append(
                f"{slug}: runtime_mapping '{rm}' does not resolve to "
                f"agents/examples/{rm} (ADR-033 rule 2)"
            )

    # Rule 3 — table reconciliation against ADR-033 (the human-readable mirror).
    adr_table = parse_adr_table()
    for slug, rm in declared.items():
        if slug not in adr_table:
            errors.append(f"{slug}: present in mapping but absent from ADR-033 table")
        elif adr_table[slug] != rm:
            errors.append(
                f"{slug}: ADR-033 table says '{adr_table[slug]}' but mapping says "
                f"'{rm}' (ADR-033 rule 3)"
            )
    for slug in sorted(adr_table.keys() - declared.keys()):
        errors.append(f"{slug}: listed in ADR-033 table but absent from mapping")

    return errors


def main() -> int:
    errors = check()
    if errors:
        print("Expert mapping drift guard FAILED (ADR-033):", file=sys.stderr)
        for err in errors:
            print(f"  - {err}", file=sys.stderr)
        return 1
    count = len(load_mapping())
    print(f"Expert mapping drift guard OK — {count} authoring experts reconciled.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
