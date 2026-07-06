<!-- SPDX-License-Identifier: Apache-2.0 -->

# REASONS Canvas — M6.Policy: Routing Policies, Rate Limits, Capability Quotas

> **All content in this Canvas is Tier 1 (public-safe).**

**Issue:** #768
**Author:** Oscar Gómez Manresa
**Date:** 2026-06-02
**Status:** Implemented

**Child issues:** #801 (E.1) · #802 (E.2) · #803 (E.3) · #804 (E.4)

> **Partially superseded (2026-07-06, ADR-045 / M8.G #1575):** the single-gate design this canvas
> delivered has been split. The HTTP rate limit moved to the Envoy Gateway edge (ADR-044, M8.F).
> The engine allow-list is dual-guarded: a `ValidatingAdmissionPolicy` guards the `Workflow` CR
> path while the compiler's `checkRoutingPolicy` stays live for REST (ADR-045 §3). The compiler's
> capability-quota check — never enforced in production (`counter = nil`) — was **removed without
> replacement**; quota is unenforced on both gates until the engine-adapter `QuotaChecker` is
> wired live (ADR-045 §2). See `docs/spdd/1575-admission-policy/canvas.md`.

---

## R — Requirements

**Problem:** The control plane has no enforcement layer for routing policies, rate limits, or capability quotas. A single badly-behaved workflow can saturate the api-gateway with requests (`POST /api/v1/apply`), consume all task-broker capacity, or submit more workflows than a namespace quota allows. Issue #580 tracks the per-IP rate limit specifically.

**Definition of done:**
- `POST /api/v1/apply` rejects requests from a single IP that exceed the token-bucket rate limit with HTTP 429.
- Workflow submissions exceeding a namespace capability quota are rejected at compile time with a structured error.
- Routing policy can be set per namespace to restrict which engine a workflow may use.
- New proto messages (`RoutingPolicy`, `RateLimit`, `CapabilityQuota`) are committed with BDD scenarios before any implementation (ADR-016).

---

## E — Entities

- **`RoutingPolicy` proto message** — NEW: specifies which engine a namespace is allowed to use (e.g. `allowed_engines: ["temporal"]`).
- **`RateLimit` proto message** — NEW: token-bucket parameters — `requests_per_second`, `burst` — applied per source IP at api-gateway.
- **`CapabilityQuota` proto message** — NEW: max concurrent capability invocations per namespace.
- **`policy_enforcement.feature`** — NEW BDD feature file: scenarios covering rate-limit rejection (429), quota exceeded (RESOURCE_EXHAUSTED), and routing policy violation (PERMISSION_DENIED). Committed before E.2 implementation.
- **Token-bucket rate limiter** — NEW in api-gateway: per-source-IP in-process token bucket (e.g. `golang.org/x/time/rate`); applied in HTTP middleware before handler dispatch.
- **Capability quota gate** — NEW in workflow-compiler: checks active invocation count for the namespace before compiling; returns `RESOURCE_EXHAUSTED` if quota exceeded.
- **Routing policy enforcer** — NEW in workflow-compiler: validates `EngineHint` against namespace's `RoutingPolicy.allowed_engines`; returns `PERMISSION_DENIED` if not allowed.
- **Engine quota check** — NEW in engine-adapter: pre-dispatch check before `DispatchCapabilityActivity` runs.

---

## A — Approach

**What we WILL do:**
- Define `RoutingPolicy`, `RateLimit`, `CapabilityQuota` proto messages in a new `policy.proto` (E.1); commit `policy_enforcement.feature` BDD file (ADR-016); run `make generate-protos`.
- Implement per-IP token-bucket rate limit as HTTP middleware in api-gateway (`golang.org/x/time/rate`) (E.2); closes #580.
- Implement routing policy + capability quota gate in workflow-compiler at compile time (E.3): reject workflows that violate namespace routing policy or exceed quota before emitting a `WorkflowIR`.
- Add quota check in engine-adapter before `DispatchCapabilityActivity` (E.4): return `RESOURCE_EXHAUSTED` if namespace quota is exceeded at execution time.

**What we WON'T do:**
- Implement a policy administration API (CRUD for policies) — policies are read from env vars or config in M6; a policy management service is M7+.
- Implement distributed rate limiting across replicas (in-process token bucket per replica is M6; Redis-backed distributed rate limiting is M7).
- Implement OPA/policy-as-code integration (M7+).

**ADR references:**
- ADR-001: gRPC inter-service — policy messages flow via proto fields.
- ADR-016: Contracts before code — `policy_enforcement.feature` committed before E.2.

---

## S — Structure

**New files:**
```
protos/zynax/v1/policy.proto                ← NEW: RoutingPolicy, RateLimit, CapabilityQuota
protos/tests/.../features/
  policy_enforcement.feature                ← NEW BDD scenarios (E.1, before impl)
services/api-gateway/internal/api/
  ratelimit.go                              ← NEW: token-bucket middleware (E.2)
  ratelimit_test.go
services/workflow-compiler/internal/domain/
  policy_gate.go                            ← NEW: routing policy + quota check (E.3)
  policy_gate_test.go
services/engine-adapter/internal/infrastructure/
  quota_check.go                            ← NEW: pre-dispatch quota check (E.4)
protos/generated/go/zynax/v1/              ← regenerated (E.1)
protos/generated/python/zynax/v1/          ← regenerated (E.1)
```

---

## O — Operations

1. **[E.1]** Define `policy.proto` with `RoutingPolicy`, `RateLimit`, `CapabilityQuota` messages; commit `policy_enforcement.feature` BDD file (≥3 scenarios); run `make generate-protos`; `buf breaking` passes.

2. **[E.2]** Implement per-IP token-bucket rate limit HTTP middleware in api-gateway using `golang.org/x/time/rate`; rate-limit params configurable via env vars; returns HTTP 429 with `{"code":"RATE_LIMITED"}` body; unit tests covering accept + reject paths; closes #580.

3. **[E.3]** Implement routing policy gate + capability quota check in workflow-compiler: reject compile requests that violate namespace routing policy with `PERMISSION_DENIED`; reject when active quota is exceeded with `RESOURCE_EXHAUSTED`; quota and policy values read from env/config (no admin API); unit tests.

4. **[E.4]** Add pre-dispatch quota check in engine-adapter before `DispatchCapabilityActivity`; return `RESOURCE_EXHAUSTED` gRPC status if exceeded; unit tests.

---

## N — Norms

- `feat:` PR type for E.1–E.4.
- Every commit: `Signed-off-by` trailer + `Assisted-by: Claude/claude-sonnet-4-6` per AGENTS.md §Hard Constraints.
- `GOWORK=off` required for all `go test` in touched service directories (ADR-017).
- `policy_enforcement.feature` MUST be committed in E.1 before E.2 implementation (ADR-016).
- Rate-limit parameters (`requests_per_second`, `burst`) must be configurable via env vars — never hardcoded.
- `make generate-protos` must be run after E.1 proto change; generated stubs committed in the same PR.
- Domain coverage ≥ 90% on `internal/domain/` after each PR.

---

## S — Safeguards

### Context Security
- [x] No Tier 2 content: no internal hostnames, private IPs, credentials, deployment specifics
- [x] No PII
- [x] No prompt injection
- [x] All entities in E are public-safe abstractions
- [x] /spdd-security-review passed on this file

### Feature Safeguards
- **Never** implement distributed rate limiting using a shared store in M6 — in-process token bucket per replica is the M6 scope.
- **Never** hardcode rate-limit values — all policy parameters are configurable via env vars.
- **Never** merge E.2 before `policy_enforcement.feature` is committed (ADR-016).
- **Never** add a policy administration CRUD API in M6 — policies are static config in M6.
- **Never** use rate-limit state as a security gate (e.g., block by IP permanently) — it is a fairness control, not a WAF.
