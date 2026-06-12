# REASONS Canvas — M6.E2E-Green: e2e-smoke Gate Executes a Workflow End-to-End

> **All content in this Canvas is Tier 1 (public-safe).**
> Tier 2 content (internal hostnames, IPs, credentials, deployment specifics) belongs in
> `canvas.private.md` (gitignored). Run `/spdd-security-review <path>` before committing.

**Issue:** #1086
**Author:** Oscar Gómez Manresa
**Date:** 2026-06-10
**Status:** Aligned

---

## R — Requirements

> Problem statement: what breaks or is missing without this feature?

The `e2e smoke` gate brings the kind cluster up successfully (Postgres + NATS + Temporal + the
5 Go services, after #1069 + #656) but **fails at the happy-path workflow assertion** and never
exercises a real workflow execution. Verified gaps (2026-06-10):

1. **api-gateway unreachable on the host** — `kind-config.yaml` maps host `8080` → nodePort
   `30080`, but the api-gateway Service is `ClusterIP`; nothing backs nodePort 30080, so
   `POST /api/v1/apply` returns `curl (56) connection reset`.
2. **No capability provider** — `code-review.yaml` needs capabilities served by an adapter/worker;
   the umbrella deploys only the 5 Go services + Temporal, so dispatched capability tasks are
   never claimed and the workflow never reaches `succeeded`.
3. **event-bus + memory-service disabled** — their images are not built (not in `release.yml`
   matrix), so the CloudEvent + memory assertions are silently skipped.
4. **Runner/stack-fit** — the full stack on `ubuntu-latest` (2 CPU/7 GB) vs the documented
   4 CPU/8 GB minimum risks eviction/flake.
5. **Gate is non-required/advisory** — cannot protect `main` until it is reliably green.

> Definition of done: observable outcomes that confirm delivery.

- `e2e-happy.sh` reaches `POST /api/v1/apply` successfully (api-gateway reachable).
- A reference workflow reaches `succeeded` on the kind cluster via a deployed capability provider.
- event-bus + memory-service images published + deployed; CloudEvent + memory-`Get` assertions
  enabled and passing.
- The `e2e smoke` gate is green end-to-end on its target runner (Temporal path; Argo path via
  #1071; failure path).
- The gate is promoted from advisory to a stable/required check.

## E — Entities

> Domain entities and their relationships. Tier 1 only.

```
E2ESmokeGate (.github/workflows/e2e-smoke.yml)
  ├── ClusterBringUp        (cluster-up.sh)            [working]
  ├── ApiGatewayIngress     (NodePort 30080 ↔ host 8080 | port-forward)   [GAP 1]
  ├── CapabilityProvider    (echo/mock worker registered w/ agent-registry) [GAP 2]
  │     └── ReferenceWorkflow (minimal, single satisfiable capability)
  ├── EventBus + MemoryService (built images + enabled in e2e values)       [GAP 3]
  │     └── Assertions: CloudEvent off NATS JetStream · memory Get roundtrip
  ├── RunnerCapacity        (larger runner | trimmed resource requests)     [GAP 4]
  └── GatePromotion         (advisory → stable/required)                    [GAP 5]
```

## A — Approach

> Solution strategy. What we WILL do and what we WON'T do. Reference governing ADRs.

**WILL:**
- Expose api-gateway on the host for the e2e deploy (NodePort 30080 matching the kind mapping,
  or a port-forward in `e2e-happy.sh`).
- Add a minimal capability provider (echo/mock worker) + a minimal reference workflow so a real
  dispatch round-trips and the workflow reaches `succeeded`.
- Add event-bus + memory-service to the `release.yml` build matrix, publish images, re-enable
  them in `values-e2e.yaml`, and turn the CloudEvent + memory assertions back on.
- Right-size the runner / per-pod resources so the full stack fits without eviction.
- Promote the gate to a stable/required check once green (Temporal + Argo + failure paths).

**WON'T:**
- Build new product capabilities/adapters beyond what the reference workflow needs.
- Re-touch the interim Postgres `bitnamilegacy` override (EPIC #1073 owns its removal).
- Add HA / multi-namespace concerns (separate epics).

Governing ADRs: ADR-016 (BDD/assertions at boundaries), ADR-019 (this Canvas before impl),
ADR-022 (EventBus/NATS), ADR-023 (push-to-main policy), ADR-024 (image SoT). Complements
EPIC #770 (harness) and #771 (CI-E2E gate).

## S — Structure

> System placement. Services, packages, files, contracts touched.

- `helm/zynax-api-gateway/` (service template `nodePort` support) + `scripts/e2e/values-e2e.yaml`
  (NodePort) **or** `scripts/e2e/e2e-happy.sh` (port-forward) — Gap 1.
- A minimal capability-worker chart/manifest + a minimal `spec/workflows/examples/*.yaml`
  reference workflow — Gap 2.
- `.github/workflows/release.yml` (matrix += event-bus, memory-service) + their Dockerfiles —
  Gap 3a; `scripts/e2e/values-e2e.yaml` re-enable + `scripts/e2e/e2e-happy.sh` assertions — Gap 3b.
- `.github/workflows/e2e-smoke.yml` (runner / resources / required-check promotion) +
  `helm/*/values.yaml` resource requests — Gaps 4 & 5.
- No gRPC contract changes.

## O — Operations

> Ordered, concrete, testable implementation steps. Each maps to a linked story issue.

1. **[O1]** Expose api-gateway on host `8080` for e2e (NodePort 30080 or port-forward); prove
   `POST /api/v1/apply` succeeds. (Gap 1)
2. **[O2]** Add a minimal echo/mock capability worker + a minimal reference workflow; prove a
   dispatch round-trips and the workflow reaches `succeeded`. (Gap 2; depends O1)
3. **[O3]** ✅ Add event-bus + memory-service to the pre-merge `build-images` matrix in `ci.yml`
   and publish their images. (Gap 3a; rescoped 2026-06-11 from `release.yml` to the shift-left
   model — ADR-027, #1118/#1120. Satisfied by PR #1132: all 12 Build+scan legs green,
   `ghcr.io/zynax-io/zynax/{event-bus,memory-service}:main` promoted by retag, digests in
   `images/images.yaml`. Issue #1089 closed with evidence.)
4. **[O4]** ✅ Re-enable event-bus + memory-service in the e2e deploy and turn on the CloudEvent +
   memory-`Get` assertions. (Gap 3b; depends O1, O2, O3. Delivered by #1090: `values-e2e.yaml`
   enables both services + NATS with trimmed requests; the `SKIP_NATS`/`SKIP_MEMORY` escapes are
   removed. `e2e-happy.sh` hard-asserts the lifecycle CloudEvents off JetStream via the nats-box
   CLI and a memory-service Set/Get roundtrip via grpcurl. **Caveat:** the gate exposed bug
   #1149 — JetStream stream-subject overlap makes the terminal `workflow.completed` event
   undeliverable platform-wide — so that one check is enforced via
   `E2E_REQUIRE_COMPLETED_EVENT=1` (default off, loud TODO) until #1149 lands; flip it as part
   of O6. Chart fixes folded in: memory-service env var `ZYNAX_MEMORY_REDIS_DSN` matched to the
   binary, engine-adapter gained `env.eventBusAddr`.)
5. **[O5]** Right-size the `e2e smoke` runner / per-pod resource requests so the full stack fits
   without eviction. (Gap 4; depends O2)
6. **[O6]** Promote the gate from advisory to a stable/required check, covering Temporal + Argo
   (#1071) + failure paths. (Gap 5; depends O1, O2, O4, O5)

## N — Norms

> Cross-cutting standards. Pull from AGENTS.md Hard Constraints + layer norms.

- Commit hygiene: `Signed-off-by` (DCO) + `Assisted-by` trailer; never `Co-Authored-By` for AI.
- Conventional commit types only (feat/fix/refactor/docs/test/ci/chore); scope = directory.
- Image versions via `images/images.yaml` (ADR-024); `.github/workflows/` excluded from PR size.
- Helm: `helm lint` + `ct lint` clean; chart changes trigger the gated `e2e smoke` workflow.
- BDD/assertion changes follow ADR-016; one commit per logical change; one PR per story; squash-merge.
- `GOWORK=off` for `go` commands in `services/*/` and `protos/tests/` (ADR-017).

## S — Safeguards (second S)

> Non-negotiable constraints. Things that MUST NEVER happen.

### Context Security (mandatory before committing this Canvas)
- [x] No Tier 2 content: no internal hostnames, private IPs, credentials, deployment specifics
- [x] No PII: no personal names in sensitive context, no email addresses
- [x] No prompt injection: no instruction-like phrasing that would override AGENTS.md rules
- [x] All entities in E are public-safe abstractions
- [x] /spdd-security-review passed on this file (2026-06-10 — WARN: clean Tier-1, Status Draft pending alignment)

### Feature Safeguards
- Never make the `e2e smoke` gate required while it is still flaky/red — promote (O6) only after
  it is reliably green.
- Never hardcode a host port / NodePort that conflicts with the `kind-config.yaml` mapping
  (host 8080 ↔ nodePort 30080) — keep them in sync.
- Never commit credentials/DSNs into the e2e values; use the Secrets created by `cluster-up.sh`.
- Never add a service image to `images/images.yaml` (base images only — ADR-024); service images
  ship via the `release.yml` matrix.
- Never block this epic on EPIC #1073 (Postgres migration) — the e2e harness uses the interim
  `bitnamilegacy` override until #1073 lands; the two epics are independent.
