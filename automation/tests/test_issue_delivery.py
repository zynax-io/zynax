# SPDX-License-Identifier: Apache-2.0
# automation/tests/test_issue_delivery.py
#
# EPIC #881: asserts the intake → plan → route leg (O4, #1099) and the
# inject → implement → verify → decide delivery leg (O6, #1101) of the
# issue-delivery Workflow (automation/workflows/issue-delivery.yaml).
# Binds the scenarios in automation/tests/features/issue_delivery.feature.
#
# The routing table and the delivery contracts live declaratively in the
# manifest (first-match-wins guarded transitions — mirroring /m6-orchestrate
# STEP 5 and the engine's resolveTransition semantics), so these tests
# evaluate the manifest itself against fixture issues and stubbed capability
# results: no running platform is required (the live e2e is O8, #1103).

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

DELIVERY_STATES = {
    "intake",
    "plan",
    "route",
    "inject",
    "implement",
    "verify",
    "decide",
    "blocked",
    "failed",
}
TERMINAL_STATES = {"decide", "blocked", "failed"}

# prohibited_auto_actions — verbatim from Wave 2
# (docs/archive/dev-advisory/orchestrator/config.yaml, ADR-028 / canvas §S).
NEVER_AUTO = [
    "merge",
    "push",
    "bump-dependency",
    "close-issue",
    "delete-branch",
    "force-push",
]

# Every capability this manifest may dispatch. None is destructive: the
# delivery leg reads, implements in an isolated workspace, verifies, records,
# and emits — it never merges/pushes/closes anything (AC: destructive actions
# never auto-execute).
ALLOWED_CAPABILITIES = {
    "read_issue",
    "identify_next_issue",
    "resolve_context_slice",
    "review",
    "run_verification_gates",
    "record_decision",
    "emit_next_issue",
}


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
    planner reply. Returns (state, ctx): "inject" when the issue was routed
    (the delivery leg starts there — O6), else a terminal state."""
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

    assert current == "inject" or states[current].get("type") == "terminal"
    return current, ctx


def run_delivery_leg(issue_title, planner_result, gates_pass=True):
    """Walk the full delivery leg (O6): intake → plan → route → inject →
    implement → verify → decide, with stubbed capability results standing in
    for the runtime providers. Returns (final_state, ctx)."""
    spec = load_workflow()["spec"]
    states = spec["states"]
    current, ctx = run_o4_leg(issue_title, planner_result)
    assert current == "inject", f"routing leg did not reach inject: {current}"

    # inject: resolve_context_slice output mappings populate the context.
    ctx["expert_agent_id"] = f"agent-{ctx['expert']}"
    ctx["context_slice_files"] = "state/current-milestone.md"
    ctx["context_slice_max_tokens"] = "3000"
    t = first_matching_transition(
        states[current], "resolve_context_slice.completed", ctx
    )
    _apply_set(t, ctx)
    current = t["goto"]

    # implement: the routed expert's review output populates the context.
    assert current == "implement"
    ctx["change_summary"] = "implemented the issue in an isolated workspace"
    ctx["recommended_actions"] = "open draft PR"
    ctx["implement_confidence"] = "high"
    ctx["implement_flags"] = ""
    t = first_matching_transition(states[current], "review.completed", ctx)
    _apply_set(t, ctx)
    current = t["goto"]

    # verify: run_verification_gates output populates the context.
    assert current == "verify"
    ctx["gates_passed"] = "true" if gates_pass else "false"
    ctx["gate_report"] = "validate-spec ok; lint ok; test ok"
    ctx["diff_summary"] = "1 file changed"
    ctx["changed_files"] = "automation/workflows/issue-delivery.yaml"
    t = first_matching_transition(
        states[current], "run_verification_gates.completed", ctx
    )
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


def test_state_machine_shape():
    """initial_state intake; exactly the O4+O6 states; correct terminals."""
    spec = load_workflow()["spec"]
    assert spec["initial_state"] == "intake"
    assert set(spec["states"]) == DELIVERY_STATES
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
    # Every routing edge enters the delivery leg (O6) at inject.
    assert all(t["goto"] == "inject" for t in route["on"])


# ── Scenario: fixture issue → correct {next_issue, expert, blocked_by} ──────


def test_fixture_issue_emits_o4_decision():
    """Given a fixture issue, the workflow emits the correct
    {next_issue, expert, blocked_by} decision (O4 acceptance criterion 1).
    Since O6 the routed decision enters the delivery leg at inject."""
    final, ctx = run_o4_leg(
        "feat(automation): issue-intake + planning Workflow", PLANNER_OK
    )
    assert final == "inject"
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
    assert final == "inject"
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


# ═════════════════════════════ O6 — delivery leg ════════════════════════════

PLANNER_O6 = {
    "next_issue": 1101,
    "blocked_by": [],
    "ready_batch": [1101],
    "rationale": "O6 is the only dependency-free Wave 4 story",
}

O6_TITLE = "feat(automation): delivery leg of issue-delivery"


def _single_action(state_name):
    actions = load_workflow()["spec"]["states"][state_name]["actions"]
    assert len(actions) == 1
    return actions[0]


# ── Scenario: Inject resolves the routed expert's context-slice binding ─────


def test_inject_resolves_context_slice_binding():
    """inject resolves the routed expert's declared slice for the decision
    log; the authoritative binding stays in task-broker at dispatch (O5)."""
    action = _single_action("inject")
    assert action["capability"] == "resolve_context_slice"
    assert set(action["input"]) == {"expert", "capability"}
    assert action["input"]["expert"] == "{{ .ctx.expert }}"
    assert action["input"]["capability"] == "review"
    mapped = {
        re.search(r"\.result\.(\w+)", v).group(1) for v in action["output"].values()
    }
    assert mapped == {"agent_id", "files", "max_tokens"}


# ── Scenario: Implement drives exactly the routed expert's review ───────────


def test_implement_dispatches_routed_expert_review():
    """implement drives one expert-keyed review dispatch matching the expert
    AgentDef contract — context_slice deliberately absent (O5 binds it)."""
    action = _single_action("implement")
    assert action["capability"] == "review"
    # Expert-keyed dispatch: task-broker narrows to exactly ctx.expert.
    assert action["input"]["expert"] == "{{ .ctx.expert }}"

    planner_caps = {c["name"]: c for c in load_planner()["spec"]["capabilities"]}
    review = planner_caps["review"]
    required = set(review["input_schema"]["required"])
    # All required review inputs are supplied except context_slice, which the
    # task-broker injection binding (O5) supplies from the registry-declared
    # slice — strict isolation (ADR-028).
    assert required - {"context_slice"} <= set(action["input"])
    assert "context_slice" not in action["input"]
    # trigger must be a member of the review contract's enum.
    trigger_enum = review["input_schema"]["properties"]["trigger"]["enum"]
    assert action["input"]["trigger"] in trigger_enum


def test_context_slice_never_inlined_anywhere():
    """No action in the manifest inlines literal context_slice content — the
    expert manifests stay the single source of truth (ADR-028). The only
    context_slice allowed anywhere is the decide record's *resolved* slice:
    pure '{{ .ctx.* }}' references written by inject — durable evidence of
    the O5 binding, never a literal slice a caller could plant."""
    for name, state in load_workflow()["spec"]["states"].items():
        for action in state.get("actions", []):
            slice_input = action.get("input", {}).get("context_slice")
            if slice_input is None:
                continue
            assert action["capability"] == "record_decision", name
            assert all(
                re.fullmatch(r"\{\{ \.ctx\.\w+ \}\}", str(v))
                for v in slice_input.values()
            ), f"literal slice content inlined in {name}: {slice_input}"


# ── Scenario: A verified change reaches decide with a decision-log row ──────


def test_happy_path_reaches_decide_with_decision_log():
    """The full delivery leg ends in the decide terminal: durable
    record_decision + next-issue CloudEvent (O6 acceptance criterion 1)."""
    final, ctx = run_delivery_leg(O6_TITLE, PLANNER_O6, gates_pass=True)
    assert final == "decide"
    assert ctx["delivery_outcome"] == "success"

    actions = {
        a["capability"]: a
        for a in load_workflow()["spec"]["states"]["decide"]["actions"]
    }
    assert set(actions) == {"record_decision", "emit_next_issue"}

    record = actions["record_decision"]["input"]
    assert record["workflow"] == "issue-delivery"
    assert record["issue_number"] == "{{ .ctx.issue_number }}"
    assert record["expert"] == "{{ .ctx.expert }}"
    assert record["next_issue"] == "{{ .ctx.next_issue }}"
    # The recorded row carries the resolved slice — durable evidence of the
    # injection binding.
    assert set(record["context_slice"]) == {"files", "max_tokens"}

    emit = actions["emit_next_issue"]["input"]
    assert emit["event"] == "zynax.automation.issue_delivery.next_issue"
    assert emit["completed_issue"] == "{{ .ctx.issue_number }}"
    assert emit["next_issue"] == "{{ .ctx.next_issue }}"


# ── Scenario: Failing gates still record a durable decision ─────────────────


def test_failing_gates_still_record_decision():
    """Gates failing is still a decision: the run ends in decide with
    delivery_outcome gates_failed — a decision-log row on every path."""
    final, ctx = run_delivery_leg(O6_TITLE, PLANNER_O6, gates_pass=False)
    assert final == "decide"
    assert ctx["delivery_outcome"] == "gates_failed"


# ── Scenario: failure events route to the failed terminal ───────────────────


@pytest.mark.parametrize(
    ("state", "event", "reason_fragment"),
    [
        ("inject", "resolve_context_slice.failed", "inject"),
        ("implement", "review.failed", "implement"),
        ("implement", "review.timeout", "implement"),
        ("verify", "run_verification_gates.failed", "verify"),
    ],
)
def test_capability_failures_route_to_failed(state, event, reason_fragment):
    """Every delivery-leg capability failure routes to the failed terminal
    with an explanatory failure_reason."""
    ctx = {}
    t = first_matching_transition(load_workflow()["spec"]["states"][state], event, ctx)
    assert t["goto"] == "failed"
    assert reason_fragment in t["set"]["context.failure_reason"]


# ── Scenario: destructive actions are never auto-executed ───────────────────


def test_prohibited_auto_actions_honoured():
    """O6 acceptance criterion 2: destructive actions never auto-execute.
    (a) every capability the manifest dispatches is non-destructive;
    (b) the recorded decision carries never_auto verbatim from Wave 2 and
    declares human_action_required — landing the change stays human."""
    states = load_workflow()["spec"]["states"]
    dispatched = {
        a["capability"] for s in states.values() for a in s.get("actions", [])
    }
    assert dispatched <= ALLOWED_CAPABILITIES, dispatched - ALLOWED_CAPABILITIES
    assert not dispatched & set(NEVER_AUTO)

    record = next(
        a for a in states["decide"]["actions"] if a["capability"] == "record_decision"
    )["input"]
    assert record["never_auto"] == NEVER_AUTO
    assert record["human_action_required"] == "true"
