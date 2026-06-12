# SPDX-License-Identifier: Apache-2.0
# automation/tests/test_orchestrator_workflow.py
#
# O3 of EPIC #881 (#1098): asserts the dev-advisory-orchestrator Workflow
# manifest is a faithful translation of the archived orchestrator config
# (docs/archive/dev-advisory/orchestrator/config.yaml — source of truth per
# ADR-028) and that its capability references match the 9 expert AgentDef
# manifests under automation/workflows/experts/ 1:1.

from pathlib import Path

import pytest
import yaml

REPO_ROOT = Path(__file__).resolve().parents[2]
WORKFLOW_PATH = REPO_ROOT / "automation/workflows/dev-advisory-orchestrator.yaml"
EXPERT_DIR = REPO_ROOT / "automation/workflows/experts"
SOURCE_CONFIG = REPO_ROOT / "docs/archive/dev-advisory/orchestrator/config.yaml"
FEATURE_PATH = REPO_ROOT / "automation/tests/features/dev_advisory_orchestrator.feature"

EXPECTED_STATES = {"fan_out", "aggregate", "act", "escalate", "done"}


@pytest.fixture(scope="module")
def workflow():
    with open(WORKFLOW_PATH) as f:
        return yaml.safe_load(f)


@pytest.fixture(scope="module")
def source_config():
    with open(SOURCE_CONFIG) as f:
        return yaml.safe_load(f)


@pytest.fixture(scope="module")
def expert_names():
    return sorted(p.stem for p in EXPERT_DIR.glob("*.yaml"))


def state(workflow, name):
    states = workflow["spec"]["states"]
    assert name in states, f"state {name!r} missing"
    return states[name]


def test_kind_and_api_version(workflow):
    """Orchestration is kind: Workflow, apiVersion zynax.io/v1 (ADR-028)."""
    assert workflow["kind"] == "Workflow"
    assert workflow["apiVersion"] == "zynax.io/v1"
    assert workflow["metadata"]["name"] == "dev-advisory-orchestrator"
    assert workflow["metadata"]["namespace"] == "automation"


def test_state_machine_shape(workflow):
    """fan_out → aggregate → act/escalate → done, starting at fan_out."""
    assert workflow["spec"]["initial_state"] == "fan_out"
    assert set(workflow["spec"]["states"]) == EXPECTED_STATES
    assert state(workflow, "escalate")["type"] == "human_in_the_loop"
    assert state(workflow, "done")["type"] == "terminal"
    assert "on" not in state(workflow, "done")


def test_fan_out_dispatches_all_nine_experts_exactly_once(workflow, expert_names):
    """fan_out invokes the review capability once per expert manifest (1:1)."""
    actions = state(workflow, "fan_out")["actions"]
    assert len(actions) == 9
    assert all(a["capability"] == "review" for a in actions)
    dispatched = sorted(a["input"]["expert"] for a in actions)
    assert dispatched == expert_names


def test_fan_out_inputs_satisfy_review_contract(workflow):
    """Every dispatch carries the review capability's required inputs.

    context_slice is intentionally absent: it is bound at dispatch time by
    the context-slice injection binding (O5, #1100) so the expert AgentDefs
    stay the single source of truth for slices (ADR-028).
    """
    for action in state(workflow, "fan_out")["actions"]:
        action_input = action["input"]
        for field in ("trigger", "diff_summary", "changed_files"):
            assert field in action_input, f"{action_input['expert']}: {field}"
        assert action_input["trigger"] == "pull_request"
        assert "context_slice" not in action_input


def test_fan_out_timeout_matches_source_config(workflow, source_config):
    """Per-expert timeout mirrors fan_out.timeout_seconds (300s)."""
    expected = f"{source_config['fan_out']['timeout_seconds'] // 60}m"
    for action in state(workflow, "fan_out")["actions"]:
        assert action["timeout"] == expected
    # On per-expert timeout the orchestrator continues with partial outputs.
    timeout_edges = [
        t for t in state(workflow, "fan_out")["on"] if t["event"] == "review.timeout"
    ]
    assert len(timeout_edges) == 1
    assert timeout_edges[0]["goto"] == "aggregate"


def test_fan_out_collects_each_expert_output_separately(workflow):
    """Each dispatch maps its result to a distinct context key (isolation)."""
    keys = []
    for action in state(workflow, "fan_out")["actions"]:
        assert list(action["output"].values()) == ["{{ .result }}"]
        keys.extend(action["output"].keys())
    assert len(set(keys)) == 9
    assert all(k.startswith("context.reviews.") for k in keys)


def test_aggregation_matches_source_config(workflow, source_config):
    """Strategy, minimum weight, and thresholds are verbatim from config."""
    src = source_config["aggregation"]
    actions = state(workflow, "aggregate")["actions"]
    assert len(actions) == 1
    agg_input = actions[0]["input"]

    assert actions[0]["capability"] == "aggregate_reviews"
    assert agg_input["strategy"] == src["strategy"]
    assert agg_input["aggregate_weight_minimum"] == src["aggregate_weight_minimum"]
    assert agg_input["conflict_resolution"] == src["conflict_resolution"]
    assert agg_input["escalation_threshold"] == src["escalation_threshold"]
    # Weights come from expert registry labels — never duplicated here.
    assert agg_input["weight_label"] == "aggregation-weight"
    assert "weights" not in agg_input


def test_aggregate_routes_to_act_or_escalate(workflow):
    """Escalation guard routes to escalate; the clean path routes to act."""
    transitions = state(workflow, "aggregate")["on"]
    targets = {t["goto"] for t in transitions}
    assert targets == {"act", "escalate"}
    escalate_edge = next(t for t in transitions if t["goto"] == "escalate")
    assert "escalation_required" in escalate_edge["guard"]


def test_act_honours_human_in_the_loop_policy(workflow, source_config):
    """auto_allowed / never_auto are verbatim from the archived config."""
    src = source_config["human_in_the_loop"]
    actions = state(workflow, "act")["actions"]
    assert len(actions) == 1
    act_input = actions[0]["input"]

    assert act_input["auto_allowed"] == src["auto_allowed"]
    assert act_input["never_auto"] == src["never_auto"]
    assert not set(act_input["auto_allowed"]) & set(act_input["never_auto"])


def test_no_state_invokes_a_prohibited_auto_action(workflow, source_config):
    """No workflow action capability is ever a prohibited_auto_action."""
    never_auto = set(source_config["human_in_the_loop"]["never_auto"])
    for state_def in workflow["spec"]["states"].values():
        for action in state_def.get("actions", []):
            assert action["capability"] not in never_auto


def test_escalate_resumes_only_via_human_signal(workflow):
    """The escalate state transitions only on human.* events."""
    transitions = state(workflow, "escalate")["on"]
    assert all(t["event"].startswith("human.") for t in transitions)
    assert {t["goto"] for t in transitions} == {"act", "done"}


def test_done_records_decision_log(workflow):
    """Every run terminates by recording a decision-log entry."""
    actions = state(workflow, "done")["actions"]
    assert [a["capability"] for a in actions] == ["record_decision"]


def test_bdd_feature_committed():
    """ADR-016: the BDD contract ships with the manifest."""
    text = FEATURE_PATH.read_text()
    assert "Feature: Dev-advisory orchestrator" in text
    for keyword in ("fan_out", "aggregate", "prohibited_auto_action", "escalate"):
        assert keyword in text, f"feature file missing {keyword!r}"
