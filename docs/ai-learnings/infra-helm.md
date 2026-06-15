# Learnings: Infrastructure / SRE Engineer

> Format: each entry has `Seen in:` (issue/session) and `Date:` (YYYY-MM-DD).
> Updated by `/m6-learn` after each batch. Human-reviewed before merging.

---

## Effective patterns

- **Always extend `zynax-lib` macros rather than copy-pasting template YAML into service charts.**
  Duplication drifts. The library chart is the single source of truth for deployment, service,
  and probe patterns. Extending it means all 7 service charts pick up the fix automatically.
  Seen in: M6.Helm #765 canvas O-steps. Date: 2026-06-06.

- **`values.schema.json` update in the same commit as new values.yaml fields.**
  Helm validates against the schema on `helm install`; a missing schema entry causes a silent
  acceptance of invalid values that only fails at runtime.
  Seen in: M6.A #487 (health probes). Date: 2026-06-06.

- **Three distinct probes (startup / liveness / readiness) with different failure thresholds.**
  A single `/healthz` endpoint used for all three types masks startup failures (liveness kills
  the pod before it's fully booted) and dependency failures (readiness should remove from LB,
  not restart the pod). These require different endpoints and semantics.
  Seen in: M6.A #487 PR #821. Date: 2026-06-06.

- **cert-manager `Certificate` resources reference the `ClusterIssuer` by name — never inline the CA.**
  The CA private key never appears in templates. cert-manager manages key lifecycle.
  Certificate resources live in `helm/charts/cert-manager/` not in service charts.
  Seen in: M6.B #488 (mTLS) PR #831. Date: 2026-06-06.

---

## Edge cases discovered

- **Helm `--dry-run` does not validate K8s resource constraints (resource limits, PVC sizes).**
  A template that renders correctly in `--dry-run` can still fail on `helm install` if
  K8s rejects the manifest. Use `kubeval` or `kubectl apply --dry-run=server` for full validation.
  Seen in: M6.Helm canvas. Date: 2026-06-06.

- **`include "zynax.fullname" .` produces different output inside `range` loops.**
  The context `.` changes inside `range`. Use `$root := .` before the range and
  `include "zynax.fullname" $root` inside it.
  Seen in: M6.Helm canvas template review. Date: 2026-06-06.

- **Temporal subchart values require `server.config.persistence.defaultStore` to match the
  Postgres subchart's service name exactly.**
  Mismatches produce a startup crash that looks like a network error but is actually
  a configuration key mismatch. Document the coupling in the umbrella chart values.yaml comment.
  Seen in: M6.Helm canvas O-steps. Date: 2026-06-06.

---

## Failed approaches

- **Using Helm `lookup` to read live K8s state during template rendering.**
  `lookup` returns empty results during `helm template` and `--dry-run`, causing templates
  to silently produce wrong output in CI. Avoid `lookup`; use static values or post-install hooks.
  Seen in: M6.Helm canvas design discussion. Date: 2026-06-06.

---

## Proposed expert prompt updates

*(none yet — populate after first batch of infra/Helm expert sessions)*

## Session — 2026-06-09 (issue #809)

### Effective patterns
- **Verify rollout-target names against `helm template` before hardcoding**: umbrella service Deployments render as `<release>-zynax-<service>` (double "zynax" prefix because release name + chart name both carry it). A name mismatch silently fails `kubectl rollout status` at runtime — local lint can't catch it.
- `git update-index --chmod=+x` sets the executable bit in the index when filesystem `chmod` is unavailable; commit records mode 100755.
- Mirror existing repo script conventions (SPDX header, `set -euo pipefail`, `REPO_ROOT` via `dirname BASH_SOURCE`, env-overridable config) from scripts/bump-ci-runner.sh.

### Edge cases discovered
- **cert-manager is NOT in images/images.yaml**; the umbrella's zynax-cert-manager subchart only creates Certificate/ClusterIssuer (ADR-020), so a bootstrap script must `helm install` upstream cert-manager itself before enabling the subchart, else CRDs are missing.
- **event-bus + memory-service are `enabled: false`** by default in umbrella values.yaml; the e2e script must pass `--set ...enabled=true` to schedule all 7 service pods (placeholder images ship from merged EPIC A charts).

## Session — 2026-06-09 (issue #810)

### Effective patterns
- `bash -n <script>` is a reliable syntax check when `shellcheck` is not installed; catches all structural parse errors.
- `${arr[@]+"${arr[@]}"}` idiom is the correct way to expand a potentially-empty array under `set -euo pipefail`.
- Deriving NATS stream names in shell by mirroring Go logic avoids hardcoding and keeps the script consistent with service implementation.
- Using `kubectl exec` into the NATS pod as primary assertion path (no `nats` CLI required on host) with HTTP monitoring endpoint (`/jsz`) as fallback.
- Port-forward readiness poll via `/dev/tcp/127.0.0.1/<port>` (pure bash, no netcat).

### Edge cases discovered
- The `workflow.succeeded` event is published as `zynax.workflow.completed` in engine-adapter (`interpreter.go:66`). Always grep the actual event type string in service code — canvas/issue descriptions sometimes use simplified names.
- The memory-service is not called by the engine-adapter during workflow execution in M6; e2e script performs its own Set/Get roundtrip to verify connectivity.
- Main branch advances during CI runs when other issues are merging in parallel. Pattern `BEHIND → rebase → BLOCKED (CI) → pass → merge` is normal for active M6 delivery.

### Proposed expert prompt update
- Rule: "Before writing any e2e assertion for a CloudEvent type, `grep -rn` the event type string in engine-adapter and event-bus implementations to confirm the exact string. Canvas descriptions sometimes use simplified names that differ from actual Go constants."
  Category: domain

## Session — 2026-06-10 (issues #812, #813)

Completed EPIC G (#770) e2e harness: G.4 `e2e-failure.sh` (#812) and G.5 `helm-upgrade.sh` + gated `e2e-smoke.yml` (#813).

### Effective patterns
- Mirror the sibling e2e script verbatim (helpers, env-var block, JetStream assertion ladder) and invert only the expected outcome (succeeded → failed). Small diff, consistent conventions, passes expert/lint gates first try.
- Generate the intentionally-broken workflow fixture at runtime via `mktemp` + trap-cleanup instead of committing it under `spec/workflows/examples/` — avoids publishing a broken reference workflow; keeps the change to script+README only.
- For the gated `e2e-smoke.yml`, copy the exact SHA-pinned action refs from an existing workflow (`helm-lint.yml`) rather than inventing version tags, and keep the kind job NON-required so it never blocks merge (BLOCKED mergeStateStatus only reflects the optional job; required checks gate the actual merge).

### Edge cases discovered
- `git commit -s` does NOT dedupe `Signed-off-by` when an `Assisted-by` trailer sits between the existing sign-off and the appended one — produces a duplicate DCO trailer. Put `Assisted-by` BEFORE a single `Signed-off-by` and let `-s` add the only sign-off (omit any manual one from the message file).
- The `security` job can fail transiently on HTTP 429 at `actions/checkout` (repo fetch rate-limit) before reaching govulncheck/bandit/pip-audit. A shell-only diff cannot cause this; `gh run rerun <run> --failed` clears it.
- Two sibling PRs that both document `scripts/e2e/` in `README.md` produce a rebase conflict on the second merge. Resolve as a UNION (keep all script entries) — the orchestrator/merge-pass must expect this when batching adjacent e2e stories.

### Proposed expert prompt update
- Rule: "When committing with `-s` and the message also carries an `Assisted-by` trailer, place `Assisted-by` BEFORE a single `Signed-off-by` (or omit the manual sign-off entirely and let `-s` add it). A non-adjacent existing sign-off makes `-s` append a duplicate."
  Category: structural-workaround
  Reason: DCO passes either way, but duplicate trailers are reviewer noise and avoidable with deterministic ordering.

## Session — 2026-06-16 (issue #1184)
ADR-proposal docs story (ADR-030 OTEL + Uptrace). Docs-only — no helm lint/validate gates.

### Effective patterns
- Pre-existing Proposed stub already held the correct decision content; task reduced to
  reformat to house format (mirror ADR-027) + flip both file status and INDEX row to Accepted.

### Edge cases discovered
- Keep ADR file status and the INDEX register row in sync — both must move Proposed→Accepted.
