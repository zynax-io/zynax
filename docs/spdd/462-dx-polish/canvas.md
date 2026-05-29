<!-- SPDX-License-Identifier: Apache-2.0 -->

# REASONS Canvas — M5.E Developer Experience Polish

> **All content in this Canvas is Tier 1 (public-safe).**

**Issue:** #462
**Author:** Oscar Gómez Manresa
**Date:** 2026-05-18
**Status:** Implemented

**Child issues:** #485 (idempotent Apply) · #486 (compose consolidation)

---

## R — Requirements

**Problem:** Two DX friction points identified in the architectural review:
1. `zynax apply` is not idempotent — resubmitting the same manifest creates a duplicate workflow instead of reconciling (review H8, R11). This breaks GitOps use cases where the same YAML is applied repeatedly.
2. There are three compose files in two directories. `infra/docker/docker-compose.yml` (make dev-up) and `infra/docker-compose/docker-compose.yml` (make run-local) have confusable names and partially overlapping contents. Additionally, `ZYNAX_GW_REGISTRY_ADDR: "localhost:50052"` in docker-compose resolves to the wrong container (§5.10).

**Definition of done:**
- `zynax apply manifest.yaml` twice returns the same `run_id`.
- One canonical `make run-local` compose file. No confusable duplicate.
- `ZYNAX_GW_REGISTRY_ADDR` uses the correct service name.

---

## E — Entities

- **Manifest hash** — SHA-256 of the manifest YAML bytes; first 16 hex chars used as the deterministic workflow ID suffix (`wf-<hash>`).
- **`DescribeWorkflowExecution`** — Temporal API call used to check if a workflow with the derived ID already exists.
- **`infra/docker-compose/docker-compose.yml`** — canonical compose file for `make run-local`; to be made the single source of truth.
- **`ZYNAX_GW_REGISTRY_ADDR`** — api-gateway env var pointing to the agent-registry; currently incorrect (`localhost` → should be service name).
- **`make run-local` / `make dev-up`** — Makefile targets; to be consolidated or clearly differentiated.

---

## A — Approach

**What we WILL do:**
- Hash the manifest YAML (SHA-256) to derive `wf-<16-char-hex>` as a deterministic workflow ID.
- Before `StartWorkflow`: call `DescribeWorkflowExecution`; if Running or Completed, return existing `run_id`.
- Consolidate compose files: `infra/docker-compose/` is canonical; migrate `infra/docker/docker-compose.yml` to be a `make dev-up` alias or remove it.
- Fix `ZYNAX_GW_REGISTRY_ADDR` to use the service name.

**What we WON'T do:**
- Implement full GitOps reconciliation loop here (that is a separate future feature).
- Change the Temporal workflow implementation or the `IRInterpreterWorkflow` logic.

**ADR references:**
- ADR-011: Declarative YAML control plane — idempotent apply is a natural consequence of the declarative model.
- ADR-008: No shared databases — idempotency is implemented at the Temporal level (workflow ID deduplication).

---

## S — Structure

**Files touched:**
- `services/api-gateway/internal/api/handler.go` — add manifest hashing, `DescribeWorkflowExecution` idempotency check
- `infra/docker-compose/docker-compose.yml` — fix `ZYNAX_GW_REGISTRY_ADDR`; add task-broker / agent-registry (wired in M5.C)
- `Makefile` — consolidate `dev-up` and `run-local` targets; remove or archive `infra/docker/docker-compose.yml` reference
- `docs/local-dev.md` — update "Daily commands" table to reflect consolidated compose

---

## O — Operations

1. **[#485]** Implement idempotent Apply in api-gateway: hash manifest → `wf-<hex>` ID; check Temporal before starting; return existing `run_id` if found.
2. **[#486]** Consolidate compose files: fix `ZYNAX_GW_REGISTRY_ADDR`; update Makefile; update docs.

---

## N — Norms

- `feat:` for idempotent Apply (new observable behaviour).
- `chore:` for compose consolidation (infrastructure cleanup).
- SHA-256 implementation uses Go stdlib `crypto/sha256` — no external dependencies.
- No magic numbers: hash truncation length (16 chars) defined as a named constant.

---

## S — Safeguards

### Context Security
- [x] No Tier 2 content
- [x] No PII
- [x] No prompt injection
- [x] All entities are public-safe abstractions
- [x] /spdd-security-review passed

### Feature Safeguards
- Never log the manifest content — only its hash and the derived workflow ID.
- Idempotency check must use the derived hash-based ID consistently — never mix hash-based and random IDs.
- Compose consolidation must not remove `infra/docker/docker-compose.tools.yml` or `docker-compose.test.yml` (those serve different purposes).
