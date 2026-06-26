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

## Session — 2026-06-16 (issue #1190)
Story: O.7 — local Uptrace docker-compose observability stack (Uptrace + ClickHouse + Postgres + OTel Collector), UI on 127.0.0.1:7020. PR #1259.

### Effective patterns
- `docker compose config --quiet` with a throwaway gitignored `.env` (deleted after): validates the full overlay including `${VAR:?}` required-var guards without standing up containers — fast local gate for substitution/YAML errors.
- Loopback-pinned port mappings (`127.0.0.1:HOST:CONTAINER`) verified via `config --format json` `host_ip`: reviewable proof the "OTLP not publicly exposed" safeguard is met.
- `:?` required-var guards on every credential env: enforces "no committed defaults" — `make obs-up` fails loud if `.env.observability` is missing.
- Mirrored existing compose conventions (plain pinned third-party tags like `postgres:16-alpine`, healthchecks, `depends_on: service_healthy`) instead of registering digests in images.yaml — the dev compose is not an images.yaml consumer, so the overlay stays out of the digest-alignment gate while still passing it.

### Edge cases discovered
- The shared OTEL lib is `libs/zynaxobs` (NOT `zynaxotel`); the OTLP endpoint env var is `ZYNAX_OTEL_EXPORTER_OTLP_ENDPOINT` (`libs/zynaxobs/providers.go`). Matching that exact name is what lets services point at the collector.
- gitleaks CI (`tools/gitleaks-ai-context.toml`) scans the FULL PR commit range with `--source .`; its `email-address` rule allowlist is path-scoped to AI-context files only. A literal `admin`-at-`example.com` style address in a non-AI infra `.env.example` WOULD be flagged. Use `you-at-example-dot-com`.
- Issue carried both `status: ready` and `status: in-progress` labels but no `feat/1190` remote branch — the deterministic empty-branch claim push is the true mutex, so labels were not a blocker.

### Failed approaches
- `gh pr view --json merged`/`closingIssuesReferences`: rejected by this gh version; use `mergedAt`/`state`/`mergeCommit`.

### Proposed expert prompt update
- Rule: Compose credential files — commit only a `.env.<name>.example` with non-email placeholders (e.g. `you-at-example-dot-com`, never a real `x`-at-`y.com` address), gitignore the real `.env.<name>`, and put `${VAR:?msg}` required-var guards on every credential. gitleaks scans the full PR range and flags literal emails outside the AI-context allowlist.
  Category: domain
- Rule: The shared OTEL package is `libs/zynaxobs`; the OTLP exporter endpoint env var is `ZYNAX_OTEL_EXPORTER_OTLP_ENDPOINT`. Any collector/OTLP wiring must match this name.
  Category: domain
  Reason: Permanent naming fact other observability stories (O.8 Helm chart, O.9 logs) will need.

## Session — 2026-06-18 (EPIC #1370 — bundled local LLM for the quickstart)

### Effective patterns
- Zero-cost local LLM in compose: add an `ollama` service **inside** the compose network (never expose it on the host LAN), and reuse host-pulled models via a **read-only** bind mount of the models dir into `/root/.ollama/models` — no multi-GB re-download. Mount only the `models` subdir RO so the container keeps a writable `/root/.ollama` for its startup keypair. Parameterise the host path (`OLLAMA_HOST_MODELS`).
- Reaching a host daemon bound to loopback from a container is undesirable (binding it to all interfaces exposes it to the LAN). Prefer bundling the dependency in-network over reconfiguring the host daemon.

## Session — 2026-06-19 (EPIC #1370 — M7 awesome-quickstart cluster)

### #1374 (chore: validate a compose overlay)
- Compose overlays are NOT covered by `make validate-spec` (which only runs AsyncAPI + capability/workflow/agent-def/policy + expert-mapping). Validate an overlay with `docker compose -f base.yml -f overlay.yml config` (exit 0 + rendered merge) and paste the merged service block as PR Evidence.
- Keep third-party runtime images (ollama/postgres/nats) as direct refs — only first-party/base images go in `images/images.yaml`.

### #1386 (fix: value inside a bind-mounted config file)
- For a value living inside a BIND-MOUNTED config file (not a compose `environment:` entry), `docker compose config` will NOT surface it — it shows only the mount. Validate the merge AND `grep` the actual value out of the mounted file as separate evidence.
- Do NOT add `${ENV:-default}` tokens inside a config file the consuming Go service reads literally (no shell interpolation) — document a one-line file edit instead.

### #1360 (chore: one-command demo Make target)
- For a one-command demo Make target, prefer `docker compose ... up -d --wait` (Compose-native health gating) over a hand-written poll loop.
- Keep the apply → capture-`run_id` → result sequence in ONE backslash-continued recipe line with a leading `export ...;` — Make runs each recipe line in a fresh shell, so a captured `run_id` is lost across lines unless joined.
- `make -n <target>` + `docker compose config` is host-safe validation for a target that would otherwise need a multi-GB live run.

### Edge cases discovered
- A fresh `make run-local` silently leaves git/ci/llm adapters `Exited(1)` for missing `GITHUB_TOKEN`/`OPENAI_API_KEY`; only the langgraph `echo` capability works out of the box (#1375). `zynax logs` streaming returns HTTP 500 `streaming not supported` against the compose api-gateway (#1373).

## Session — 2026-06-25 (M7.K keystone — kind demo lifecycle, #1492)

### #1492 (feat(infra): make kind-up / demo-on-kind + Platform-ready banner + model preflight)
- WRAP the existing `scripts/e2e/cluster-up.sh` rather than reinvent kind bring-up: it already blocks on `kubectl rollout status` for all 7 services + Temporal-namespace registration, so gating the "Platform ready" banner on its exit status (plus a defence-in-depth rollout re-check) makes AC "no premature go signal" fall out for free. (domain)
- An opt-in `KIND_LOAD_IMAGES` knob (default OFF) lets the laptop demo `kind load` local images while leaving the CI kind path (GHCR pull) byte-identical — both `e2e smoke (temporal)` and `e2e smoke (argo)` stay green, proving the change is non-invasive. (structural-workaround)
- Real idempotency = TWO runs on ONE cluster: run 2 hits `helm upgrade` -> revision 2 and reuses the persistent Postgres creds Secret — the exact spot where "passed run 1, failed run 2" bites. A single run does NOT exercise this. (domain)
- Workflow status from api-gateway is under `.status` as the WorkflowStatus proto enum (`WORKFLOW_STATUS_COMPLETED`/`_FAILED`), with lowercase aliases — NOT `.state`/`succeeded`. When polling a run in a demo/e2e script, match the SAME alias set as `scripts/e2e/e2e-happy.sh` (`succeeded|completed|*COMPLETED|*SUCCEEDED` for success; `*FAILED|*ERROR|*CANCELED|*TERMINATED|*TIMED_OUT` for failure). Polling `.state` alone silently hangs for the full timeout. (domain)

### Edge cases discovered
- Temporal frontend/history/matching/engine-adapter CrashLoopBackOff at ~70s on COLD kind bring-up is EXPECTED churn (engine-adapter crash-loops until `cluster-up.sh` registers the Temporal `default` namespace post-frontend-ready); the 600s rollout wait absorbs it — do not treat early restarts as failure. (domain)
- The kind umbrella deploys NO ollama/llm-adapter — it uses an in-cluster `echo` capability (echo-worker), so the kind hero path needs no model. This makes "no host-LAN model exposure" trivially true and the model preflight purely host-side/informational. (domain)
- Do NOT edit a bash demo script while a run is still executing it — bash buffers script reads and a live edit can corrupt the running process; let the run finish (or kill it after the server-side workflow shows COMPLETED), then re-run with the fixed script (which doubles as the idempotency re-run). (structural-workaround)

## Session — 2026-06-26 (M7.K closeout loop — #1488 kind gateway, #1500 Argo wedge)

### #1488 (fix(infra): reliable localhost:8080 on kind)
- "Flaky NodePort on multi-node" was a RED HERRING — verify the RENDERED Service first (`kubectl get svc -o yaml`): a live `nodePort` (31079) mismatching the pinned value (30080) pointed at a render/merge defect, not kube-proxy timing. (domain)
- Umbrella charts vendor `.tgz` subcharts under `charts/`; `helm template`/`install` render the COMMITTED tgz, which can DRIFT from `helm/<sub>/` source. Diff `tar -xzf charts/<sub>.tgz -O <sub>/templates/<f>.yaml` vs source to expose it. Fix bring-up to always `helm dependency build` (source = SoT) rather than committing re-timestamped binary blobs. After a kind boot, `git checkout -- helm/zynax-umbrella/charts/` to discard the rebuild churn before committing. (structural-workaround)
- Add the assertion that exercises the REAL user path: every prior e2e script port-forwards to 18080, so the raw `localhost:8080` NodePort the banner advertises was never tested — that gap is why the bug shipped. Put the guard in `cluster-up.sh` so it runs in CI e2e-smoke for free. `kind create cluster --name` overrides the `name:` in the kind config file, so `CLUSTER_NAME=…` isolation is safe. (domain)

### #1500 (feat(infra): Argo-on-kind wedge)
- Prove the engine-portability wedge on ONE cluster: boot `E2E_ENGINE=argo`, prove Argo via the `Workflow` CR `Succeeded` (CR name == gateway run_id), then `helm upgrade --reuse-values -f values-e2e.yaml --set zynax-engine-adapter.env.activeEngine=temporal` to flip the engine-adapter IN PLACE and re-run the byte-identical manifest (Temporal pods stay deployed under the argo profile). (domain)
- The CLI `--engine` flag is a deploy-time-OVERRIDDEN hint — the engine-adapter picks ONE `active_engine` from umbrella values at boot (ADR-015/039), so don't use the flag to prove portability; flip the deployed value. Prove the engine via engine-adapter logs (`selecting workflow engine active_engine=…`, Temporal's `IRInterpreterWorkflow`/`DispatchCapabilityActivity`). (domain)
- `gh pr edit --add-label` aborts on the projects-classic GraphQL deprecation and silently fails; use REST `gh api -X POST repos/<o>/<r>/issues/<n>/labels -f "labels[]=<label>"`. For an infra/docs-only PR the `changes` filter skips the docker-build + per-engine e2e matrix; `e2e smoke (temporal)` shows a fast guard pass and there is NO separate argo required check — get runtime evidence from a local boot, don't wait for an argo CI leg. (structural-workaround)
