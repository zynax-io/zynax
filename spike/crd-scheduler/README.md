<!-- SPDX-License-Identifier: Apache-2.0 -->

# ADR-039 M7 spike — CRD-native Scheduler proof

This directory de-risks the [ADR-039](../../docs/adr/ADR-039-crd-native-scheduler.md)
one-way door **before** the M8 build commits to it. It is **throwaway** — delete the
whole `spike/` tree before M8. The **only durable artifact** is
[`config/crd/agents.zynax.io.yaml`](config/crd/agents.zynax.io.yaml), which M8 reuses
verbatim.

It is deliberately **not** in the repo `go.work` (it builds standalone like `cmd/zynax`);
run every Go command with `GOWORK=off`.

## What it proves

| # | Claim | How | Result |
|---|-------|-----|--------|
| 1 | `controller-runtime` + `client-go` compile under **`GOWORK=off`** (the heaviest unknown) | `GOWORK=off go build ./...` | ✅ **PASS** — `BUILD_EXIT=0`, no `go.work` conflict |
| 2 | The scoring pipeline is correct (capability match, hard constraints, readiness, expert isolation, weighted score, degradation) | 10 unit tests, pure, offline | ✅ **PASS** (`go test ./internal/scorer`) |
| 3 | Informer cache builds the `capIndex`-shaped index from `Agent` CR events on a real API server | KIND + reconcile loop | ✅ **PASS** — 3 agents indexed |
| 4 | Scoring picks the lower-latency **ready** agent | KIND: `reviewer-fast` (80ms) vs `reviewer-slow` (400ms) | ✅ **PASS** — `reviewer-fast` chosen |
| 5 | **Stale-liveness fix**: a not-ready agent is never selected (the bug the push registry had) | KIND: `reviewer-dead` `ready=false` despite 5ms latency | ✅ **PASS** — skipped |
| 6 | **Degradation**: Prometheus-down still returns a ready agent, never fails | KIND: `?fail=1` | ✅ **PASS** — `prometheus_consulted=false`, ready agent returned |
| 7 | **Resync-on-restart**: kill the scheduler, the index rebuilds from the API server with zero persisted state | KIND: kill + restart PoC | ✅ **PASS** — index 3→3, `SelectAgent` works immediately |

Runtime checks (3–7) ran on KIND `v1.30.0` via [`hack/verify.sh`](hack/verify.sh):
**7 passed, 0 failed.**

## Dependency / footprint delta (the cost M8 carries)

| Metric | Value |
|--------|-------|
| `sigs.k8s.io/controller-runtime` | `v0.24.1` |
| `k8s.io/client-go` | `v0.36.0` |
| Total modules (`go list -m all`) | 148 |
| Compiled PoC binary | ~43 MB |

This is the size/dependency weight the M8 scheduler image inherits — acceptable for a
control-plane component, and the precedent (`ArgoEngine`, observability) already pulls
large trees. **Fallback if `GOWORK=off` ever conflicts in the workspace:** pull the
scheduler out of `go.work` as a standalone module (it isn't needed there). The spike
confirms this fallback is *not* required at `controller-runtime v0.24.1`.

## Reproduce

```bash
cd spike/crd-scheduler
GOWORK=off go test ./...        # scorer unit tests (offline)
GOWORK=off go build ./...       # GOWORK=off x controller-runtime build check
bash hack/verify.sh             # full KIND runtime proof (needs kind, kubectl, jq, docker)
```

## Layout

```
config/crd/agents.zynax.io.yaml   DURABLE — the Agent CRD + OpenAPI v3 schema (reused in M8)
config/samples/agents.yaml        sample Agent CRs (annotations carry FAKE metrics)
internal/scorer/                  pure selection core — the logic M8's domain layer lifts
  index.go                        capIndex-shaped in-memory index (mirrors memory_repo)
  scorer.go                       ordered short-circuiting pipeline + degradation
  metrics.go                      MetricsSource port + in-process fake (M8 → PromQL client)
  scorer_test.go                  10 unit tests
cmd/poc/main.go                   controller-runtime manager + informer + /select HTTP surface
hack/verify.sh                    KIND runtime harness (throwaway)
```

## Carry-forward into the M8 build

- **Settled by this spike:** GOWORK=off build, resync statelessness, degradation, the
  scoring pipeline, and the CRD schema shape.
- **Still open for M8 (per ADR-039 Consequences):** the `status` reconciler that derives
  `ready`/`replicas` from the EndpointSlice (the spike sets `status.ready` by hand);
  leader election for the single-writer reconciler; the real PromQL `MetricsSource` + TTL
  cache; the `scheduler.proto` contract + task-broker cutover; removal of push-registration;
  and the first-run-without-Compose (k3d) resolution.
