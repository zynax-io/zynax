# SPDX-License-Identifier: Apache-2.0
# automation/tests/test_learning_synthesizer.py
#
# O7 of EPIC #881 (#1102): asserts the learning-synthesizer AgentDef
# (automation/workflows/learning-synthesizer.yaml) — the on-platform
# /m6-learn — declares the synthesize_learnings contract per the canvas
# and that the apply is human-gated: the only expressible proposal status
# is pending-human-review and the agent exposes no apply/write capability.
# Binds the scenarios in automation/tests/features/learning_synthesizer.feature.
#
# The synthesis rules (recurrence >= 2, dedup against applied) are declared
# in the manifest contract, so these tests evaluate the manifest itself
# against sample session results via a contract-driven reference
# implementation: no running platform is required (the live e2e is O8, #1103).

import json
from pathlib import Path

import pytest
import yaml

try:
    import jsonschema

    HAS_JSONSCHEMA = True
except ImportError:
    HAS_JSONSCHEMA = False

REPO_ROOT = Path(__file__).resolve().parents[2]
MANIFEST_PATH = REPO_ROOT / "automation/workflows/learning-synthesizer.yaml"
SCHEMA_PATH = REPO_ROOT / "spec/schemas/agent-def.schema.json"
EXPERTS_DIR = REPO_ROOT / "automation/workflows/experts"

# prohibited_auto_actions — verbatim from Wave 2 (canvas #881 §S / ADR-028).
NEVER_AUTO = [
    "merge",
    "push",
    "bump-dependency",
    "close-issue",
    "delete-branch",
    "force-push",
]

# Capability names that would mean the synthesizer can change a manifest
# itself — the human-gated contract forbids all of them.
FORBIDDEN_NAME_FRAGMENTS = ["apply", "edit", "write", "commit", "push", "merge"]


def load_manifest():
    with open(MANIFEST_PATH) as f:
        return yaml.safe_load(f)


def get_capability(manifest, capability_name):
    caps = {c["name"]: c for c in manifest["spec"]["capabilities"]}
    assert capability_name in caps, f"capability {capability_name!r} missing"
    return caps[capability_name]


# ── sample session results (the AC 1 fixture) ───────────────────────────────
# Two patterns recur in >= 2 separate sessions (must be proposed), one is a
# single-session pattern (recurrence rule excludes it), and one recurring
# pattern is already in the apply log (dedup excludes it).

SAMPLE_SESSION_RESULTS = [
    {
        "domain": "ci-release",
        "session": "#1138",
        "patterns": ["Retag promotion must be restartable after partial failure"],
    },
    {
        "domain": "ci-release",
        "session": "#1139",
        "patterns": ["Retag promotion must be restartable after partial failure"],
    },
    {
        "domain": "go-services",
        "session": "#1100",
        "patterns": ["Bind the context slice before persisting the task row"],
    },
    {
        "domain": "go-services",
        "session": "#1095",
        "patterns": ["Bind the context slice before persisting the task row"],
    },
    {
        "domain": "ci-release",
        "session": "#1132",
        "patterns": ["Staging cleanup races the retag promotion"],
    },
    {
        "domain": "go-services",
        "session": "#1099",
        "patterns": ["Wrap ctx.Err() before return"],
    },
    {
        "domain": "go-services",
        "session": "#1101",
        "patterns": ["Wrap ctx.Err() before return"],
    },
]

APPLIED_PATTERNS = ["Wrap ctx.Err() before return"]

EXPECTED_PROPOSED = {
    "Retag promotion must be restartable after partial failure",
    "Bind the context slice before persisting the task row",
}

# Learning domain (docs/ai-learnings file stem) → O2 expert AgentDef the
# proposal feeds back into (canvas #881: "feeds proposals back into the
# expert AgentDefs from O2").
DOMAIN_TARGETS = {
    "ci-release": "ci-release.yaml",
    "go-services": "persistence-state.yaml",
}


def synthesize(session_results, applied_patterns):
    """Contract-driven reference implementation of synthesize_learnings.

    Every rule it enforces is read from the manifest itself: the recurrence
    floor comes from the declared `minimum` on the recurrence field, the
    proposal status from the declared (single-value) status enum, and dedup
    from the required applied_patterns input. Mirrors /milestone-learn:
    cluster, filter by recurrence, dedup against applied, propose only —
    never touch a manifest.
    """
    cap = get_capability(load_manifest(), "synthesize_learnings")
    item_props = cap["output_schema"]["properties"]["proposed_manifest_updates"][
        "items"
    ]["properties"]
    min_recurrence = item_props["recurrence"]["minimum"]
    (status,) = item_props["status"]["enum"]

    clusters = {}
    for block in session_results:
        for pattern in block["patterns"]:
            clusters.setdefault((block["domain"], pattern), set()).add(block["session"])

    proposals = []
    for (domain, pattern), sessions in sorted(clusters.items()):
        if len(sessions) < min_recurrence:
            continue
        if pattern in applied_patterns:
            continue
        proposals.append(
            {
                "target_manifest": (
                    f"automation/workflows/experts/{DOMAIN_TARGETS[domain]}"
                ),
                "proposed_addition": f"- **{pattern}**",
                "category": "domain",
                "recurrence": len(sessions),
                "source_sessions": sorted(sessions),
                "status": status,
            }
        )

    rows = "\n".join(
        f"| {i} | {p['target_manifest']} | {p['proposed_addition']} "
        f"| {p['category']} | {', '.join(p['source_sessions'])} | pending | — |"
        for i, p in enumerate(proposals, start=1)
    )
    return {
        "proposed_manifest_updates": proposals,
        "apply_log_entry": (
            "| # | Domain | Pattern | Category | Source sessions | Status | Delta |\n"
            "|---|--------|---------|----------|-----------------|--------|-------|\n"
            + rows
        ),
        "summary": (
            f"{len(proposals)} proposed | 0 applied | 0 rejected "
            f"| {len(proposals)} pending human review"
        ),
    }


# ── Scenario: Manifest validates against the AgentDef schema ────────────────


@pytest.mark.skipif(not HAS_JSONSCHEMA, reason="jsonschema package required")
def test_manifest_validates_against_schema():
    """learning-synthesizer.yaml validates against agent-def.schema.json
    (AC 3)."""
    with open(SCHEMA_PATH) as f:
        schema = json.load(f)
    jsonschema.validate(load_manifest(), schema)


# ── Scenario: exactly the synthesize_learnings capability ───────────────────


def test_exposes_only_synthesize_learnings():
    """One capability, synthesize_learnings — and no capability name even
    suggests applying, editing, or committing a change."""
    caps = [c["name"] for c in load_manifest()["spec"]["capabilities"]]
    assert caps == ["synthesize_learnings"], f"unexpected capabilities: {caps}"
    for name in caps:
        for fragment in FORBIDDEN_NAME_FRAGMENTS:
            assert fragment not in name, f"{name!r} suggests a write: {fragment!r}"


# ── Scenario: the capability I/O contract matches the canvas ────────────────


def test_io_contract_matches_canvas():
    """session_results[] (+ applied_patterns for dedup, context_slice per
    ADR-028) → proposed_manifest_updates[] (+ apply-log entry + summary)."""
    cap = get_capability(load_manifest(), "synthesize_learnings")
    assert set(cap["input_schema"]["required"]) == {
        "session_results",
        "applied_patterns",
        "context_slice",
    }
    assert set(cap["output_schema"]["required"]) == {
        "proposed_manifest_updates",
        "apply_log_entry",
        "summary",
    }
    assert (
        cap["output_schema"]["properties"]["proposed_manifest_updates"]["type"]
        == "array"
    )


# ── Scenario: context slice per canvas Appendix A ────────────────────────────


def test_context_slice_matches_canvas():
    """Learning record + current expert manifests, capped at 4000 tokens —
    canvas #881 Appendix A; budget label matches the declared default."""
    manifest = load_manifest()
    cap = get_capability(manifest, "synthesize_learnings")
    slice_schema = cap["input_schema"]["properties"]["context_slice"]

    assert set(slice_schema["required"]) == {"files", "max_tokens"}
    assert slice_schema["properties"]["files"]["default"] == [
        "docs/ai-learnings/*.md",
        "automation/workflows/experts/*.yaml",
    ]
    max_tokens = slice_schema["properties"]["max_tokens"]["default"]
    assert max_tokens == 4000
    assert int(manifest["metadata"]["labels"]["context-max-tokens"]) == max_tokens


# ── Scenario: the recurrence rule is declared in the contract ────────────────


def test_recurrence_rule_declared():
    """A proposal can never claim recurrence below 2 — the /milestone-learn
    recurrence rule is part of the schema, not just prose."""
    cap = get_capability(load_manifest(), "synthesize_learnings")
    recurrence = cap["output_schema"]["properties"]["proposed_manifest_updates"][
        "items"
    ]["properties"]["recurrence"]
    assert recurrence["minimum"] == 2


# ── Scenario: sample session results → proposed_manifest_updates[] ──────────


def test_sample_session_results_emit_proposals():
    """AC 1: given sample session results, emits proposed_manifest_updates[]
    — recurring not-yet-applied patterns in; single-session and
    already-applied patterns out."""
    result = synthesize(SAMPLE_SESSION_RESULTS, APPLIED_PATTERNS)
    proposals = result["proposed_manifest_updates"]

    proposed = {p["proposed_addition"].strip("- *") for p in proposals}
    assert proposed == EXPECTED_PROPOSED
    assert all(p["recurrence"] >= 2 for p in proposals)
    assert all(len(p["source_sessions"]) == p["recurrence"] for p in proposals)
    # the single-session pattern is filtered by the recurrence rule
    assert not any("Staging cleanup" in p["proposed_addition"] for p in proposals)
    # the already-applied pattern is deduplicated, never re-proposed
    assert not any("ctx.Err()" in p["proposed_addition"] for p in proposals)
    # every proposal row lands in the apply-log entry as pending
    assert result["apply_log_entry"].count("| pending |") == len(proposals)


@pytest.mark.skipif(not HAS_JSONSCHEMA, reason="jsonschema package required")
def test_sample_io_round_trip_validates_against_contract():
    """The sample input and the synthesized result validate against the
    capability's declared input_schema / output_schema."""
    cap = get_capability(load_manifest(), "synthesize_learnings")
    slice_props = cap["input_schema"]["properties"]["context_slice"]["properties"]
    payload = {
        "session_results": SAMPLE_SESSION_RESULTS,
        "applied_patterns": APPLIED_PATTERNS,
        "context_slice": {
            "files": slice_props["files"]["default"],
            "max_tokens": slice_props["max_tokens"]["default"],
        },
    }
    jsonschema.validate(payload, cap["input_schema"])

    result = synthesize(SAMPLE_SESSION_RESULTS, APPLIED_PATTERNS)
    jsonschema.validate(result, cap["output_schema"])


# ── Scenario: proposals target existing O2 expert AgentDefs ──────────────────


def test_proposals_target_existing_expert_agentdefs():
    """Every target_manifest is a real expert AgentDef from O2 — the
    feedback loop points at manifests that exist."""
    result = synthesize(SAMPLE_SESSION_RESULTS, APPLIED_PATTERNS)
    for proposal in result["proposed_manifest_updates"]:
        target = REPO_ROOT / proposal["target_manifest"]
        assert target.parent == EXPERTS_DIR, proposal["target_manifest"]
        assert target.is_file(), f"target does not exist: {proposal['target_manifest']}"


# ── Scenario: apply is human-gated — no manifest is auto-edited ──────────────


def test_apply_is_human_gated():
    """AC 2: no manifest is auto-edited. The only expressible proposal
    status is pending-human-review, no capability matches a prohibited auto
    action, and synthesis leaves every expert manifest on disk untouched."""
    cap = get_capability(load_manifest(), "synthesize_learnings")
    status_enum = cap["output_schema"]["properties"]["proposed_manifest_updates"][
        "items"
    ]["properties"]["status"]["enum"]
    assert status_enum == ["pending-human-review"]

    caps = {c["name"] for c in load_manifest()["spec"]["capabilities"]}
    assert not caps & {a.replace("-", "_") for a in NEVER_AUTO}

    before = {p: p.read_bytes() for p in sorted(EXPERTS_DIR.glob("*.yaml"))}
    result = synthesize(SAMPLE_SESSION_RESULTS, APPLIED_PATTERNS)
    after = {p: p.read_bytes() for p in sorted(EXPERTS_DIR.glob("*.yaml"))}
    assert before == after, "synthesis must never edit an expert manifest"

    assert all(
        p["status"] == "pending-human-review"
        for p in result["proposed_manifest_updates"]
    )


def test_manifest_is_labelled_human_gated():
    """The human-gating contract is discoverable via a registry label."""
    labels = load_manifest()["metadata"]["labels"]
    assert labels["human-gated"] == "true"
