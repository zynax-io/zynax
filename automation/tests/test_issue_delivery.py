# SPDX-License-Identifier: Apache-2.0
# automation/tests/test_issue_delivery.py
#
# O4 of EPIC #881 (#1099): asserts the intake → plan → route leg of the
# issue-delivery Workflow (automation/workflows/issue-delivery.yaml).
# Binds the scenarios in automation/tests/features/issue_delivery.feature.
#
# The routing table lives declaratively in the manifest (route state,
# first-match-wins guarded transitions — mirroring /m6-orchestrate STEP 5 and
# the engine's resolveTransition semantics), so these tests evaluate the
# manifest itself against fixture issues: no running platform is required.

import json
import re
from pathlib import Path

import pytest
import yaml

try:
    import jsonschema

    HAS_JSONSCHEMA = True
except ImportError:
    HAS_JSONSCHEMA = False

REPO_ROOT = Path(__file__).resolve().parents[2]
WORKFLOW_PATH = REPO_ROOT / "automation/workflows/issue-delivery.yaml"
SCHEMA_PATH = REPO_ROOT / "spec/schemas/workflow.schema.json"
PLANNER_PATH = REPO_ROOT / "automation/workflows/experts/planner.yaml"
EXPERTS_DIR = REPO_ROOT / "automation/workflows/experts"

O4_STATES = {"intake", "plan", "route", "routed", "blocked", "failed"}
TERMINAL_STATES = {"routed", "blocked", "failed"}


def load_workflow():
    """Load the manifest, normalising PyYAML's YAML-1.1 quirk: the bare key
    `on` parses as boolean True under safe_load, while the authoritative
    validator (zynax-ci, go-yaml v3 / YAML 1.2) keeps it the string "on"."""
    with open(WORKFLOW_PATH) as f:
        return _normalize_on(yaml.safe_load(f))


def _normalize_on(obj):
    if isinstance(obj, dict):
        return {
            ("on" if key is True else key): _normalize_on(value)
            for key, value in obj.items()
        }
    if isinstance(obj, list):
        return [_normalize_on(item) for item in obj]
    return obj


def load_planner():
    with open(PLANNER_PATH) as f:
        return yaml.safe_load(f)


# ── minimal guard evaluator ──────────────────────────────────────────────────
# Supports exactly the CEL subset used by the manifest guards:
#   ctx.key == 'value' · ctx.key != 'value' · ctx.key in ['a', 'b'] · A || B


def eval_guard(expr, ctx):
    parts = [p.strip() for p in expr.split("||")]
    return any(_eval_clause(p, ctx) for p in parts)


def _eval_clause(clause, ctx):
    m = re.fullmatch(r"ctx\.(\w+)\s*(==|!=)\s*'([^']*)'", clause)
    if m:
        key, op, value = m.groups()
        actual = ctx.get(key, "")
        return (actual == value) if op == "==" else (actual != value)
    m = re.fullmatch(r"ctx\.(\w+)\s+in\s+\[([^\]]*)\]", clause)
    if m:
        key, items = m.groups()
        values = [v.strip().strip("'\"") for v in items.split(",")]
        return ctx.get(key, "") in values
    raise AssertionError(f"guard clause not in the supported CEL subset: {clause!r}")


# ── minimal state-machine walker (engine resolveTransition semantics) ───────


def classify_title(title):
    """Mirror of the read_issue contract: extract conventional-commit
    type + scope from the issue title (mechanical parse — the routing
    decision itself stays in the manifest)."""
    m = re.match(r"^(\w+)(?:\(([^)]*)\))?:", title)
    assert m, f"fixture title is not conventional-commit shaped: {title!r}"
    return m.group(1), m.group(2) or ""


def first_matching_transition(state, event, ctx):
    """First transition whose event matches and whose guard passes —
    engine-adapter resolveTransition semantics. An empty event (action-less
    state sentinel) matches any declared transition event."""
    for t in state.get("on", []):
        if event and t["event"] != event:
            continue
        guard = t.get("guard")
        if guard and not eval_guard(guard, ctx):
            continue
        return t
    raise AssertionError(f"no transition matched event {event!r} with ctx {ctx}")


def run_o4_leg(issue_title, planner_result):
    """Walk intake → plan → route against a fixture issue and a stubbed
    planner reply. Returns (final_state, ctx)."""
    spec = load_workflow()["spec"]
    states = spec["states"]
    ctx = {}

    # intake: read_issue output mappings populate the context.
    commit_type, scope = classify_title(issue_title)
    ctx["issue_title"] = issue_title
    ctx["commit_type"] = commit_type
    ctx["issue_scope"] = scope
    current = spec["initial_state"]
    t = first_matching_transition(states[current], "read_issue.completed", ctx)
    current = t["goto"]

    # plan: identify_next_issue output mappings populate the context.
    assert current == "plan"
    ctx["next_issue"] = str(planner_result["next_issue"])
    ctx["blocked_by"] = ", ".join(str(n) for n in planner_result["blocked_by"])
    ctx["ready_batch"] = ", ".join(str(n) for n in planner_result["ready_batch"])
    ctx["plan_rationale"] = planner_result["rationale"]
    t = first_matching_transition(states[current], "identify_next_issue.completed", ctx)
    _apply_set(t, ctx)
    current = t["goto"]

    # route (when reached): action-less state — sentinel empty event.
    if current == "route":
        t = first_matching_transition(states[current], "", ctx)
        _apply_set(t, ctx)
        current = t["goto"]

    assert states[current].get("type") == "terminal"
    return current, ctx


def _apply_set(transition, ctx):
    for key, value in transition.get("set", {}).items():
        assert key.startswith("context."), f"set key outside context: {key}"
        if "{{" not in str(value):
            ctx[key.removeprefix("context.")] = str(value)


PLANNER_OK = {
    "next_issue": 1099,
    "blocked_by": [],
    "ready_batch": [1099],
    "rationale": "O4 is the only dependency-free Wave 4 story",
}


# ── Scenario: Manifest declares the O4 state machine ─────────────────────────


@pytest.mark.skipif(not HAS_JSONSCHEMA, reason="jsonschema package required")
def test_workflow_validates_against_schema():
    """issue-delivery.yaml validates against workflow.schema.json."""
    with open(SCHEMA_PATH) as f:
        schema = json.load(f)
    jsonschema.validate(load_workflow(), schema)


def test_o4_state_machine_shape():
    """initial_state intake; exactly the O4-leg states; correct terminals."""
    spec = load_workflow()["spec"]
    assert spec["initial_state"] == "intake"
    assert set(spec["states"]) == O4_STATES
    for name, state in spec["states"].items():
        if name in TERMINAL_STATES:
            assert state.get("type") == "terminal", f"{name} must be terminal"
            assert not state.get("on"), f"terminal {name} must not transition"
        else:
            assert state.get("type", "normal") == "normal"
            assert state.get("on"), f"{name} needs outbound transitions"


# ── Scenario: Plan state calls the planner with its exact contract ──────────


def test_plan_action_matches_planner_capability_contract():
    """The plan action invokes identify_next_issue with the planner
    manifest's exact input/output contract (canvas O4 / ADR-028)."""
    plan_actions = load_workflow()["spec"]["states"]["plan"]["actions"]
    assert len(plan_actions) == 1
    action = plan_actions[0]
    assert action["capability"] == "identify_next_issue"

    planner_caps = {c["name"]: c for c in load_planner()["spec"]["capabilities"]}
    cap = planner_caps["identify_next_issue"]

    assert set(action["input"]) == set(cap["input_schema"]["required"])

    mapped_outputs = {
        re.search(r"\.result\.(\w+)", v).group(1) for v in action["output"].values()
    }
    assert mapped_outputs == set(cap["output_schema"]["required"])


def test_routing_targets_are_registered_expert_agentdefs():
    """Every expert the route table can select exists as an O2 AgentDef."""
    route = load_workflow()["spec"]["states"]["route"]
    experts = {t["set"]["context.expert"] for t in route["on"]}
    available = {p.stem for p in EXPERTS_DIR.glob("*.yaml")}
    assert experts <= available, f"unknown experts: {experts - available}"
    # The fallback (last transition) must be unguarded → route never dead-ends.
    assert "guard" not in route["on"][-1]
    assert route["on"][-1]["set"]["context.expert"] == "planning-task-split"
    # route is action-less: it must rely on the engine's sentinel-event pass.
    assert "actions" not in route


# ── Scenario: fixture issue → correct {next_issue, expert, blocked_by} ──────


def test_fixture_issue_emits_o4_decision():
    """Given a fixture issue, the workflow emits the correct
    {next_issue, expert, blocked_by} decision (acceptance criterion 1)."""
    final, ctx = run_o4_leg(
        "feat(automation): issue-intake + planning Workflow", PLANNER_OK
    )
    assert final == "routed"
    assert ctx["next_issue"] == "1099"
    assert ctx["expert"] == "planning-task-split"
    assert ctx["blocked_by"] == ""


# ── Scenario Outline: routing table selects one expert per issue class ──────


@pytest.mark.parametrize(
    ("title", "expert"),
    [
        ("feat(protos): add memory service RPC", "api-contract"),
        ("fix(task-broker): lease renewal race", "persistence-state"),
        ("ci(actions): pin runner image digests", "ci-release"),
        ("fix(security): rotate cosign signing key", "security-supply-chain"),
        ("test(bdd): cover dispatch timeout path", "qa-bdd"),
        ("docs(adr): record engine decision", "arch-adr"),
        ("docs(readme): refresh quickstart", "docs-agents"),
        ("chore(automation): tidy milestone state", "planning-task-split"),
    ],
)
def test_routing_table_selects_expert(title, expert):
    """Classify + route path picks exactly one expert per issue class."""
    final, ctx = run_o4_leg(title, PLANNER_OK)
    assert final == "routed"
    assert ctx["expert"] == expert


# ── Scenario: dependency-blocked plan ends in blocked ────────────────────────


def test_blocked_plan_ends_in_blocked_state():
    """next_issue == 0 routes to the blocked terminal with blocked_by set."""
    final, ctx = run_o4_leg(
        "feat(protos): add memory service RPC",
        {
            "next_issue": 0,
            "blocked_by": [1097, 1098],
            "ready_batch": [],
            "rationale": "O3/O4 prerequisites still open",
        },
    )
    assert final == "blocked"
    assert ctx["blocked_by"] == "1097, 1098"
    assert "expert" not in ctx
