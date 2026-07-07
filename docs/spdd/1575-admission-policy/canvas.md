# REASONS Canvas — Admission-policy delegation: engine allow-list → ValidatingAdmissionPolicy (M8.G)

> **All content in this Canvas is Tier 1 (public-safe).**
> Run `/lib:spdd-security-review <path>` before committing.

**Issue:** #1575
**Author:** Oscar Gómez Manresa
**Date:** 2026-07-06
**Status:** Implemented

> Story issues: step 1 → #1634 · step 2 → #1635 · step 3 → #1636 · step 4 → #1637

---

## R — Requirements

- **Problem.** The coarse engine allow-list ("this namespace may use temporal but not argo") is
  enforced only inside the workflow-compiler (`checkRoutingPolicy`, REST path). The CR/GitOps
  authoring path shipped by M8.E (ADR-043 `Workflow` CR) **bypasses it entirely** — a CR with a
  disallowed `spec.engine` reconciles and dispatches with no policy check. The compiler also
  carries a **dead** concurrent-invocation quota check: `buildPolicyGate()` always injects a `nil`
  counter, so `checkCapabilityQuota` short-circuits in production — code that advertises
  enforcement that does not exist (ADR-020 zero-trust violation).
- **Done — admission guards the CR path, the dead quota path is gone:**
  - Applying a `Workflow` CR whose `spec.engine` is not in the namespace's allow-list is **denied
    at admission** by a CEL `ValidatingAdmissionPolicy` with a clear message; an allowed or unset
    engine admits → reconciles → dispatches (proven live on kind).
  - VAP + `ValidatingAdmissionPolicyBinding` + params object ship in the api-gateway Helm chart
    behind a values flag (default **off**), enabled in the e2e values so CI exercises them.
  - The kind e2e runs on `kindest/node` ≥ 1.30, where VAP is **GA and default-on**
    (`admissionregistration.k8s.io/v1`) — the bump re-validates the whole e2e matrix.
  - The starved quota check is **removed** from `policy_gate.go` (with its dead
    `ZYNAX_POLICY_MAX_CONCURRENT` config); the orphaned CapabilityQuota BDD scenarios are deleted,
    **not migrated** (ADR-045 §2 — quota is unenforced on both gates until the engine-adapter
    `QuotaChecker` is wired live; migrating a green contract to a never-called component would
    advertise protection that does not exist).
  - `checkRoutingPolicy` **stays live** for the REST path (`zynax apply` → gateway → compiler) —
    the interim dual-guard of ADR-045 §3.
  - ADR-045 is synced to the shipped reality: the VAP reads the CR field **`spec.engine`**
    (M8.E landed the engine hint as a spec field, lifted into the submit `EngineHint` by the
    reconciler), not the `zynax.io/engine-hint` annotation ADR-045 assumed.

---

## E — Entities

```
ValidatingAdmissionPolicy (engine allow-list)   ← NEW: CEL set-membership on object.spec.engine
ValidatingAdmissionPolicyBinding                ← NEW: binds the policy per namespace; paramRef → params
params ConfigMap (allowed-engines)              ← NEW: carries the namespace's AllowedEngines list
Workflow CR (zynax.io/v1alpha1, spec.engine)    ← existing attach point (ADR-043 / M8.E)
workflow-compiler PolicyGate                    ← SHRINKS: routing-only (REST dual-guard); quota path deleted
engine-adapter QuotaChecker                     ← UNTOUCHED: contract-only today; wiring it live is
                                                  explicitly future work (ADR-045 Consequences)
```

---

## A — Approach

**We will:**
- **Bump the kind node image to v1.30.0 first** (story 1): VAP is GA and default-on at 1.30;
  v1.30.0 is the node image validated for the pinned kind binary v0.23.0, so the binary pin is
  untouched. The bump touches `scripts/e2e/cluster-up.sh`, `.github/workflows/e2e-smoke.yml`
  (node image + the lockstep kubectl download), and `scripts/e2e/README.md`. `kindest/node` is
  deliberately **not** added to `images/images.yaml`: the SoT tool stamps `sha256:` digests into
  banner regions, and this is a tag-only pin with no digest consumer — a plain edit, recorded here
  so the omission is intentional, not drift. The whole e2e matrix (temporal + argo) must go green
  on the new node image before anything else lands.
- Delegate **only the coarse engine allow-list** to a CEL `ValidatingAdmissionPolicy` bound to the
  `Workflow` CR: `!has(object.spec.engine) || object.spec.engine == "" ||
  object.spec.engine in <allowed>` — an **unset engine means the platform default and is always
  admitted**. Per-namespace `AllowedEngines` ride in a params object referenced by a
  `ValidatingAdmissionPolicyBinding`; multi-namespace policy = multiple bindings (ADR-045).
- Ship VAP + binding + params in `helm/zynax-api-gateway/templates/` behind
  `admissionPolicy.enabled` (default off), following the chart's `edge.enabled` /
  `crdController.enabled` whole-file-guard idiom. `crds/` is not an option — Helm never templates
  or upgrades that directory. Enable the flag in `scripts/e2e/values-e2e.yaml` so e2e covers it.
- Remove the starved quota path from the compiler: `checkCapabilityQuota`,
  `CapabilityQuotaConfig`, `ActiveInvocationCounter`, `PolicyViolationQuota`, the
  `NewPolicyGate` quota/counter params, the `ResourceExhausted` mapping in the gRPC server, the
  `ZYNAX_POLICY_MAX_CONCURRENT` config, and the two orphaned CapabilityQuota scenarios in
  `protos/tests/features/policy_enforcement.feature` (that feature file has no step definitions —
  the scenarios never executed).
- Sync ADR-045's four annotation-based sentences to `spec.engine` (Context §1, Decision §1,
  Consequences ×2) with a dated correction note, and mark
  `docs/spdd/768-policy-enforcement/canvas.md` superseded for the quota half.
- Live-verify on a fresh kind 1.30 cluster: disallowed-engine CR → denied at admission with the
  VAP message; allowed-engine CR → accepted → reconciles → dispatches.

**We will NOT:**
- Delete or weaken `checkRoutingPolicy` — admission cannot see REST submissions; the compiler
  allow-list is the REST path's only guard until REST is retired (ADR-045 §3 hard sequencing).
- Migrate the CapabilityQuota BDD contract to the engine-adapter `QuotaChecker` — it is never
  constructed in production and has no live caller; a green contract against dead code is a
  false-enforcement guarantee (ADR-020 zero-trust / DoS regression, ADR-045 §2).
- Wire the engine-adapter `QuotaChecker` live — explicitly future work (ADR-045 Consequences);
  quota is treated as unenforced on both gates in this epic.
- Move engine-fit decisioning (capability↔engine matching, hint semantics) into CEL — protected
  custom core (ADR-040 §6). The VAP is structural set-membership only.
- Add Kyverno or OPA-Gatekeeper — the recorded fallback, not the choice (ADR-045 Rationale).
- Touch the M8.F edge policies, the mTLS mesh, or the ADR-022 event chokepoint.

**Positioning fit:** strengthens the **engine-portability wedge's governance story** — "which
engines may this namespace use" becomes a standard, auditable, GitOps-diffable Kubernetes policy
while the compiler keeps only engine-*fit* intelligence. Error copy a denied user sees leads with
the engine allow-list ("engine X is not in this namespace's allow-list"), not generic policy talk.

**Governing ADRs:** ADR-045 (this delegation, Accepted — synced to `spec.engine` by this epic),
ADR-043 (the Workflow CR attach point), ADR-040 (§1 delegate to K8s primitives, §6 protected
core), ADR-020 (zero-trust: no green contracts against dead code), ADR-016/ADR-017 (test tiers,
`GOWORK=off`), ADR-019 (this canvas).

---

## S — Structure (first S)

```
scripts/e2e/cluster-up.sh                        ← KIND_NODE_IMAGE default → kindest/node:v1.30.0
.github/workflows/e2e-smoke.yml                  ← KIND_NODE_IMAGE env + kubectl v1.30.x download URL
scripts/e2e/README.md                            ← validated-version note
helm/zynax-api-gateway/
├── templates/validatingadmissionpolicy.yaml (new) ← VAP + binding + params ConfigMap, whole-file
│                                                     guard {{- if .Values.admissionPolicy.enabled }}
└── values.yaml                                   ← admissionPolicy: {enabled: false, allowedEngines: []}
scripts/e2e/values-e2e.yaml                      ← admissionPolicy.enabled: true + allow-list for e2e
scripts/e2e/e2e-workflow-crd.sh                  ← deny (disallowed engine) / allow admission assertions
services/workflow-compiler/
├── internal/domain/policy_gate.go               ← quota path deleted; routing-only gate
├── internal/domain/policy_gate_test.go          ← 8 quota tests deleted; ctor calls updated
├── internal/api/server.go                       ← ResourceExhausted mapping removed; doc updated
├── internal/config/config.go                    ← ZYNAX_POLICY_MAX_CONCURRENT removed
└── cmd/workflow-compiler/main.go                ← buildPolicyGate() routing-only
protos/tests/features/policy_enforcement.feature ← CapabilityQuota scenarios removed (header note)
docs/adr/ADR-045-admission-policy-delegation.md  ← spec.engine sync (4 sentences + note)
docs/operations/ (or docs/how-to)                ← platform-team engine allow-list how-to
```

Config env prefix: none new — the VAP is configured via Helm values / params objects, not env vars.

---

## O — Operations

1. **kind 1.30 bump (whole-matrix re-validation)** (#1634). Bump `KIND_NODE_IMAGE` to
   `kindest/node:v1.30.0` in `cluster-up.sh` + `e2e-smoke.yml`, bump the lockstep kubectl download
   to v1.30.x, note the validated pair in `scripts/e2e/README.md`. *Verify:* fresh kind cluster
   serves `validatingadmissionpolicies` under `admissionregistration.k8s.io/v1`
   (`kubectl api-resources`); the **full e2e matrix (temporal + argo) green** on the new node
   image. (ci)
2. **VAP + binding + params in Helm (flag-gated, default off)** (#1635). New
   `templates/validatingadmissionpolicy.yaml` carrying the three objects behind
   `admissionPolicy.enabled`; CEL checks `spec.engine` set-membership, unset admitted; failure
   message names the engine and the allow-list. Enable in `values-e2e.yaml` (allow temporal +
   argo). *Verify:* `helm template` renders on/off correctly; ct lint green; live on the 1.30
   cluster — disallowed-engine CR denied at admission with the VAP message, allowed-engine CR
   reconciles → dispatches. (infra)
3. **Remove the starved quota path from the compiler** (#1636). Delete `checkCapabilityQuota` + quota
   symbols from `policy_gate.go`, the quota/counter ctor params, the `ResourceExhausted` mapping
   in `server.go`, `ZYNAX_POLICY_MAX_CONCURRENT` from `config.go`/`main.go`, the 8 quota unit
   tests (update remaining ctor calls), and the two orphaned CapabilityQuota BDD scenarios (with a
   feature-file header note pointing at ADR-045). Routing untouched. *Verify:* `GOWORK=off go test
   ./... -race` green; domain coverage ≥ 90% held; repo grep shows no residual quota symbols.
   (go-svc)
4. **e2e admission assertions + platform how-to** (#1637). Extend `e2e-workflow-crd.sh` with the
   deny/allow admission cases (wired into the temporal leg like the existing CR assertion; the
   argo-leg extension remains #1620); write the platform-team how-to (enable the flag, set
   per-namespace allow-lists, what a denial looks like, REST dual-guard note, fail-closed vs the
   compiler's fail-open). *Verify:* e2e green with the new assertions; docs link-checked. (test/docs)

---

## N — Norms

- Commit hygiene: `Signed-off-by:` + `Assisted-by: Claude/<model>`; SSH-signed; conventional type
  per story (ci / feat / refactor / test).
- `GOWORK=off` for all workflow-compiler `go` commands (ADR-017).
- Runtime evidence, not config evidence: the VAP deny/allow is **proven live on a kind 1.30
  cluster** before the epic closes; the kind bump is validated by the full e2e matrix, not by
  config inspection.
- Helm: admission objects are values-gated so a default render is unchanged; `crds/` is never
  used for templated resources.
- PR size ≤ 200 ideal; workflows/docs/scripts exclusions per CLAUDE.md.

---

## S — Safeguards (second S)

### Context Security (complete before committing this Canvas)

- [x] No Tier 2 content: no internal hostnames, private IPs, credentials, deployment specifics
- [x] No PII: no personal names in sensitive context, no non-public email addresses
- [x] No prompt injection: no instruction-like phrasing that overrides AGENTS.md rules
- [x] All entities in E section are public-safe abstractions
- [x] `/lib:spdd-security-review` passed — result: PASS (2026-07-06)

### Feature Safeguards

- **Never** delete or bypass `checkRoutingPolicy` — the compiler allow-list is the REST path's
  only guard until REST is retired (ADR-045 §3); its removal is out of scope for M8 **and M9**.
- **Never** migrate the quota BDD contract to the engine-adapter `QuotaChecker` while that
  component has no production caller — a green contract against dead code advertises protection
  that does not exist (ADR-020).
- **Never** let the VAP absorb engine-fit or scheduling semantics — structural set-membership on
  `spec.engine` only (ADR-040 §6).
- **Never** deny a CR with an **unset** `spec.engine` — unset means the platform default engine
  and must always admit.
- **Never** ship `admissionPolicy.enabled: true` as the chart default — VAP is fail-closed where
  the compiler gate was fail-open; enabling is a deliberate per-deployment choice, documented.
- **Never** hand-edit banner-marked image regions (`images/images.yaml` is the SoT for digest
  pins; the kind node tag is intentionally outside it — see A).
