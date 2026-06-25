<!-- SPDX-License-Identifier: Apache-2.0 -->
# scripts/bench — stack resource benchmark

`stack-resources.sh` measures a **running** Zynax stack's footprint and the hero
workflow's wall-clock, then appends one row to a markdown comparison table. It is
the data source for the ADR-041 before/after numbers (Docker Compose vs full kind
vs lean kind). It is **read-only** against the stack — it never brings anything up
or down. Bring a stack up first, then point the script at it.

## What it captures per row

| Column | kind | compose |
|--------|------|---------|
| Pods/containers | `kubectl get pods -n <ns>` count | `docker compose ps -q` count |
| CPU | `kubectl top pods` sum (installs metrics-server; falls back to summed Pod **requests** offline) | `docker stats` CPU% sum |
| Memory | `kubectl top pods` sum (or requests fallback) | `docker stats` MEM sum |
| PVC reserved | sum of `spec.resources.requests.storage` over all PVCs | n/a (named volumes are sparse) |
| Image+layer disk | `du -sk /var/lib/containerd` across every kind node | `docker compose images` + named-volume sizes |
| Demo (apply→done) | echo workflow timed via gateway port-forward, **twice** (cold/warm) | same, via `localhost:7080` |

The CPU/memory cell notes its source (`used (metrics-server)` vs `requests`) so a
reader never mistakes a fallback request-sum for measured usage.

## Usage

```bash
# Full (prod-mirroring) kind stack — the "before"
make kind-up PROFILE=full
scripts/bench/stack-resources.sh --runtime kind --profile full-kind
make kind-down

# Lean kind stack — the "after"
make kind-up PROFILE=lite
scripts/bench/stack-resources.sh --runtime kind --profile lean-kind
make kind-down

# Docker Compose baseline
make run-local
scripts/bench/stack-resources.sh --runtime compose --profile compose
make stop-local
```

All three rows accumulate in `docs/benchmarks/kind-lean-resources.md` (override
with `--out`). Measure each stack **at rest** (right after bring-up, before any
run) so the comparison is apples-to-apples; the script's own timing run is the
loaded number. Pass `--no-workload` to capture resources only.

## Options

`--runtime kind|compose` · `--profile <label>` (required) · `--namespace <ns>`
(kind, default `zynax`) · `--cluster <name>` (default `zynax-e2e`) · `--out <file>`
· `--no-workload` · `--workflow <path>` (default the echo `e2e-demo.yaml`).

## Notes

- `kubectl top` needs metrics-server; the script installs it into the kind cluster
  on first run (with `--kubelet-insecure-tls`, required for kind's self-signed
  kubelet certs) and waits for metrics to populate (~15 s).
- Requires `python3` (unit parsing) and, for the timing run, `jq` + `curl`.
- The echo workflow (`spec/workflows/examples/e2e-demo.yaml`) is used on purpose:
  it needs no model, so the demo time reflects **stack/orchestration** latency,
  not LLM inference — keeping the before/after about the runtime, not the model.
