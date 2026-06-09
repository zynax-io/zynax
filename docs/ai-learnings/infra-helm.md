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
