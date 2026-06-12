<!-- SPDX-License-Identifier: Apache-2.0 -->

# Dev-Automation EPIC — Status and Architecture Direction

> **EPIC:** [#873](https://github.com/zynax-io/zynax/issues/873) M6.DevAuto  
> **Generated:** 2026-06-03 from live repo state at HEAD of `main` (post PR #872)  
> **Location:** `automation/STATUS-AND-DIRECTION.md` — dedicated automation folder,
> outside all AGENTS.md globs (proof: no `automation/` path appears in any layer rule file).  
> **Consumer:** Read explicitly by GH Actions workflows and (aspirational) Zynax agent runtime.
> Never auto-loaded as ambient context.

---

## Status update (2026-06-12) — read this first

This document is the **original 2026-06-03 architecture direction**, kept for design
rationale and wave history. Since it was generated:

- **Waves 0–3 were superseded and retired** (#1107, #1129) — configs archived under
  `docs/archive/dev-advisory/`; the near-term plane is now the generalized Claude Code
  delivery commands (EPIC #1108). Wave 3's per-merge mesh was demoted to `weekly-audit.yml`.
- **Wave 4 prerequisites landed:** M6.H #626 (Postgres repos) ✅ and M6.I #772 (EventBus
  over NATS JetStream) ✅.
- **Wave 4 (EPIC #881) was delivered to the platform-readiness boundary:** ADR-028 split
  (O1 #1096), 9 expert AgentDefs (O2 #1097), orchestrator Workflow (O3 #1098),
  issue-delivery Workflow intake→plan→route (O4 #1099) + delivery leg
  inject→implement→verify→decide (O6 #1101), context-slice injection binding in
  task-broker (O5 #1100), learning-synthesizer AgentDef (O7 #1102), status reconcile
  (O9 #1104).
- **The live `zynax apply` e2e (O8 #1103) is deferred to M7** — four code-verified
  platform gaps: workflow-compiler rejects `output:` (deferred to M7+ in `manifest.go`),
  Go-template guards vs CEL evaluation (fail-closed), no capability providers for
  review/aggregate/act/notify/record, no gateway outputs/decision-log read path. The
  strict xfail in `automation/tests/test_platform_readiness.py` remains the honest gate.
  EPICs #873/#881 closed at this boundary; #1103 continues under M7.

Sections below describing Waves 0–3 as runnable, `automation/experts/` +
`automation/orchestrator/` as live paths, or Wave 4 as blocked on M6.H/M6.I are
**historical**. Current entry point: [`README.md`](README.md) · canvas:
`docs/spdd/881-self-hosted-issue-delivery/canvas.md`.

---

## 1. PR + Git-History Synthesis

### Architecture direction

The Zynax project follows a strict three-layer hexagonal model (ADR-001, AGENTS.md):
- **L1 (YAML):** declarative agent/workflow definitions in `spec/`
- **L2 (gRPC contracts):** versioned proto contracts in `protos/zynax/v1/`
- **L3 (execution):** Go services in `services/`, Python adapters in `agents/`

Cross-layer coupling is a hard blocker at review. All inter-service calls are gRPC
(ADR-001). All adapters call platform services via gRPC stubs only (ADR-013). No shared
databases (ADR-008). Pluggable workflow engines behind the `WorkflowEngine` interface
(ADR-015). BDD `.feature` files committed before implementation (ADR-016). SPDD canvas
required before any `feat:` PR (ADR-019). All merges: branch → PR → squash → delete
branch (ADR-023).

### Milestone trajectory (cited evidence)

| Phase | Key decisions | ADRs | Notable reversals |
|-------|--------------|------|------------------|
| M1–M2 | Proto contracts first; BDD scenarios before implementation | ADR-001, ADR-016 | — |
| M3–M4 | task-broker + agent-registry slipped from M3/M4 to M5.C | ADR-008 | Partial delivery flagged in state/current-milestone.md |
| M5 | Adapter library (6 adapters); truth pass removed false security claims (SECURITY.md); GOWORK=off mandate (ADR-017) | ADR-017 | PR #775 closed unmerged; SECURITY.md false mTLS/SBOM/cosign claims removed |
| M5→M6 | SBOM+cosign (#489, PR #833 merged); mTLS (#488, PR #831 merged); stateless compiler (#490, PR #774 merged); health probes (#487, PR #821 merged) | ADR-020, ADR-022, ADR-023 | Merge-queue (#544) removed as CI gate |
| M6 current | Helm library chart A.0 merged (PR #872); ADR-022 accepted (event-bus = gRPC wrapper over NATS); M6.Helm A.1–A.13 pending | ADR-022, ADR-023 | — |

### M6 EPIC ledger (at time of this doc)

| EPIC | Issue | Status | Canvas |
|------|-------|--------|--------|
| M6.A Health probes | #463 | ✅ Complete (PR #821) | Implemented |
| M6.B mTLS | #464 | ✅ Complete (PR #831) | Implemented |
| M6.C Supply chain | #465 | ✅ Complete (PR #833) | Implemented |
| M6.D Stateless compiler | #466 | ✅ Complete (PR #774) | Implemented |
| M6.Helm | #765 | 🔄 In progress (A.0 merged, A.1–A.13 pending) | Aligned |
| M6.H Postgres repos | #626 | ⬜ Designed, no implementation | Aligned |
| M6.F Config convergence | #670 | ⬜ No canvas yet | — |
| M6.Images SoT | #855 | ⬜ Canvas aligned, O1–O7 pending | Aligned |
| M6.Build | #837 | ⬜ SPDD-exempt | — |
| M6.NS Multi-namespace | #767 | ⬜ Canvas aligned | Aligned |
| M6.Argo | #766 | ⬜ Canvas aligned | Aligned |
| M6.SDK PyPI | #769 | ⬜ Canvas aligned | Aligned |
| M6.Policy | #768 | ⬜ Canvas aligned | Aligned |
| M6.J memory-service | #773 | ⬜ Canvas aligned, BLOCKED on M6.H | Aligned |
| M6.I event-bus | #772 | ⬜ ADR-022 accepted; stories #823–#828 created; no canvas yet | — |
| M6.G e2e harness | #770 | ⬜ Canvas aligned, BLOCKED on A+I+J+B | Aligned |
| **M6.DevAuto** | **#873** | ⬜ **This EPIC** | See §5 |

---

## 2. Current Capability Reality

**These gaps determine what the aspirational plane may and may not rely on.**

| Capability | Status | Blocker |
|------------|--------|---------|
| **State persistence** | ❌ In-memory only | M6.H #626 (Postgres-backed task-broker + agent-registry) not implemented |
| **EventBusService** | ❌ Log-only stub | M6.I #772 (NATS JetStream implementation) not started; ADR-022 just accepted |
| **Capability autodiscovery** | Partial: 6 adapters registered (http, git, ci, llm, langgraph, echo); round-robin dispatch | No dynamic orchestration; no fan-out protocol |
| **CEL evaluation** | ✅ Implemented (not fail-open; missing guard = error) | — |
| **SDK (Python)** | ✅ Implemented (Option A, `agents/sdk/`; docstrings step 2 pending #376) | — |
| **mTLS** | ✅ Implemented (PR #831) | — |
| **SBOM + cosign** | ✅ Implemented (PR #833) | — |
| **Helm charts** | ❌ None yet (A.0 zynax-lib merged; A.1–A.13 pending) | M6.Helm #765 |
| **Postgres-backed repos** | ❌ Not implemented | M6.H #626 |
| **gRPC health checking** | ❌ Not implemented | M6.A stories pending (#656) |

**Implication for Wave 4:** The orchestrator+expert AgentDef approach requires durable
state persistence (workflows survive restart) and durable async messaging (expert fan-out
via EventBusService). Neither exists yet. Wave 4 is strictly aspirational until M6.H and
M6.I are both complete.

---

## 3. What's Coming in M6 (Relevant to This EPIC)

| M6 EPIC | DevAuto relevance |
|---------|-----------------|
| **M6.Images SoT (#855)** | Wave 3 drift-check expert consumes `images/images.yaml` (O1 of #855); drift-check expert integration blocked until O1–O2 ship |
| **M6.Build (#837)** | Wave 3 image-test expert runs on newly-built images; native multi-arch removes QEMU overhead from test runner |
| **M6.H (#626)** | **Wave 4 prerequisite #1** — Postgres-backed repos provide durable workflow state |
| **M6.I (#772)** | **Wave 4 prerequisite #2** — EventBusService provides durable fan-out between orchestrator and experts |
| **M6.G (#770)** | The e2e harness tests the platform that Wave 4 runs on; M6.G completion is a soft prerequisite for Wave 4 confidence |

---

## 4. Existing Automation Surface (What We Build On)

### GitHub Actions workflows (14 total)

| Workflow | Purpose | DevAuto builds on |
|----------|---------|------------------|
| `ci.yml` | Go + Python unit tests, lint, security | Wave 3 security-rescan expert extends this |
| `pr-checks.yml` | Proto lint, validate-spec, pr-size | Wave 0 adds `dev-advisory.yml` alongside |
| `pr-size.yml` | Line-count gate ≤900 | Wave 2 orchestrator reads this verdict |
| `service-release.yml` | Build + push all 7 service images to GHCR | Wave 3 image-test expert triggers on completion |
| `tools-image.yml` | Tools container build + auto-bump issue | Wave 3 image-test expert covers this image too |
| `release.yml` | Unified release orchestration | Wave 3 security-rescan covers release artifacts |
| `ai-context-budget.yml` | AI context budget threshold check | Complements Wave 0 expert context-slice budgets |

### Security tooling already in place

- **Scorecard** (`ossf/scorecard-action`) — runs on main pushes; feeds Wave 3 security expert baseline
- **CodeQL** — implicit via GH Advanced Security; Wave 3 security expert monitors alerts
- **cosign** — release artifacts signed (PR #833); Wave 3 image-test expert verifies signatures
- **SBOM (SPDX)** — generated via `anchore/sbom-action` (PR #833); Wave 3 references these
- **govulncheck + bandit + pip-audit** — `make security` in Docker; Wave 3 security-rescan runs this

### Branch protection and merge policy

ADR-023: all changes via branch → PR → squash-merge → delete branch. No direct main pushes.
Required signatures on commits (SSH). DCO `Signed-off-by` on every commit. `Assisted-by: Claude/<model>` (not `Co-Authored-By`).

---

## 5. Two-Plane Separation (The Core Design Constraint)

**The seductive failure** is wiring aspirational "Zynax runs itself" automation into
`main` CI as though the platform supports it — producing green-looking workflows built
on capabilities that don't exist yet. This EPIC explicitly avoids that.

```
┌─────────────────────────────────────────────────────────────┐
│  NEAR-TERM PLANE (Waves 0–3)                                │
│  GitHub Actions + Claude Code subagents                     │
│  Runnable today. No Zynax runtime dependency.               │
│                                                             │
│  Wave 0: Advisory — expert subagents on PR events           │
│  Wave 1: Orchestrated advisory — single aggregated comment  │
│  Wave 2: Gated automation — non-destructive actions only    │
│  Wave 3: Post-merge completeness mesh                       │
└────────────────────────────┬────────────────────────────────┘
                             │
         automation/tests/test_platform_readiness.py
         @pytest.mark.xfail(strict=True, ...)
         ← THE HONEST LINE BETWEEN PLANES →
         Fails today. Passes only when M6.H + M6.I complete.
                             │
┌────────────────────────────▼────────────────────────────────┐
│  ASPIRATIONAL PLANE (Wave 4)                                │
│  Zynax AgentDef workflows running on Zynax itself           │
│  BLOCKED: needs M6.H (Postgres) + M6.I (event-bus)         │
│  GATED: failing test must flip to pass first                │
│                                                             │
│  Wave 4: Orchestrator+experts as AgentDef workflows         │
└─────────────────────────────────────────────────────────────┘
```

Every automation asset in this folder carries an explicit plane label. Anything
needing an unbuilt Zynax capability lives behind the failing test — never wired
into main CI as if it works.

---

## 6. Expert Mesh Design

### 6.1 Context-split taxonomy

The orchestrator fans out to 9 context-scoped expert agents. Each expert reasons over
its own bounded slice of the codebase and returns a structured output. No expert
exceeds its declared `max_tokens` budget.

| Expert | Context slice | max_tokens |
|--------|--------------|-----------|
| **Orchestrator** | PR metadata, git diff summary, M6 state, all expert outputs | 8000 |
| **Architecture/ADR** | AGENTS.md (all), docs/adr/INDEX.md + referenced ADRs, changed file list | 4000 |
| **Persistence/state** | services/*/internal/domain/, migration files, ADR-008, ADR-021 | 3000 |
| **API/contract** | protos/, spec/schemas/, spec/asyncapi/, capability registrations | 4000 |
| **Security/supply-chain** | .github/workflows/ security steps, SECURITY.md, Scorecard output | 3000 |
| **QA/BDD** | protos/tests/ (feature files), coverage reports, ADR-016 | 3000 |
| **Docs/AGENTS** | AGENTS.md (all layers), README, ARCHITECTURE.md, CONTRIBUTING.md | 3000 |
| **CI/release** | .github/workflows/, images/images.yaml (post-M6.Images O1), release pipeline | 3000 |
| **Planning/task-split** | state/current-milestone.md, M6-planning.md, open issues (last 50) | 3000 |

### 6.2 Input/output contracts (all experts)

**Input (from orchestrator fan-out):**
```yaml
trigger: pull_request | push | post_merge
pr_number: int          # if trigger == pull_request
diff_summary: string    # git diff --stat output
changed_files: []       # list of changed file paths
context_slice: string   # contents filtered to this expert's files glob
```

**Output (every expert must produce):**
```yaml
summary: string           # ≤200 words, plain text, no markdown
recommended_actions: []   # ordered list; each: { action, rationale, priority }
reasons_decisions: []     # reasoning chain for each recommendation
confidence: low|medium|high
flags: []                 # tier-2 security flags or blocker flags (empty if none)
```

### 6.3 Aggregation/decision protocol

**Orchestrator decision algorithm:**
1. Collect all 9 expert outputs
2. For each recommended action, tally expert support: weight by confidence level
   (high=3, medium=2, low=1) and expert type (security/arch/contract = weight ×1.5)
3. Actions with aggregate weight ≥ threshold → include in aggregated verdict
4. **Escalate to human when:**
   - ≥2 high-confidence experts recommend contradictory actions
   - Any expert raises a `flags` entry (tier-2 security or blocker flag)
   - Aggregate confidence for the top-recommended action is `low`
5. **Never auto-act on:** merge, push, bump-dependency, close-issue, delete-branch
6. **Record in decision-log:** every orchestrator run produces a JSON artifact
   (schema: `automation/orchestrator/decision-log-schema.yaml`) stored as a CI artifact

---

## 7. Wave Model

### Wave 0 — Advisory/CI-only (near-term)

- **Plane:** Near-term
- **Trigger:** `pull_request` events (opened, synchronize, reopened)
- **What runs:** 9 expert subagents in parallel (Claude Code invocations from GH Actions)
- **Output:** Individual job summaries + single collated PR comment (advisory only)
- **Human boundary:** Human reads and decides all actions
- **Zynax capabilities needed:** None — pure Claude Code + GH Actions
- **Runnable today:** Yes
- **File:** `.github/workflows/dev-advisory.yml`
- **Issue:** #877 (DevAuto.4)

### Wave 1 — Orchestrated Advisory (near-term)

- **Plane:** Near-term
- **Trigger:** After all Wave 0 expert jobs complete (needs pattern)
- **What runs:** Orchestrator subagent aggregates expert outputs → single decision-support comment
- **Output:** Aggregated verdict + decision-log JSON artifact + escalation flag if needed
- **Human boundary:** Human decides all actions; escalation flag prompts human attention
- **Zynax capabilities needed:** None
- **Runnable today:** Yes (extends Wave 0 workflow)
- **Issue:** #878 (DevAuto.5)

### Wave 2 — Gated Automation (near-term)

- **Plane:** Near-term
- **Trigger:** After Wave 1 orchestrator completes
- **What runs:** Orchestrator executes `auto_allowed` actions (auto-label, auto-assign, draft-issue, request-changes)
- **Output:** Logged actions in decision-log; PR updated
- **Human boundary:** Merge and push are explicitly prohibited. All actions logged and reversible.
- **Zynax capabilities needed:** None
- **Runnable today:** Yes (extends Wave 1)
- **Risks:** Over-labelling noise (mitigation: high-confidence threshold), `request-changes` abuse (mitigation: security expert only)
- **Issue:** #879 (DevAuto.6)

### Wave 3 — Post-Merge Completeness Mesh (near-term)

> **Demoted (#1113):** the per-merge mesh (`post-merge-completeness.yml`) is now a
> schedule-only `weekly-audit.yml` that fails loudly instead of auto-filing `[AUTO]` issues.
> The description below is the original Wave 3 design, kept for historical context.

- **Plane:** Near-term
- **Trigger:** `workflow_run` on `service-release.yml` completion + `push` to main
- **What runs:** 4 post-merge experts (image-test, integration, drift-check, security-rescan) + completeness-verdict aggregator
- **Output:** Commit status (pass/fail) + decision-log artifact + auto-created issues on failure
- **Human boundary:** Humans fix the auto-created issues; no automated fixes
- **Zynax capabilities needed:** None
- **Drift-check dependency:** Integrates with M6.Images SoT `images/images.yaml` (after #856 merges)
- **Issue:** #880 (DevAuto.7)

### Wave 4 — Self-Hosted Aspirational (aspirational plane)

> **Delivered to boundary (2026-06-12):** manifests + bindings shipped (O1–O7 + O9 of
> EPIC #881, per ADR-028); the live-platform e2e (O8) is deferred to M7 — see #1103 and
> the status update at the top of this file. The bullet list below is the original design.

- **Plane:** Aspirational
- **What would run:** Orchestrator + 9 experts as Zynax `kind: AgentDef` workflows
- **Zynax capabilities needed:**
  - **M6.H #626** (Postgres-backed repos) — durable state so workflow survives restart
  - **M6.I #772** (EventBusService) — durable fan-out between orchestrator and experts
  - All Helm charts deployed (M6.Helm #765) — running platform
  - M6.G e2e harness (#770) — confidence the platform works end-to-end
- **Gate:** `automation/tests/test_platform_readiness.py` `xfail` must flip to pass
- **NOT wired into main CI** until the failing test passes
- **Issue:** #881 (DevAuto.8)

---

## 8. Post-Merge Completeness Agents (Wave 3 Detail)

> **Demoted (#1113):** these agents now run as the schedule-only `weekly-audit.yml`
> (no per-merge triggers, no `[AUTO]` issue creation — a red run is the signal).

All post-merge agents are in the **near-term plane** (GitHub Actions).

| Agent | Trigger | On failure |
|-------|---------|-----------|
| **Image-test** | After `service-release.yml` completes | Opens `chore(ci): image <svc> failed smoke test / exceeded size budget` issue |
| **Integration** | Push to main touching `services/` or `infra/` | Opens `fix(ci): integration test failure after merge — <commit-sha>` issue |
| **Drift-check** | Every main push | Opens `chore(ci): images.yaml drift detected — <image>:<tag>` issue (requires M6.Images O1 #856) |
| **Security-rescan** | Weekly + on `services/` or dependency merge | Opens `fix(security): new govulncheck/bandit/pip-audit finding — <package>` issue |
| **Completeness-verdict** | After all 4 agents complete | Posts commit status `dev-automation/completeness: pass/fail`; feeds back to decision-log |

Auto-created issues use `[AUTO]` prefix in title (distinguishes from human-authored issues).
Labels: `type: <bug|chore>`, `area: ci`, `milestone: M6`, `status: needs-triage`, `priority: medium`.

---

## 9. The Failing Test

> **Update (2026-06-12):** the schema-validation tests in this file now pass (O2/O3 of
> #881); only `test_orchestrator_executes_on_platform` remains `xfail(strict=True)`.
> Its flip is O8 (#1103), deferred to M7 — the remaining blockers are the four platform
> gaps listed in the status update at the top of this file, not M6.H/M6.I.

**Location:** `automation/tests/test_platform_readiness.py`  
**Framework:** pytest with `@pytest.mark.xfail(strict=True, ...)`  
**Issue:** #882 (DevAuto.9)

```python
@pytest.mark.xfail(
    strict=True,
    reason=(
        "Wave 4 aspirational: requires M6.H (Postgres-backed repos, #626) + "
        "M6.I (event-bus implementation, #772) + automation/workflows/ AgentDef "
        "YAMLs authored in DevAuto.8 (#881). Fails until all three land."
    )
)
class TestPlatformReadiness:
    def test_orchestrator_agentdef_schema_valid(self): ...
    def test_expert_agentdefs_schema_valid(self): ...
    def test_orchestrator_executes_on_platform(self, zynax_client): ...
```

**Why it fails today (3 independent reasons):**
1. `automation/workflows/` does not exist — AgentDef YAMLs authored in #881 (blocked on M6.H+I)
2. task-broker + agent-registry use in-memory repos — workflow state lost on restart
3. EventBusService is a log-only stub — cannot deliver messages between orchestrator and experts

**Tracking strategy:** `strict=True` means:
- XFAIL → build green ✅ (expected failure, test ran and failed as expected)
- XPASS → build red ❌ (unexpected pass alerts the team — something changed)
- SKIP → not acceptable (silently hides the gate)

**Flip condition:** Remove `@pytest.mark.xfail` when M6.H + M6.I are merged AND
`automation/workflows/` AgentDef YAMLs exist. The test then runs clean and green.
Open a `test(automation): wave4 gate passes — remove xfail marker` PR at that point.

---

## 10. File Layout Proof (automation/ is NOT ambient context)

> **Update (2026-06-12):** `experts/` and `orchestrator/` were archived to
> `docs/archive/dev-advisory/` in #1129; `workflows/` now holds the delivered Wave 4
> manifests. Current layout: see [`README.md`](README.md). The non-ambient-context
> proof below still holds.

```
automation/             ← THIS FOLDER — not in any AGENTS.md glob
├── STATUS-AND-DIRECTION.md    ← this file (living architecture doc)
├── README.md                  ← folder entry point (DevAuto.10, #883)
├── experts/                   ← expert YAML configs (DevAuto.2, #875)
│   ├── schema.yaml
│   ├── orchestrator.yaml
│   ├── arch-adr.yaml
│   ├── persistence-state.yaml
│   ├── api-contract.yaml
│   ├── security-supply-chain.yaml
│   ├── qa-bdd.yaml
│   ├── docs-agents.yaml
│   ├── ci-release.yaml
│   └── planning-task-split.yaml
├── orchestrator/              ← orchestrator config (DevAuto.3, #876)
│   ├── config.yaml
│   ├── decision-log-schema.yaml
│   └── aggregation-protocol.md
├── workflows/                 ← AgentDef YAMLs (DevAuto.8, #881 — aspirational)
│   ├── dev-advisory-orchestrator.yaml
│   └── experts/
└── tests/                     ← failing platform-readiness test (DevAuto.9, #882)
    ├── test_platform_readiness.py
    ├── conftest.py
    └── requirements.txt
```

**Proof no AGENTS.md glob loads `automation/`:**
- Root `AGENTS.md` — constitution + index table; no glob patterns; no path auto-load
- `services/AGENTS.md` — covers `services/` subtree only
- `agents/AGENTS.md` — covers `agents/` subtree only
- Layer-1 YAML is consumed from `spec/` only
- Layer-2 protos from `protos/zynax/v1/` only
- Layer-3 execution from `services/` and `agents/` only

`automation/` is not in any of these paths. ✅

Root `AGENTS.md` gets **one pointer row only** (DevAuto.10, #883):
> `| automation/ | Dev-automation orchestrator + expert mesh configs — consumed explicitly by GH Actions and the Zynax agent runtime; never auto-loaded here |`

---

## 11. Reconciliation Result

No existing open issues for: agent mesh, orchestrator, dev-loop automation, self-hosting,
or expert mesh. This EPIC is net-new. Existing work that this EPIC EXTENDS (not duplicates):

| Existing work | How DevAuto extends it |
|--------------|----------------------|
| M6.Images SoT #855 | Wave 3 drift-check expert integrates with `images/images.yaml` (after O1 #856 ships) |
| `tools-image.yml` (#844) | Wave 3 image-test expert adds post-build smoke test on top of existing build |
| `ai-context-budget.yml` | Wave 0 expert context slices align with this budget gate |
| `make security` (govulncheck+bandit+pip-audit) | Wave 3 security-rescan expert runs this on post-merge |
| ADR-023 (merge discipline) | Wave 2 orchestrator respects this — never auto-merges |

---

*Zynax — automation/STATUS-AND-DIRECTION.md · Apache 2.0*  
*Assisted-by: Claude/claude-sonnet-4-6*
