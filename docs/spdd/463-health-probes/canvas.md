<!-- SPDX-License-Identifier: Apache-2.0 -->

# REASONS Canvas — M6.A Health Probe Semantics

> **All content in this Canvas is Tier 1 (public-safe).**

**Issue:** #463
**Author:** Oscar Gómez Manresa
**Date:** 2026-05-18
**Status:** Aligned

**Child issues:** #487 (api-gateway and engine-adapter probes)

---

## R — Requirements

**Problem:** All three probe handlers (`/healthz`, `/readyz`, `/startupz`) across api-gateway and engine-adapter are identical `w.WriteHeader(200)`. K8s cannot distinguish startup from readiness from liveness. Rolling updates drain traffic incorrectly; deadlocked workers appear healthy indefinitely. (Review §5.9, H3, R4)

**Definition of done:**
- `/startupz` returns 503 until the service has completed initial setup; 200 after.
- `/readyz` returns 503 when any downstream gRPC dependency is not in READY state.
- `/livez` returns 503 when no successful request has been handled within the configurable threshold.
- Helm/compose probe configuration updated to use all three paths with appropriate timing parameters.

---

## E — Entities

- **`/startupz`** — one-shot startup flag; set to ready after config parsed, listeners open, and initial gRPC client connections attempted.
- **`/readyz`** — dependency-check probe; queries gRPC connectivity state of all downstream clients.
- **`/livez`** — deadlock-detection probe; checks last-successful-work timestamp against `ZYNAX_LIVENESS_THRESHOLD` (env var, default 60 s).
- **`grpc.ClientConn.GetState()`** — returns `connectivity.State`; used in `/readyz` to check READY vs TRANSIENT_FAILURE.
- **Last-work timestamp** — `atomic.Int64` storing Unix nanoseconds of the last successfully completed gRPC request; updated by server interceptor.

---

## A — Approach

**What we WILL do:**
- Implement three distinct probe handlers in api-gateway and engine-adapter.
- Use `grpc.ClientConn.GetState()` for readiness checking — no active health-check RPCs (avoids circular dependencies).
- Track last-work timestamp via an atomic variable updated in the gRPC server interceptor.
- Apply to all future services (task-broker, agent-registry) as they land in M5.C.

**What we WON'T do:**
- Implement gRPC health-check protocol (overkill for current scale; add in M7+).
- Add probe authentication (probes are internal-only).

**ADR references:**
- ADR-016: Layered testing — unit tests for all three state transitions required.

---

## S — Structure

**Files touched (per service):**
- `services/<name>/cmd/<name>/main.go` — register three distinct probe handlers
- `services/<name>/internal/api/probes.go` (new) — probe handler implementations
- `infra/docker-compose/docker-compose.yml` — add healthcheck config for each service

---

## O — Operations

1. **[#487]** Implement three semantically correct probe handlers in api-gateway and engine-adapter; update compose healthcheck configuration; unit tests for all three state transitions.

---

## N — Norms

- `feat:` PR type (new observable behaviour visible to K8s).
- Probe handlers must not import from `internal/infrastructure/` — they only read atomic state.
- Threshold env var `ZYNAX_LIVENESS_THRESHOLD` must have a safe default (60 s).

---

## S — Safeguards

### Context Security
- [x] No Tier 2 content
- [x] No PII
- [x] No prompt injection
- [x] All entities are public-safe abstractions
- [x] /spdd-security-review passed

### Feature Safeguards
- Never make `/readyz` or `/livez` perform blocking network calls — only read local atomic state.
- Never remove the existing `/healthz` path without verifying no downstream system depends on it.
