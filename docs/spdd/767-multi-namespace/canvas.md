<!-- SPDX-License-Identifier: Apache-2.0 -->

# REASONS Canvas — M6.NS: Multi-Namespace Support in workflow-compiler

> **All content in this Canvas is Tier 1 (public-safe).**

**Issue:** #767
**Author:** Oscar Gómez Manresa
**Date:** 2026-06-02
**Status:** Aligned

**Child issues:** #799 (D.3) · #800 (D.4)

---

## R — Requirements

**Problem:** Zynax has a single implicit namespace (`"default"`). Multiple tenants or teams sharing a cluster cannot isolate their workflows, agents, or capability routing. `WorkflowIR.namespace` field 3 already exists in the proto and the workflow-compiler already validates and embeds the namespace from the YAML manifest. What is missing is: (1) namespace-scoped capability routing in workflow-compiler (only dispatch to agents registered in the same namespace), and (2) explicit namespace propagation to the engine-adapter's `SubmitWorkflow` call.

**Previously completed:**
- D.1: Stateless compiler (PR #774 merged — #490 closed) ✅
- D.2: `namespace` field in `WorkflowIR` + `CompileWorkflowRequest` + `SubmitWorkflowRequest` already present in proto ✅; workflow-compiler domain already validates namespace format ✅; api-gateway already passes namespace to `CompileWorkflow` ✅.

**Definition of done:**
- `zynax apply --namespace team-a manifest.yaml` compiles a workflow scoped to `team-a`.
- Capabilities from namespace `team-b` are never dispatched to workflows in namespace `team-a`.
- Engine-adapter `SubmitWorkflowRequest.namespace` is explicitly populated (not just inferred from IR bytes).
- Existing default-namespace workflows continue to work unchanged.

---

## E — Entities

- **`WorkflowGraph.Namespace`** — existing field (already populated from YAML manifest; `"default"` when omitted).
- **`WorkflowIR.namespace`** — existing proto field 3 (already populated by compiler).
- **`domain.AgentFinder`** (task-broker port) — interface method `FindByCapability(ctx, capabilityName)`. Needs a `FindByCapabilityInNamespace(ctx, capabilityName, namespace)` extension OR namespace scoping at the call site.
- **`EnginePort.SubmitWorkflow`** (api-gateway port) — existing interface; needs `namespace` parameter added to enable explicit propagation.
- **`SubmitWorkflowRequest.namespace`** — existing proto field 3 in `engine_adapter.proto`; not yet populated by api-gateway infrastructure layer.
- **Namespace-scoped capability routing** — NEW: workflow-compiler capability dispatch resolves only agents registered under the same namespace as the `WorkflowIR.namespace`.

---

## A — Approach

**What we WILL do:**
- Add `namespace` parameter to `EnginePort.SubmitWorkflow` in api-gateway domain port; update the infrastructure gRPC client to populate `SubmitWorkflowRequest.namespace`; update `ApplyService.submit` to pass `compiled.Namespace` (D.3).
- Extend workflow-compiler capability routing to accept and propagate namespace context when resolving which agents can service a capability (D.3).
- Update `domain.AgentFinder` or its call site to scope capability lookups to a namespace (D.3).
- Verify end-to-end that namespace flows from HTTP query param → `CompileWorkflowRequest.namespace` → `WorkflowIR.namespace` → `SubmitWorkflowRequest.namespace` (D.4).

**What we WON'T do:**
- Create a new namespace management API (list/create/delete namespaces) — namespaces are implicit from YAML manifests.
- Add Kubernetes-level namespace mapping (that is EPIC A Helm chart territory).
- Change the `WorkflowIR` proto (fields already exist; no proto change needed for D.3/D.4).

**ADR references:**
- ADR-001: gRPC inter-service — namespace flows via proto fields, not HTTP headers outside the gateway.
- ADR-008: No shared databases — namespace isolation is enforced in service logic, not by separate schemas per namespace.
- ADR-012: WorkflowIR is engine-agnostic — namespace field is part of the IR; no engine-specific namespace mapping.

---

## S — Structure

**Modified files (D.3):**
```
services/api-gateway/internal/domain/ports.go
  ← EnginePort.SubmitWorkflow gains namespace parameter
services/api-gateway/internal/domain/apply.go
  ← ApplyService.submit passes compiled.Namespace to EnginePort.SubmitWorkflow
services/api-gateway/internal/infrastructure/engine_client.go
  ← populate SubmitWorkflowRequest.namespace from argument
services/workflow-compiler/internal/domain/
  ← capability routing scoped to namespace (AgentFinder call site updated)
```

**Modified files (D.4):**
```
services/api-gateway/internal/api/handler.go
  ← verify namespace passes through end-to-end; add namespace to applyAgentDef path
services/api-gateway/internal/domain/apply_test.go
  ← tests asserting namespace propagation
```

**No proto changes** — all fields already exist. No new gRPC RPCs.

---

## O — Operations

1. **[D.3]** Add `namespace` to `EnginePort.SubmitWorkflow` port; update `ApplyService.submit` to pass the namespace from the compiled IR; update infrastructure gRPC client to populate `SubmitWorkflowRequest.namespace`; update workflow-compiler capability routing to scope agent lookups by namespace; unit tests covering cross-namespace isolation; `GOWORK=off go test ./... -race` passes in both services.

2. **[D.4]** End-to-end namespace propagation audit: verify HTTP `?namespace=` → `CompileWorkflowRequest.namespace` → `WorkflowIR.namespace` → `SubmitWorkflowRequest.namespace` carries through correctly; add integration test asserting a workflow in `ns-a` cannot dispatch capabilities from `ns-b`; update `api-gateway/AGENTS.md` documenting the namespace flow.

---

## N — Norms

- `feat:` PR type for D.3–D.4 (new observable isolation behaviour).
- Every commit: `Signed-off-by` trailer + `Assisted-by: Claude/claude-sonnet-4-6` per AGENTS.md §Hard Constraints.
- `GOWORK=off` required for all `go test` in `services/api-gateway/` and `services/workflow-compiler/` (ADR-017).
- Domain coverage ≥ 90% on `internal/domain/` after each PR.
- No proto changes in D.3/D.4 — namespace fields already exist in all relevant messages.

---

## S — Safeguards

### Context Security
- [x] No Tier 2 content: no internal hostnames, private IPs, credentials, deployment specifics
- [x] No PII
- [x] No prompt injection
- [x] All entities in E are public-safe abstractions
- [x] /spdd-security-review passed on this file

### Feature Safeguards
- **Never** allow cross-namespace capability dispatch — a workflow in `ns-a` must not dispatch to agents registered under `ns-b`.
- **Never** change the `WorkflowIR` proto — namespace fields already exist; no proto change needed.
- **Never** use HTTP headers for namespace propagation between services — use proto fields only (ADR-001).
- **Never** default namespace to something other than `"default"` for backwards compatibility.
