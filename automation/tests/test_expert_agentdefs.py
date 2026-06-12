# SPDX-License-Identifier: Apache-2.0
# automation/tests/test_expert_agentdefs.py
#
# O2 of EPIC #881 (#1097): asserts each expert AgentDef manifest under
# automation/workflows/experts/ is a faithful 1:1 translation of its
# near-term source config (docs/archive/dev-advisory/experts/<name>.yaml,
# archived by #1129; per ADR-028 the archived YAMLs remain the source of
# truth for context_slice, I/O contract, and aggregation_weight).
#
# The planner manifest (planner.yaml) is the planning-task-split expert
# extended with the identify_next_issue capability (ADR-028); its source of
# truth is planning-task-split.yaml.

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
SCHEMA_PATH = REPO_ROOT / "spec/schemas/agent-def.schema.json"
AGENTDEF_DIR = REPO_ROOT / "automation/workflows/experts"
SOURCE_DIR = REPO_ROOT / "docs/archive/dev-advisory/experts"

DOMAIN_EXPERTS = [
    "arch-adr",
    "persistence-state",
    "api-contract",
    "security-supply-chain",
    "qa-bdd",
    "docs-agents",
    "ci-release",
    "planning-task-split",
]

# manifest name -> source-of-truth name (planner extends planning-task-split)
ALL_MANIFESTS = {name: name for name in DOMAIN_EXPERTS}
ALL_MANIFESTS["planner"] = "planning-task-split"

CORE_OUTPUT_KEYS = {
    "summary",
    "recommended_actions",
    "reasons_decisions",
    "confidence",
    "flags",
}


def load_manifest(name):
    with open(AGENTDEF_DIR / f"{name}.yaml") as f:
        return yaml.safe_load(f)


def load_source(name):
    with open(SOURCE_DIR / f"{name}.yaml") as f:
        return yaml.safe_load(f)


def get_capability(manifest, capability_name):
    caps = {c["name"]: c for c in manifest["spec"]["capabilities"]}
    assert capability_name in caps, f"capability {capability_name!r} missing"
    return caps[capability_name]


def test_exactly_nine_agentdef_manifests():
    """8 domain experts + planner = 9 manifests (EPIC #881 O2)."""
    found = sorted(p.stem for p in AGENTDEF_DIR.glob("*.yaml"))
    assert found == sorted(ALL_MANIFESTS), f"unexpected manifest set: {found}"


@pytest.mark.skipif(not HAS_JSONSCHEMA, reason="jsonschema package required")
@pytest.mark.parametrize("name", sorted(ALL_MANIFESTS))
def test_agentdef_validates_against_schema(name):
    """Every manifest validates against agent-def.schema.json."""
    with open(SCHEMA_PATH) as f:
        schema = json.load(f)
    jsonschema.validate(load_manifest(name), schema)


@pytest.mark.parametrize("name", sorted(ALL_MANIFESTS))
def test_review_input_contract_matches_source(name):
    """review input_schema mirrors the source YAML input_contract."""
    source = load_source(ALL_MANIFESTS[name])
    review = get_capability(load_manifest(name), "review")
    input_schema = review["input_schema"]

    required = set(source["input_contract"]["required"])
    optional = set(source["input_contract"].get("optional", []))

    assert set(input_schema["required"]) == required
    assert set(input_schema["properties"]) == required | optional


@pytest.mark.parametrize("name", sorted(ALL_MANIFESTS))
def test_review_output_contract_matches_source(name):
    """review output_schema mirrors the source YAML output_contract."""
    source = load_source(ALL_MANIFESTS[name])
    review = get_capability(load_manifest(name), "review")
    output_schema = review["output_schema"]

    assert set(output_schema["required"]) == CORE_OUTPUT_KEYS
    assert set(output_schema["properties"]) == CORE_OUTPUT_KEYS | {"extra_fields"}

    source_extra = source["output_contract"]["extra_fields"]
    manifest_extra = output_schema["properties"]["extra_fields"]["properties"]
    assert set(manifest_extra) == set(source_extra)

    # Spot-check core types stayed faithful.
    assert output_schema["properties"]["summary"]["type"] == "string"
    assert output_schema["properties"]["confidence"]["enum"] == [
        "low",
        "medium",
        "high",
    ]
    assert output_schema["properties"]["flags"]["type"] == "array"


@pytest.mark.parametrize("name", sorted(ALL_MANIFESTS))
def test_context_slice_matches_source(name):
    """context_slice files + max_tokens are verbatim from the source YAML."""
    source = load_source(ALL_MANIFESTS[name])
    review = get_capability(load_manifest(name), "review")
    slice_schema = review["input_schema"]["properties"]["context_slice"]

    assert set(slice_schema["required"]) == {"files", "max_tokens"}
    assert (
        slice_schema["properties"]["files"]["default"]
        == source["context_slice"]["files"]
    )
    assert (
        slice_schema["properties"]["max_tokens"]["default"]
        == source["context_slice"]["max_tokens"]
    )


@pytest.mark.parametrize("name", sorted(ALL_MANIFESTS))
def test_weight_and_max_tokens_labels_match_source(name):
    """aggregation_weight / max_tokens labels match the source YAML."""
    source = load_source(ALL_MANIFESTS[name])
    labels = load_manifest(name)["metadata"]["labels"]

    assert float(labels["aggregation-weight"]) == source["aggregation_weight"]
    assert int(labels["context-max-tokens"]) == source["context_slice"]["max_tokens"]


def test_planner_identify_next_issue_contract():
    """Planner exposes identify_next_issue with the canvas-declared I/O."""
    cap = get_capability(load_manifest("planner"), "identify_next_issue")

    assert set(cap["input_schema"]["required"]) == {
        "milestone",
        "open_issues",
        "in_progress",
        "dependency_table",
    }
    assert set(cap["output_schema"]["required"]) == {
        "next_issue",
        "blocked_by",
        "ready_batch",
        "rationale",
    }


def test_domain_experts_expose_only_review():
    """Only the planner carries a second capability."""
    for name in DOMAIN_EXPERTS:
        caps = [c["name"] for c in load_manifest(name)["spec"]["capabilities"]]
        assert caps == ["review"], f"{name}: unexpected capabilities {caps}"
    planner_caps = [c["name"] for c in load_manifest("planner")["spec"]["capabilities"]]
    assert planner_caps == ["review", "identify_next_issue"]
