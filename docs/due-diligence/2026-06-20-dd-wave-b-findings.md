<!-- SPDX-License-Identifier: Apache-2.0 -->

# Zynax Due-Diligence — Wave B (Derived Technical) Findings

> **Run output of issue #1403** — the second execution wave of the investment-grade
> due-diligence framework ([2026-06-18-zynax-due-diligence-framework.md](2026-06-18-zynax-due-diligence-framework.md)),
> consuming the Wave A ground truth ([2026-06-20-dd-wave-a-findings.md](2026-06-20-dd-wave-a-findings.md)).
> A **findings artifact**, not a verdict: the recommendation is produced only after Wave D
> synthesis (#1405) and the final report (#1406).

| Field | Value |
|-------|-------|
| Wave | **B — Derived technical** (consumes Wave A; framework §3.2) |
| Issue | #1403 — *DD execution: run Wave B (derived-technical agents)* |
| Date | 2026-06-20 |
| Repository HEAD audited | `main` @ `e3135a6` (the code state; the Wave A *doc* merged later as `8926b28`) |
| Agents run | 6 — §5.6, 5.14, 5.15, 5.16, 5.22, 5.26 |
| Evidence discipline | §0.4 — every claim carries `path:line`, a command+output, or `UNKNOWN`; roadmap/marketing = `CLAIMED` |

## Provenance & consumption note

Each agent ran **read-only** and returned the §3.4 YAML packet plus a §6.2 prose section, scoring
only its primary zone (anti-overlap matrix §3.1) and recording cross-zone observations as
cross-references. Per §3.2, Wave B consumes Wave A's Architecture (5.1), Engineering (5.5), Testing
(5.7), and DevOps (5.9) findings. **Caveat:** at run time the local checkout had not yet pulled the
merged Wave A doc, so several agents re-derived the upstream findings independently from code and
cited the owning Wave A agent — the cross-references are sound, but were verified bottom-up rather
than read from the Wave A packet. Contributor emails and absolute local paths were neutralised
(secret/PII gate); no other content was altered.

## Wave B scorecard

> Provisional, un-weighted. Final aggregation (confidence-weighting, contradiction resolution) is
> the orchestrator's, in Wave D. Note 5.14 Technical Debt uses an **inverted** scale (high = low debt).

| Agent | Dim | Score | Conf | Most severe red flag (evidence) | Strongest green flag (evidence) |
|-------|-----|:---:|:---:|----------------------------------|----------------------------------|
| 5.6 Performance | D6 | 6 | High | task-broker capability fan-out is unbounded + uncancellable (one goroutine/task, detached ctx) — `service.go:80-87,316-328` | both hot-path benches beat targets 5–300×; HPA+PDB+limits on every chart |
| 5.14 Technical Debt | D4 | 8↓ | High | single-engine reality — only Temporal interprets the IR; Argo is a non-interpreting stub (from 5.1) | best-in-class dep-debt tracking: inline CVE-cited pin floors + dated pip-audit suppression — `agents/sdk/pyproject.toml:26,35,86-90` |
| 5.15 Maintainability | D4 | 7 | High | `services/AGENTS.md:22,46` claims a "CI-enforced" import boundary, but **no import linter exists** (grep empty) — convention only | build-system-enforced modularity: 11 separate Go modules make cross-service `internal/` coupling mechanically impossible |
| 5.16 Scalability | D3/D6 | 5 | High | two acknowledged SPOFs — non-HA Postgres (`postgres/templates/statefulset.yaml`) + single-node JetStream (`nats.go:144`); umbrella default falls back to in-memory maps | genuinely stateless compute with zero-downtime rollout; C7 closed |
| 5.22 OpenSSF | D5/D7 | 6 | High | Scorecard badge renders "no data" (no `scorecard.yml`; API 404) **and** no top-level LICENSE file while README links it | Token-Permissions 21/21 least-privilege; all GitHub Actions SHA-pinned (Pinned-Dependencies) |
| 5.26 Innovation | D16/D10 | 6 | Med | the boldest moat (engine portability) is innovation at the interface but not embodied at execution; all 4 candidates reframe self-cited prior art | adapter-first no-SDK routing is genuinely embodied — 2-RPC `AgentService`, 5 cross-language adapters, zero SDK import |

Un-weighted mean ≈ **6.3 / 10**. Directional only.

## Aggregate drift test (Wave B contributions)

| Claim | Register | Result | Evidence (agent) |
|-------|:---:|:---:|------------------|
| Stateless workflow-compiler (no unbounded in-memory map) | C7 | **VERIFIED (fixed, stronger than claimed)** | Map removed entirely; only a stale proto comment remains — `workflow_compiler.proto:50-53` (5.14) |
| cel-go guard evaluator (fail-closed) replaced the bespoke fail-open one | C8 | **VERIFIED** | `cel-go` v0.28.1 driving `evalGuard`, fail-closed — `interpreter.go:203,220-259` (5.14) |
| benchmarks gated / benchstat live in CI | architecture-review | **CONTRADICTED** | `bench-gate` verb implemented + baseline committed, but no workflow invokes it; fail-open even if run (5.6) |
| "production-ready / horizontally scalable" | M6 / strategy | **PARTIAL** | Stateless compute scales (HPA/PDB); stateful substrate has non-HA SPOFs (5.16) |
| OpenSSF Scorecard badge reflects a real score | README §badge | **CONTRADICTED** | Badge present, no `scorecard.yml`, live API 404 → "no data" (5.22) |
| Apache-2.0 LICENSE file present | §1.9 | **CONTRADICTED** | No top-level LICENSE in repo or git history (5.22; cross-confirmed in Wave C) |
| Engine portability "without a rewrite" is novel/defensible | thesis | **PARTIAL** | Real at the IR/port boundary; only Temporal interprets the IR (5.26, consuming 5.1) |

## Cross-cutting themes (provisional, for the orchestrator)

- **The verified-strength / partial-enforcement split from Wave A deepens.** Performance hot paths,
  modularity, and dep-debt hygiene are strong; but the bench gate isn't wired, the import boundary
  isn't linted, and the stateful tier isn't HA — *capabilities exist, enforcement/proof lags.*
- **Scalability is the weakest derived dimension (5/10).** Non-HA Postgres + single-node JetStream
  + an umbrella default that silently reverts to in-memory state is the technical ceiling.
- **Two new supply-chain/governance contradictions surfaced** (no LICENSE file; decorative OpenSSF
  badge) that compound the Wave A security posture and feed CNCF/OpenSSF readiness.
- **The moat is execution discipline, not protectable IP** (5.26): the durable advantage is the
  embodied no-SDK routing + the disciplined build, not the portability narrative.

## Handoff

Wave B feeds the synthesis waves: the Performance/Scalability/Debt/OpenSSF/Innovation findings flow
to **Wave D (#1405)** — Risk (5.23), Investment (5.17), Business Strategy (5.18) + the Part 4
orchestrator — and into the **final report (#1406)**.

---

# Per-Agent Findings Packets

> Each section is the verbatim, read-only output of one Wave B agent: its §3.4 YAML handoff packet
> followed by its §6.2 prose section. Local absolute paths and contributor emails were neutralised
> (secret/PII gate); no other content was altered.

---
# Agent 5.6 — Performance Agent · Wave B (derived technical)

> Issue #1403 · HEAD `e3135a60e4abb20886d51f81d6448b22fe04cb64` · READ-ONLY audit.
> Every claim is grounded in `path:line` / command-output, or marked `UNKNOWN`.
> Roadmap/marketing = `CLAIMED`; code/CI/contract/executed-proof = `VERIFIED`.
> Consumes Wave A 5.1 Architecture + 5.7 Testing (cited, not re-scored — §3.1 anti-overlap).

---

## (a) §3.4 Handoff packet

```yaml
agent: "5.6 Performance"
wave: "B"
dimension_groups: ["D6", "D3", "D16"]   # D6 primary; contributes D3 (eng quality) + D16 (scalability)
overall_score: 6
overall_confidence: "High"

sub_scores:
  - dimension: "Hot-path identification & quality (compile / IR interpret / dispatch / event publish)"
    score: 7
    confidence: "High"
    justification: "All four hot paths are clean Go; IR interpreter caches CEL programs in a sync.Map (no per-eval recompile); compiler is fully stateless. Linear scans (findState O(n) per transition) are fine at current IR sizes but unindexed."
    evidence:
      - "services/engine-adapter/internal/domain/interpreter.go:67-100 — Run() drives the state machine; per-iteration findState() linear scan over all states"
      - "services/engine-adapter/internal/domain/interpreter.go:192-261 — celEnv built once via sync.Once; cel.Program cached per-expr in progCache sync.Map (avoids recompilation on replays)"
      - "services/engine-adapter/internal/domain/interpreter.go:334-341 — findState() is an O(n) linear scan executed on every transition (no state index/map)"
      - "services/workflow-compiler/internal/api/server.go:46,157-172 — CompileWorkflow returns IR in-response, stores nothing; GetCompiledWorkflow always NOT_FOUND (stateless)"
      - "services/event-bus/internal/infrastructure/nats.go:158-209 — Publish() is synchronous PublishMsg (blocks on JetStream ack), not PublishAsync"
  - dimension: "Benchmark coverage & meaningfulness (are baselines real?)"
    score: 7
    confidence: "High"
    justification: "Two representative benchmarks on the two hottest CPU paths, with a committed machine-independent baseline; both RUN clean here (E1) and land within ~5% of baseline, well under their stated targets. But only 2 services carry benches; no dispatch/event-publish/IO bench; no load test."
    evidence:
      - "services/engine-adapter/internal/domain/interpreter_bench_test.go:43-62 — BenchmarkIRInterpreter drives a 5-state/10-action IR end-to-end with stub ports (target <1ms)"
      - "services/workflow-compiler/internal/domain/manifest_bench_test.go:28-39 — BenchmarkParseManifest parses the realistic code-review.yaml (target <500µs)"
      - "cmd (E1): cd services/engine-adapter && GOWORK=off go test ./internal/domain/... -bench=BenchmarkIRInterpreter -benchmem -count=2 → 21183 & 21259 ns/op, 38320 B/op, 278 allocs/op (PASS)"
      - "cmd (E1): cd services/workflow-compiler && GOWORK=off go test ./internal/domain/... -bench=BenchmarkParseManifest -benchmem -count=2 → 165260 & 167631 ns/op, 89686 B/op, 1401 allocs/op (PASS)"
      - "tools/bench-baseline.txt:13-20 — committed baseline (IRInterpreter ~21-22µs, ParseManifest ~166-205µs); Makefile:11-12 BENCH_SERVICES=engine-adapter workflow-compiler (only 2)"
  - dimension: "Benchmark regression gate — IMPLEMENTED but NOT WIRED into CI (drift core)"
    score: 3
    confidence: "High"
    justification: "A tested bench-gate verb (benchstat parser) exists AND a baseline is committed, but NO workflow / tools-ci script invokes make bench / bench-gate / benchstat; and even when run the gate is fail-open by default. The 'benchstat gate live' / 'benchmarks gated' claims are CONTRADICTED."
    evidence:
      - "cmd/zynax-ci/benchgate/benchgate.go:22-23,88-103 — DefaultThresholdPct=20; Summary() fail-open by default ('baseline not yet stabilised'); only BENCH_GATE_ENFORCE=true blocks"
      - "cmd/zynax-ci/cmd/bench_gate.go:32-33,61-67 — 'Fail-open by default (EPIC R safeguard): a regression WARNS but exits 0'"
      - "cmd: grep -rln 'bench-gate|bench_gate|bench-regression|BENCH_GATE_ENFORCE|make bench|make fuzz' .github/ tools/ automation/ → only tools/README.md, tools/bench-baseline.txt (NO workflow invokes it)"
      - "cmd: grep -rln benchstat .github/ → (empty: no CI workflow runs benchstat)"
      - "tools/bench-baseline.txt:4-5 — header CLAIMS 'CI compares fresh runs against this file with benchstat'; Makefile:17 same claim — contradicted by the grep above"
      - "Wave A 5.7 Testing (docs/due-diligence/2026-06-20-dd-wave-a-findings.md:64-73,106-108) corroborates: fuzz+bench 'do NOT gate CI ... local-only'"
  - dimension: "Concurrency model — backpressure & cancellation (task-broker fan-out)"
    score: 4
    confidence: "High"
    justification: "Capability dispatch fan-out spawns ONE goroutine per task with NO pool/semaphore/limit, on a DETACHED context that can never be cancelled and carries no deadline. The agent gRPC stream Recv loop has no client-side timeout. Unbounded goroutine + memory growth and no caller-side cancellation under load."
    evidence:
      - "services/task-broker/internal/domain/service.go:80-87 — DispatchTask: s.bg.Add(1); go func(){ s.executeAsync(detach(ctx),...) } — one goroutine per dispatch, no concurrency ceiling"
      - "services/task-broker/internal/domain/service.go:316-328 — detachedCtx: Done() returns nil, Err() returns nil, Deadline() returns zero — the async goroutine can NEVER be cancelled and has NO deadline"
      - "services/task-broker/internal/infrastructure/agent_executor.go:47,59-79 — ExecuteCapability stream + Recv() loop runs on the detached ctx; only task.TimeoutSeconds (a field the agent must self-honor) bounds it — no broker-side deadline"
      - "cmd: grep -rniE 'maxConcurrent|worker.?pool|semaphore|errgroup|limiter|maxInFlight' services/task-broker --include=*.go (non-test) → (empty: no concurrency limit anywhere)"
      - "services/task-broker/internal/infrastructure/agent_executor.go:35 — grpc.NewClient dialed PER Execute call (per-dispatch connection, no pooling)"
  - dimension: "Concurrency model — engine-adapter (Temporal) path"
    score: 8
    confidence: "High"
    justification: "The Temporal interpreter path is well-disciplined: activity StartToCloseTimeout + exponential-backoff RetryPolicy + non-retryable classification; CEL program cache is replay-safe; per-namespace concurrency quota exists. Worker uses default (sane) SDK concurrency; not explicitly tuned."
    evidence:
      - "services/engine-adapter/internal/infrastructure/temporal_workflow.go:77-87 — ActivityOptions{StartToCloseTimeout, RetryPolicy{InitialInterval 1s, Backoff 2.0, Max 30s, MaximumAttempts, NonRetryableErrorTypes}}"
      - "services/engine-adapter/internal/infrastructure/temporal_workflow.go:105-110 — lifecycle-event activity bounded by 5s StartToCloseTimeout"
      - "services/engine-adapter/internal/infrastructure/quota_check.go:75-103 — per-namespace MaxConcurrent ceiling (but default unbounded + FAIL-OPEN on counter error)"
      - "services/engine-adapter/cmd/engine-adapter/main.go:248-256 — worker.New(...) with default worker.Options (no MaxConcurrentActivityExecutionSize tuning; SDK default ~1000)"
  - dimension: "Memory — bounded state (Part 1 §1.10 C7 unbounded compiler map)"
    score: 9
    confidence: "High"
    justification: "C7 is FIXED in code: the workflow-compiler Server struct holds no map; the IR is never stored; GetCompiledWorkflow always returns NOT_FOUND. The only residual is a STALE proto comment still describing the removed unbounded map (a doc-drift, not a memory leak)."
    evidence:
      - "services/workflow-compiler/internal/api/server.go:27-31 — Server struct: only generateID + policyGate; NO map / store field"
      - "services/workflow-compiler/internal/api/server.go:46,157-172 — IR 'not stored anywhere'; stateless NOT_FOUND"
      - "cmd (E1): bench memory stable across iterations — 38320 B/op interpreter, 89686 B/op compiler (per-op, GC-reclaimed, no growth)"
      - "STALE: protos/zynax/v1/workflow_compiler.proto:50-53 — comment still says 'unbounded in-memory map ... planned for M6' (cross-ref Wave A 5.1 docs/due-diligence/2026-06-20-dd-wave-a-findings.md:101,149-151)"
  - dimension: "IO / latency per workflow step (gRPC, Temporal, Postgres, NATS)"
    score: 5
    confidence: "Medium"
    justification: "api-gateway pools long-lived gRPC conns with per-call deadlines (good). But: NATS Publish re-runs ensureStream (an AddStream RTT) on EVERY publish (uncached); task-broker dials a new gRPC conn per dispatch; Postgres pools use pgx defaults (no MaxConns tuning). Each adds avoidable per-step latency; none load-measured."
    evidence:
      - "services/api-gateway/internal/infrastructure/clients.go:65-104 — 4 downstream conns dialed ONCE at startup, reused; callTimeout per unary RPC deadline (good)"
      - "services/event-bus/internal/infrastructure/nats.go:161-209 — Publish() calls ensureStream() (b.js.AddStream RTT) on every event; grep shows no sync.Once/cache/knownStreams → redundant NATS round-trip per publish"
      - "services/task-broker/internal/infrastructure/agent_executor.go:35 — grpc.NewClient per Execute (per-dispatch dial cost)"
      - "services/task-broker/internal/infrastructure/postgres/repository.go:33 — pgxpool.New(ctx, dsn) with default config (pgx default max 4 conns/pool); no MaxConns/MaxConnLifetime set"
      - "docs/adr/ADR-022-event-bus-architecture.md:53 — extra gRPC hop '~0.1–1 ms per publish on local network' (CLAIMED, localhost)"
  - dimension: "Scalability config (HPA / PDB / resources / statelessness)"
    score: 8
    confidence: "High"
    justification: "Every service chart ships HPA (CPU 70% + memory 80%, min 2 / max 10) + PDB + resource requests/limits. Compiler stateless; stateful services Postgres-backed. Production-credible. Caveat: memory-based HPA may lose to OOM-kill if task-broker goroutine fan-out spikes faster than HPA reacts."
    evidence:
      - "helm/zynax-*/templates/hpa.yaml + pdb.yaml — all 6 zynax service charts carry both (find listing)"
      - "helm/zynax-api-gateway/templates/hpa.yaml:16-28 — autoscaling/v2 HPA on cpu + memory Utilization"
      - "helm/zynax-task-broker/values-production.yaml:4-19 — replicaCount 2; autoscaling min2/max10, CPU 70%/mem 80%; resources requests cpu200m/mem128Mi, limits cpu1000m/mem512Mi"
      - "TENSION: task-broker 512Mi limit + unbounded goroutine fan-out (service.go:80-87) → OOM-kill risk outpaces memory-HPA averaging"

drift_test:
  - claim: "Benchmarks + gate now real / 'benchstat gate live in M7.R #493' / 'benchmarks gated' (docs/architecture/2026-06-18-architecture-review.md:32,37,51)"
    result: "CONTRADICTED"
    evidence:
      - "bench-gate verb EXISTS + tested (cmd/zynax-ci/benchgate/benchgate.go) but is fail-open by default (benchgate.go:88-103)"
      - "NO CI workflow invokes it: grep -rln 'bench-gate|benchstat|make bench' .github/ tools/ → only README + baseline file, zero workflow refs"
      - "corroborated by Wave A 5.7 Testing (docs/due-diligence/2026-06-20-dd-wave-a-findings.md:106-108): 'Fuzz + benchmarks exist and are functional but do NOT gate CI'"
  - claim: "Interpreter < 100µs (docs/architecture/2026-06-18-architecture-review.md:51,240)"
    result: "VERIFIED"
    evidence:
      - "cmd (E1): BenchmarkIRInterpreter → ~21.2µs/op on an 11th-gen i7 — comfortably under 100µs (5-state/10-action IR)"
      - "tools/bench-baseline.txt:13-15 — baseline ~21-22µs/op confirms"
  - claim: "Manifest compile < 50ms (docs/architecture/2026-06-18-architecture-review.md:51,240)"
    result: "VERIFIED"
    evidence:
      - "cmd (E1): BenchmarkParseManifest → ~166µs/op (0.166ms) on the realistic code-review.yaml — far under 50ms"
      - "tools/bench-baseline.txt:18-20 — baseline ~166-205µs/op confirms"
  - claim: "Unbounded in-memory IR store replaced (Part 1 §1.10 C7; architecture-review.md:32)"
    result: "VERIFIED"
    evidence:
      - "services/workflow-compiler/internal/api/server.go:27-31 (no map in Server struct), :157-172 (stateless NOT_FOUND)"
      - "stale-comment residual only: protos/zynax/v1/workflow_compiler.proto:50-53"

red_flags:
  - severity: "High"
    finding: "Task-broker capability-dispatch fan-out is unbounded and uncancellable: one goroutine per dispatched task with NO worker pool/semaphore/concurrency limit, on a DETACHED context whose Done()/Err() return nil and Deadline() is zero. The downstream agent gRPC stream has no broker-side deadline (only an advisory TimeoutSeconds field the agent must self-honor). A burst of dispatches or a set of hung agents → unbounded goroutine + memory growth with no caller-side cancellation; the 512Mi pod limit can OOM-kill before the memory-HPA averages up."
    evidence:
      - "services/task-broker/internal/domain/service.go:80-87,222-227 (unbounded go func per task)"
      - "services/task-broker/internal/domain/service.go:316-328 (detachedCtx: Done nil, Err nil, no deadline)"
      - "services/task-broker/internal/infrastructure/agent_executor.go:47,59-79 (stream Recv loop, no client deadline)"
      - "grep maxConcurrent|semaphore|errgroup|limiter in services/task-broker → empty"
  - severity: "Medium"
    finding: "Benchmark regression gate is implemented and tested but NEVER wired into CI, and is fail-open even when run. The committed bench-baseline.txt + tested benchgate create the APPEARANCE of a guard; a perf regression on either hot path lands on main undetected. The architecture review's 'benchstat gate live' / 'benchmarks gated' is delivery-vs-narrative drift (Part 1 §1.10 class)."
    evidence:
      - "grep -rln 'bench-gate|benchstat|make bench' .github/ tools/ → no workflow"
      - "cmd/zynax-ci/cmd/bench_gate.go:32-33 (fail-open default); docs/architecture/2026-06-18-architecture-review.md:32,51"
  - severity: "Medium"
    finding: "Zero load/stress testing anywhere in the repo (no k6/vegeta/locust/wrk/ghz). The 10x/100x scaling story is wholly inferential: no concurrent-workflow test, no SLOs, Postgres pools at pgx defaults (no MaxConns tuning), EventBus throughput unmeasured. Honestly admitted in the review but unaddressed."
    evidence:
      - "cmd: grep -rliE 'load.?test|vegeta|k6|locust|wrk |ghz |stress.?test' (go/sh/yml) → empty"
      - "services/task-broker/.../postgres/repository.go:33 (pgxpool.New default config)"
      - "docs/architecture/2026-06-18-architecture-review.md:242 ('No load tests ... Postgres connection pooling not benchmarked. EventBus throughput limits unknown.')"
  - severity: "Low"
    finding: "NATS event publish re-runs ensureStream (a JetStream AddStream round-trip) on EVERY Publish call with no caching — a redundant network RTT on the event-publish hot path (idempotent but not free under high event rates)."
    evidence:
      - "services/event-bus/internal/infrastructure/nats.go:161-168 (ensureStream per Publish); grep no sync.Once/cache/knownStreams"
  - severity: "Low"
    finding: "No pprof/expvar/runtime-profiling endpoints in any service; perf diagnosis under load would require code changes. Benchmarks cover only 2 of 7 services and only the CPU-bound paths (no dispatch, IO, or event-publish bench)."
    evidence:
      - "cmd: grep -rln 'net/http/pprof|runtime/pprof|expvar' services --include=*.go (non-test) → empty"
      - "Makefile:11-12 (BENCH_SERVICES = engine-adapter workflow-compiler only)"

green_flags:
  - strength: "Both committed hot-path benchmarks RUN clean here (E1) and beat their published targets by 5x-300x: IRInterpreter ~21µs (<100µs target), ParseManifest ~166µs (<50ms target). The headline perf FIGURES are real and reproducible, not aspirational."
    evidence: ["cmd E1 ×2 (interpreter 21.2µs, compiler 166µs)", "tools/bench-baseline.txt:13-20"]
  - strength: "C7 unbounded-memory risk genuinely eliminated: workflow-compiler is fully stateless (no store field, IR returned in-response), so per-instance memory is request-scoped and GC-reclaimed."
    evidence: ["services/workflow-compiler/internal/api/server.go:27-31,157-172"]
  - strength: "Engine-adapter Temporal path is performance-disciplined: bounded activity timeouts + exponential-backoff retry + non-retryable classification + a replay-safe sync.Map CEL program cache (compile once per unique expr)."
    evidence: ["services/engine-adapter/internal/infrastructure/temporal_workflow.go:77-110", "services/engine-adapter/internal/domain/interpreter.go:192-261"]
  - strength: "Production-credible horizontal-scale config: every service chart ships HPA (CPU+memory, min2/max10), PDB, and resource requests/limits; api-gateway pools long-lived gRPC conns with per-call deadlines."
    evidence: ["helm/zynax-*/templates/{hpa,pdb}.yaml", "helm/zynax-task-broker/values-production.yaml:4-19", "services/api-gateway/internal/infrastructure/clients.go:65-104"]

open_questions:
  - "What breaks first at 10x: most likely the task-broker unbounded goroutine fan-out (OOM) or Postgres default pool exhaustion (4 conns/pool) — neither is load-tested."
  - "Is the bench-gate intended to ever be wired + enforced, or is the committed baseline decorative? (benchgate.go fail-open comment implies 'after a stable 3-run baseline' — when?)"
  - "Will ensureStream be cached so the per-publish AddStream RTT is paid once per stream rather than per event?"
  - "Are Temporal worker concurrency knobs (MaxConcurrentActivityExecutionSize) ever tuned, or is the SDK default the permanent ceiling?"

unknowns:
  - "Real per-step IO latency (gRPC + Temporal + Postgres + NATS) under concurrency — no load test, no SLO, no profiling endpoint; only single-op micro-benchmarks + ADR-claimed localhost figures (~0.1-1ms hop). E1 not obtainable read-only/offline for a multi-service path."
  - "Live CI behavior of the (unwired) bench gate — confirmed by static evidence it is not invoked; could not observe a CI run offline."
  - "Whether the memory-based HPA actually protects task-broker against the goroutine-fan-out OOM in practice (timing-dependent; not reproduced)."

cross_references:
  - to_agent: "5.7 Testing"
    note: "Confirms 5.7's finding that fuzz/bench do not gate CI; I add that the bench-gate is fail-open by default AND that the architecture review claims it as 'live' (drift). Primary bench-gating zone is 5.7's."
    evidence: ["docs/due-diligence/2026-06-20-dd-wave-a-findings.md:64-73,106-108", "cmd/zynax-ci/cmd/bench_gate.go:32-33"]
  - to_agent: "5.1 Architecture"
    note: "Builds on 5.1's stale-comment finding (workflow_compiler.proto:50-53) — confirmed C7 fixed in code; and on 5.1's process-per-engine / IR-durability-at-scale open item. Architecture review perf scoring (6.0) is the doc I drift-test against."
    evidence: ["docs/due-diligence/2026-06-20-dd-wave-a-findings.md:101,149-151,188-190"]
  - to_agent: "5.16 Scalability"
    note: "Primary scalability-arch zone is 5.16's. Cross-zone observations for them: unbounded task-broker goroutine fan-out (service.go:80-87,316-328), Postgres pgxpool default config (no MaxConns), NATS per-publish ensureStream RTT, and the OOM-vs-HPA timing tension — all bear on horizontal-scale limits."
    evidence: ["services/task-broker/internal/domain/service.go:80-87,316-328", "services/task-broker/internal/infrastructure/postgres/repository.go:33"]
  - to_agent: "5.10 Documentation / 5.13 Governance"
    note: "tools/bench-baseline.txt:4-5 + Makefile:17 + architecture-review.md:32,51 assert a benchstat CI gate that does not exist — doc-vs-reality reconciliation debt."
    evidence: ["tools/bench-baseline.txt:4-5", "docs/architecture/2026-06-18-architecture-review.md:32,51"]

recommendations:
  - priority: "P0"
    action: "Bound the task-broker dispatch fan-out: replace the unbounded per-task goroutine with a worker pool / weighted semaphore (config'd ceiling), and give the agent gRPC stream a broker-side deadline (derive from task.TimeoutSeconds) instead of the never-cancelling detached context. Pool the agent gRPC connection per endpoint."
    rationale: "Today a dispatch burst or a few hung agents → unbounded goroutine+memory growth with no cancellation; the 512Mi pod can OOM before the memory-HPA reacts. This is the most likely first failure at scale."
  - priority: "P1"
    action: "Wire make bench → benchstat → zynax-ci bench-gate into a scheduled CI job (e.g. weekly-audit.yml) and flip BENCH_GATE_ENFORCE=true once the 3-run baseline is stable; until then, stop documenting the gate as 'live'."
    rationale: "Closes the regression-detection gap AND the delivery-vs-narrative drift on architecture-review.md:32,51 — the tooling already exists; only the wiring + an honest label are missing."
  - priority: "P1"
    action: "Add a minimal load test (ghz against api-gateway apply + a concurrent-workflow harness) and tune Postgres pgxpool MaxConns/lifetime; publish first SLOs."
    rationale: "The entire 10x/100x scaling story is currently inferential (review.md:242 admits it); a single load profile would convert assumptions into measured break-points and validate the HPA config."
  - priority: "P2"
    action: "Cache ensureStream results (sync.Map of known streams) so the JetStream AddStream RTT is paid once per stream, not per publish; add pprof endpoints (debug-gated) to the services."
    rationale: "Removes a redundant network RTT from the event-publish hot path and enables under-load profiling without code changes; both cheap."
  - priority: "P2"
    action: "Refresh the stale 'unbounded in-memory map / planned for M6' comment at workflow_compiler.proto:50-53 to match the shipped stateless implementation."
    rationale: "The C7 risk is fixed in code; the lingering comment re-asserts a since-removed limitation (cross-ref 5.1)."
```

---

## (b) §6.2 Prose section

## 5.6 Performance — Score: 6 (High)

**Mission recap:** Assess performance architecture, expected scalability, benchmark quality, and likely future bottlenecks across the compile, IR-interpretation, capability-dispatch, and event-publish hot paths. (Wave B — consumes 5.1 Architecture and 5.7 Testing.)

**Verdict:** Zynax has genuine, executable performance discipline on its CPU-bound core: two representative micro-benchmarks on the two hottest paths run clean here (interpreter ~21µs/op, manifest compile ~166µs/op — both beating their published targets) against a committed, machine-independent baseline, and the C7 unbounded-memory risk is genuinely eliminated (the compiler is fully stateless). The Temporal interpretation path is well-engineered (bounded activity timeouts, backoff retries, a replay-safe CEL program cache), and every Helm chart ships HPA + PDB + resource limits. But the score is held at an adequate-6 by three real gaps that bite under load: (1) the task-broker capability-dispatch fan-out is **unbounded and uncancellable** — one goroutine per task with no pool/semaphore on a detached context that can never be cancelled and carries no deadline; (2) the benchmark regression gate is **implemented and tested but never wired into CI** and is fail-open even when run — yet the architecture review markets it as "live"; and (3) there is **zero load testing** anywhere, so the entire 10x/100x story is inferential. This is a project that has done the micro-optimization homework but not the systems-under-concurrency homework.

**Sub-dimension scores:**

| Sub-dimension | Score | Confidence | Evidence |
|---|---|---|---|
| Hot-path identification & quality | 7 | High | `interpreter.go:67-100,192-261,334-341`; `server.go:46,157-172`; `nats.go:158-209` |
| Benchmark coverage & meaningfulness | 7 | High | bench tests + E1 runs (21µs / 166µs); `tools/bench-baseline.txt:13-20`; only 2 svcs |
| Bench regression gate (wired?) | 3 | High | `benchgate.go:88-103` fail-open; grep over `.github/`/`tools/` → no invocation |
| Concurrency — task-broker fan-out | 4 | High | `service.go:80-87,316-328`; `agent_executor.go:47,59-79`; no semaphore (grep empty) |
| Concurrency — engine-adapter | 8 | High | `temporal_workflow.go:77-110`; `quota_check.go:75-103`; `interpreter.go:192-261` |
| Memory / bounded state (C7) | 9 | High | `server.go:27-31,157-172`; stale `workflow_compiler.proto:50-53` |
| IO / latency per step | 5 | Medium | `clients.go:65-104` (good); `nats.go:161-168`, `agent_executor.go:35`, `repository.go:33` |
| Scalability config (HPA/PDB/resources) | 8 | High | `helm/zynax-*/templates/{hpa,pdb}.yaml`; `task-broker/values-production.yaml:4-19` |

**Drift test:**
- *"Benchmarks + gate now real / benchstat gate live in M7.R #493"* (`architecture-review.md:32,51`) → **CONTRADICTED.** The `zynax-ci bench-gate` verb exists and is tested (`cmd/zynax-ci/benchgate/`), and a baseline is committed, but no workflow or `tools/ci` script invokes `make bench`/`bench-gate`/`benchstat` (grep empty), and the gate is fail-open by default (`bench_gate.go:32-33`). Corroborated by Wave A 5.7 (`docs/due-diligence/2026-06-20-dd-wave-a-findings.md:106-108`).
- *"Interpreter < 100µs"* (`architecture-review.md:51,240`) → **VERIFIED** by E1 (~21µs/op).
- *"Manifest compile < 50ms"* (`architecture-review.md:51,240`) → **VERIFIED** by E1 (~0.166ms/op).
- *"Unbounded in-memory IR store replaced"* (C7) → **VERIFIED** in code (`server.go:27-31,157-172`); only a stale proto comment remains.

**Red flags (severity-ordered):**
1. **High** — Task-broker dispatch fan-out is unbounded (one goroutine per task, no semaphore) on a detached, never-cancellable, deadline-less context; the agent stream has no broker-side timeout. OOM/goroutine-leak risk under load with no caller cancellation (`service.go:80-87,316-328`; `agent_executor.go:47,59-79`).
2. **Medium** — Bench regression gate is built and tested but never wired into CI and fail-open; the review claims it is "live" — delivery-vs-narrative drift (`bench_gate.go:32-33`; `architecture-review.md:32,51`).
3. **Medium** — No load/stress testing anywhere; Postgres pools at pgx defaults; EventBus throughput unmeasured; scaleout inferential (`grep` empty; `architecture-review.md:242`).
4. **Low** — NATS `ensureStream` runs an `AddStream` RTT on every publish, uncached (`nats.go:161-168`).
5. **Low** — No pprof/profiling endpoints; benches cover only 2 of 7 services, CPU paths only (`grep` empty; `Makefile:11-12`).

**Green flags:**
- Both hot-path benchmarks reproduce here (E1) and beat published targets by 5x-300x — the perf figures are real, not aspirational.
- C7 unbounded-memory genuinely eliminated — compiler fully stateless (`server.go:27-31`).
- Temporal path disciplined: bounded timeouts + backoff retry + replay-safe CEL cache (`temporal_workflow.go:77-110`; `interpreter.go:192-261`).
- Production-credible HPA/PDB/resource config on every chart; api-gateway pools long-lived gRPC conns with per-call deadlines (`helm/zynax-*`; `clients.go:65-104`).

**Open questions / unknowns:** What breaks first at 10x (likely task-broker goroutine OOM or Postgres 4-conn-pool exhaustion); whether the bench gate is ever intended to enforce; real multi-service per-step IO latency under concurrency (no load test, no profiling, only localhost ADR estimates).

**Recommendations:** P0 — bound the task-broker fan-out (worker pool/semaphore) and give the agent stream a real deadline + pooled connection. P1 — wire `make bench → benchstat → bench-gate` into scheduled CI and stop documenting the gate as "live" until it enforces; add a minimal load test + tune pgxpool + publish SLOs. P2 — cache `ensureStream`, add pprof endpoints, refresh the stale proto comment.

**Cross-references:** 5.7 Testing (bench/fuzz do not gate — their primary zone; I add fail-open + the "live" drift); 5.1 Architecture (C7 stale comment, process-per-engine, IR durability); 5.16 Scalability (unbounded fan-out, default Postgres pool, NATS per-publish RTT, OOM-vs-HPA tension — their primary zone, recorded as cross-zone observations); 5.10/5.13 (bench-gate doc-vs-reality reconciliation debt).
# Agent 5.14 — Technical Debt · Wave B (derived technical) · Issue #1403

> Audit target: `the repository root` @ HEAD `e3135a60e4abb20886d51f81d6448b22fe04cb64` (branch `main`). READ-ONLY.
> Scoring is **INVERTED**: high score = LOW debt (per §5.14 rubric, framework line 1065).
> Every claim carries `path:line` or a command→output, or is marked `UNKNOWN`. Roadmap/marketing = `CLAIMED`.
> Consumes Wave A 5.1 (Architecture), 5.5 (Engineering), 5.7 (Testing) — cited, not re-scored (§3.1).

---

## (a) §3.4 Handoff packet

```yaml
agent: "5.14 Technical Debt"
wave: "B"
dimension_groups: ["D4", "D3", "D15"]   # D4 primary (tech debt); contributes D3 (arch) + D15 (risk)
overall_score: 8
overall_confidence: "High"

sub_scores:
  - dimension: "Debt-marker density (TODO/FIXME/HACK/XXX in source)"
    score: 9
    confidence: "High"
    justification: "Only 5 raw marker hits in all source; 3 are non-debt (regex pattern strings + mktemp XXXXXX); just 2 GENUINE markers, both EPIC-tagged Helm placeholders. Near-zero density across a 7-service Go + Python codebase."
    evidence:
      - "cmd: grep -rnE 'TODO|FIXME|HACK|XXX' (go/py/proto/yaml/yml/sh, excl generated) → 5 total"
      - "agents/examples/go-review-expert/src/go_review_expert/agent.py:39,41 (regex DETECTOR strings, not debt)"
      - "scripts/e2e/e2e-failure.sh:139 (mktemp ...XXXXXX.yaml — template, not debt)"
      - "helm/zynax-event-bus/values.yaml:10 — '# TODO(EPIC-I #772): replace placeholder ... once event-bus ships' (issue-tagged)"
      - "helm/zynax-memory-service/values.yaml:10 — '# TODO(EPIC-J #773): ... once memory-service ships' (issue-tagged)"
  - dimension: "Lint suppression hygiene (//nolint scope + justification)"
    score: 8
    confidence: "High"
    justification: "215 //nolint directives, ZERO bare (every one names a linter). 70 carry inline justifications (the higher-risk gosec G115/G304 cases); the 145 without are idiomatic self-evident patterns (defer Body.Close errcheck, wrapcheck passthrough). Modest, scoped — not blanket suppression."
    evidence:
      - "cmd: grep -rohE '//nolint:[a-z,]+' → gosec×105, wrapcheck×56, errcheck×21, cyclop/funlen×16, ... (215 total, all scoped)"
      - "cmd: grep '//nolint($|[^:])' minus '//nolint:' → empty (NO bare nolint anywhere)"
      - "cmd: 70 nolints w/ same-line '//' justification; 145 without"
      - "samples-without: argo_client.go 'defer resp.Body.Close() //nolint:errcheck' (idiomatic); service.go '//nolint:wrapcheck' on passthrough returns"
      - "Wave A 5.5: tools/golangci-lint.yml:17-33 (14 linters, default:none); 3× golangci-lint run → '0 issues' (E1) — strict gate actually passes"
  - dimension: "Python type-ignore / noqa suppression hygiene"
    score: 8
    confidence: "High"
    justification: "24 '# type: ignore' (all error-code-scoped: [override]/[import-untyped]/[call-arg]/[misc]) + 5 noqa; most cluster on the untyped SDK base class (the ADR-010 drift Wave A flagged), the rest on untyped 3rd-party libs (grpc.aio, langgraph). Specific, not blanket."
    evidence:
      - "cmd: grep 'type: ignore' agents (excl _pb2) → 24; grep 'noqa' → 5"
      - "agents/examples/echo/src/echo_agent/agent.py:24 — '# type: ignore[misc]  # SDK base is untyped (no py.typed)'"
      - "agents/adapters/langgraph/src/langgraph_adapter/__main__.py:11 — '# type: ignore[import-untyped]' (grpc.aio untyped)"
      - "cross-ref Wave A 5.5: agents/sdk/src/zynax_sdk/agent.py:59 concrete Agent base (ADR-010 says Protocol) — root cause of the SDK [misc] ignores"
  - dimension: "Architectural debt: deferred isolation/RBAC, multi-tenancy"
    score: 6
    confidence: "High"
    justification: "Namespace isolation IS enforced at the data layer; but multi-namespace policy + policy-admin API explicitly deferred to M7+, and auth is a SINGLE shared bearer token (no RBAC, no per-user/role authz, disabled when empty). Real, tracked, dated deferral — the largest architectural gap for a sale."
    evidence:
      - "services/workflow-compiler/internal/config/config.go:60-61 — 'Only a single namespace policy is supported in M6. Multi-namespace policies and a policy administration API are deferred to M7+.'"
      - "services/api-gateway/internal/api/auth.go:13-25 — requireBearer: single static key, constant-time compare, DISABLED when key=='' (no RBAC)"
      - "services/memory-service/internal/api/handler.go:20 + datacontext.go:121 — namespace/run isolation IS enforced per-RPC (data layer holds)"
      - "config.go:29 — 'A policy administration API is deferred to M7+.'"
  - dimension: "Architectural debt: single-engine reality (Argo stub)"
    score: 5
    confidence: "High"
    justification: "Inherits Wave A 5.1's HIGH finding: only Temporal interprets the IR; Argo path is a non-interpreting stub (asserts non-empty payload, exits 0). Capability-dispatch parity 'deliberately out of scope'. Honestly documented in-repo, but it is the headline portability moat that is functionally unbuilt — the single largest reality-vs-narrative debt."
    evidence:
      - "scripts/e2e/manifests/argo-ir-interpreter.yaml:13-14 — 'parity with the Temporal IRInterpreterWorkflow) is deliberately out of scope for the smoke gate.'"
      - "Wave A 5.1 red_flag(High): argo_engine.go:62-98 never calls IRInterpreter.Run; e2e-argo.sh:232-266 asserts only CR phase==Succeeded"
      - "Wave A 5.1 drift_test: 'Engine-agnostic — Temporal OR Argo without rewrite' → PARTIAL (CONTRADICTED at execution boundary)"
  - dimension: "Test/lint debt: skipped tests, coverage exemptions, fail-open"
    score: 7
    confidence: "High"
    justification: "Only 6 Go t.Skip total: 1 hard service-level skip (engine-adapter BDD, deferred M6/M7 pending Temporal dev-container) + 5 legit env guards; 0 Python skips; NO coverage exemptions in the gate config. Inherits 5.7's fuzz/bench-not-gated + per-changed-service-coverage gaps. One documented domain fail-OPEN (PolicyGate quota)."
    evidence:
      - "cmd: grep t.Skip services/cmd/protos → 6 hits"
      - "services/engine-adapter/tests/steps_test.go:17 — 't.Skip(... deferred to M6/M7 when a Temporal dev-container is wired ...)' (the only HARD skip)"
      - "cmd/zynax/{cmd/mcp_test.go:47,118, gitops/watcher_test.go:201,206, validate/context_test.go:290} — env guards (windows/root/fixture-absent)"
      - "tools/coverage-gates.env — grep exempt|exclude|skip → empty (no coverage exemptions)"
      - "Wave A 5.7 red_flags(Medium×4): fuzz/bench not in CI; coverage gate per-CHANGED-service; engine-adapter BDD skipped; 2 of 4 integration suites gated"
      - "Wave A 5.5: services/workflow-compiler/internal/domain/policy_gate.go:184 (quota counter error → fail-OPEN, documented)"
  - dimension: "Dependency debt: renovate config + CVE pins"
    score: 9
    confidence: "High"
    justification: "Renovate is best-in-class: grouped/scheduled, digest-pinned Docker+Actions, automerge only patch. CVE pins carry CVE id + rationale INLINE; the single pip-audit suppression is DATED with a re-evaluation date and reason. govulncheck+bandit+pip-audit gate CI; HIGH-sev dep block on PRs."
    evidence:
      - "renovate.json:45-160 — grouped (go-services/python-agents/docker-images/github-actions/grpc-go), pinDigests true, automerge only matchUpdateTypes:[patch]"
      - "agents/sdk/pyproject.toml:26 — 'urllib3>=2.7.0  # CVE-2026-44431, CVE-2026-44432'; :35 'idna>=3.15  # CVE-2026-45409' (CVE-cited pin floors)"
      - "agents/sdk/pyproject.toml:86-90 — pip-audit ignore PYSEC-2026-196 'Added: 2026-06-08  Re-evaluate: 2026-09-08' (dated, reasoned: CI-runner pip vuln, not an SDK dep)"
      - ".github/workflows/pr-checks.yml:69 'Block new HIGH-severity vulnerable dependencies'; ci.yml:689-730 govulncheck+bandit+pip-audit; weekly-audit.yml:190 rescan"
  - dimension: "Debt tracking discipline (issue-tagged, dated deferrals)"
    score: 9
    confidence: "High"
    justification: "Deferrals are tracked with specific GH issue numbers AND dates: #466 (M5→M6), #1103 (M6→M7, operator decision 2026-06-12 w/ 4 code-verified gaps) kept behind a strict xfail gate; 3 Proposed ADRs (031/033/034, dated 2026-06-15) tied to named M7 EPICs. This is the opposite of untracked debt."
    evidence:
      - "state/current-milestone.md:26 — '5 deferred issues (#235 #239 #376 #466 #656) moved to M6'"
      - "state/current-milestone.md:130 — 'O8 #1103 deferred to M7 — operator decision 2026-06-12, four code-verified platform gaps ... strict xfail in automation/tests/test_platform_readiness.py remains the honest gate'"
      - "docs/adr/INDEX.md:44,46,47 — ADR-031/033/034 status Proposed, dated 2026-06-15, each mapped to an M7 EPIC"
      - "automation/tests/test_platform_readiness.py:52 — 'the xfail marker is gone ... fixture SKIPS CLEANLY' (honest tripwire, not a fake pass)"

drift_test:
  - claim: "C7 — workflow-compiler refactored from unbounded in-memory IR map to stateless (fixed M6)"
    result: "PARTIAL"
    evidence:
      - "VERIFIED in code: services/workflow-compiler/internal/api/server.go:22-24,46 — 'The compiler is stateless ... it is not stored anywhere'; Server struct (server.go:27-31) holds only generateID + policyGate, NO map"
      - "VERIFIED: server.go:168-171 GetCompiledWorkflow UNCONDITIONALLY returns NOT_FOUND ('compiler is stateless — retain ir_payload') — the map was REMOVED, a stronger fix than a TTL cache"
      - "STALE-DEBT residue: protos/zynax/v1/workflow_compiler.proto:50-53 STILL documents 'an unbounded in-memory map with no TTL ... planned for M6 via the stateless-compiler refactor (issue #466)' — the contract comment contradicts the shipped stateless code (also flagged Wave A 5.1)"
  - claim: "C8 — bespoke fail-open guard evaluator replaced with cel-go (fixed M6)"
    result: "VERIFIED"
    evidence:
      - "services/engine-adapter/internal/domain/interpreter.go:17 imports 'github.com/google/cel-go/cel'; :203 cel.NewEnv; :236 env.Compile; :250 prog.Eval — real cel-go, not bespoke"
      - "services/engine-adapter/go.mod:7 — 'github.com/google/cel-go v0.28.1' (pinned current)"
      - "interpreter.go:216,220-259 — evalGuard is FAIL-CLOSED on every path (empty/compile/eval/non-bool all return false) — replaces the old fail-OPEN evaluator C8 describes"
  - claim: "Technical debt is large but untracked (the risk the diligence exists to catch)"
    result: "CONTRADICTED"
    evidence:
      - "Debt density is near-zero (5 raw markers, 2 genuine, both issue-tagged); deferrals carry GH issue numbers + dates (state/current-milestone.md:26,130); CVE suppressions are dated with re-evaluate dates (pyproject.toml:89)"
      - "The debt that exists (Argo stub, single-bearer auth, multi-tenancy) is HONESTLY DOCUMENTED in-repo, not hidden"

red_flags:
  - severity: "High"
    finding: "Single-engine reality: the headline multi-engine portability moat is functionally unbuilt — only Temporal interprets the IR; the Argo path is a non-interpreting stub ('capability-dispatch parity ... deliberately out of scope'). This is the largest single architectural debt and the one most likely to block a sale predicated on the portability claim. (Inherited from Wave A 5.1; quantified here as remediation cost.)"
    evidence:
      - "scripts/e2e/manifests/argo-ir-interpreter.yaml:13-14"
      - "Wave A 5.1: services/engine-adapter/internal/infrastructure/argo_engine.go:62-98; scripts/e2e/e2e-argo.sh:232-266"
  - severity: "Medium"
    finding: "Deferred multi-tenancy / no RBAC: API auth is a single shared static bearer token (no roles, no per-user authz, silently disabled when the key is empty), and multi-namespace policy + a policy-admin API are deferred to M7+. Data-layer namespace isolation holds, but the control-plane authz model is minimal — a real gap for any multi-tenant SaaS positioning."
    evidence:
      - "services/api-gateway/internal/api/auth.go:13-25"
      - "services/workflow-compiler/internal/config/config.go:60-61"
  - severity: "Medium"
    finding: "Fuzz + benchmarks exist but never gate CI, and the domain coverage gate enforces only on CHANGED services per PR — both are silent-regression windows. (Inherited from Wave A 5.7; counted here as test-debt.) Mitigated today: all 7 domains ≥90% at HEAD."
    evidence:
      - "Wave A 5.7: grep fuzz/bench/benchstat over CI workflows → empty; _test-go.yml:132-134 (per-changed-service gate)"
  - severity: "Low"
    finding: "Stale contract debt: workflow_compiler.proto:50-53 still documents the unbounded in-memory map 'planned for M6' even though the C7 refactor landed (server is stateless). A reader of the contract would believe a since-fixed limitation is still live. Same drift class as ADR-034-vs-code (Wave A 5.1)."
    evidence:
      - "protos/zynax/v1/workflow_compiler.proto:50-53 vs services/workflow-compiler/internal/api/server.go:168-171"
  - severity: "Low"
    finding: "145 of 215 //nolint directives carry no same-line justification (all are scoped to a named linter, and most are idiomatic — defer Body.Close errcheck, wrapcheck passthrough). A stricter shop would require a reason on every suppression; low risk but a hygiene gap."
    evidence:
      - "cmd: 215 //nolint, 70 with inline '//' justification, 145 without; samples: argo_client.go defer resp.Body.Close() //nolint:errcheck"

green_flags:
  - strength: "Near-zero debt-marker density: 5 raw TODO/FIXME/HACK/XXX hits in the entire source tree, only 2 genuine, both EPIC+issue-tagged. Almost no scattered, untracked in-code debt — exceptional for a 7-service polyglot platform."
    evidence: ["cmd: grep markers → 5; helm/zynax-event-bus/values.yaml:10; helm/zynax-memory-service/values.yaml:10"]
  - strength: "Both M6 remediations actually landed in code: C8 (cel-go fail-closed guard) fully VERIFIED (interpreter.go:17,203,250 + go.mod:7); C7 (stateless compiler) VERIFIED in code — the in-memory map was REMOVED entirely, not just bounded (server.go:168-171)."
    evidence: ["services/engine-adapter/internal/domain/interpreter.go:216,250", "services/workflow-compiler/internal/api/server.go:22-24,168-171", "services/engine-adapter/go.mod:7"]
  - strength: "Best-in-class dependency-debt tracking: CVE pin floors cite the CVE id inline; the single pip-audit suppression is dated with a re-evaluate date + reason; renovate is grouped/scheduled/digest-pinned; govulncheck+bandit+pip-audit gate CI and a HIGH-sev dep is blocked on PRs."
    evidence: ["agents/sdk/pyproject.toml:26,35,86-90", "renovate.json:45-160", ".github/workflows/pr-checks.yml:69"]
  - strength: "Deferrals are tracked + dated: every deferral carries a GH issue number and date (#466, #1103 w/ 2026-06-12 operator decision + 4 code-verified gaps kept behind a strict xfail); 3 Proposed ADRs are dated and mapped to named M7 EPICs. Debt is managed, not accumulated."
    evidence: ["state/current-milestone.md:26,130", "docs/adr/INDEX.md:44,46,47", "automation/tests/test_platform_readiness.py:52"]
  - strength: "Lint suppression is scoped, never blanket: 0 bare //nolint across the whole Go tree; every directive names a linter; the high-risk gosec G115/G304 cases carry inline justifications (corroborates Wave A 5.5)."
    evidence: ["cmd: grep '//nolint($|[^:])' minus '//nolint:' → empty", "Wave A 5.5: tools/golangci-lint.yml:17-33"]

open_questions:
  - "What is the eng-week cost to bring Argo to IR-interpretation parity (sidecar/operator running IRInterpreter vs re-implementing the state machine in Argo DAG primitives)? This is the single biggest debt line and Wave A 5.1 left it open too."
  - "Is the single-bearer-token auth model intended for production, or is an OIDC/RBAC layer planned? No RBAC ADR or issue was found in scope."
  - "Will the stale workflow_compiler.proto:50-53 in-memory-map comment and the Proposed-status ADR-034 (which contradicts shipped idempotent apply) be reconciled, given both describe since-fixed/never-shipped behavior?"

unknowns:
  - "Whether the renovate backlog is actually being drained (open dep-update PR count) — cannot query live GitHub PR state read-only/offline; config is verified, throughput is UNKNOWN."
  - "True remediation eng-weeks are estimates from code-surface size, not from a measured spike; the Argo-parity figure especially is a planning-grade range, not a verified cost."
  - "Whether any additional debt lives only in private canvas.private.md files (gitignored, Tier 2) — not inspectable in this read-only public-tree audit."

cross_references:
  - to_agent: "5.1 Architecture"
    note: "Single-engine reality (Argo stub) and ADR-034-vs-code / stale proto comment are 5.1's findings; I inherit and cost them as debt, not re-score. C7 stale comment confirmed at HEAD."
    evidence: ["scripts/e2e/manifests/argo-ir-interpreter.yaml:13-14", "protos/zynax/v1/workflow_compiler.proto:50-53"]
  - to_agent: "5.5 Engineering"
    note: "Highest-leverage debt items 5.5 flagged for me: (1) ADR-010 SDK base-class drift (root cause of the SDK type:ignore[misc] cluster); (2) 3 funlen-suppressed parsers. Both confirmed; both modest."
    evidence: ["agents/sdk/src/zynax_sdk/agent.py:59", "Wave A 5.5 cross-ref to 5.14"]
  - to_agent: "5.7 Testing"
    note: "Fuzz/bench-not-gated, per-changed-service coverage gate, engine-adapter BDD hard-skip — all counted here as test-debt; severity matches 5.7's Medium ratings, all tracked."
    evidence: ["Wave A 5.7 red_flags", "services/engine-adapter/tests/steps_test.go:17"]
  - to_agent: "5.2 Security"
    note: "Single-bearer-token auth (no RBAC), fail-open PolicyGate quota, deferred multi-tenancy — debt with a security dimension; defer the security scoring to 5.2."
    evidence: ["services/api-gateway/internal/api/auth.go:13-25", "services/workflow-compiler/internal/domain/policy_gate.go:184"]

recommendations:
  - priority: "P0"
    action: "Reconcile the C7 stale proto comment (workflow_compiler.proto:50-53) and the Proposed ADR-034 with the shipped stateless/idempotent code, AND re-label the multi-engine portability claim as 'Temporal reference interpreter; Argo execution parity in progress' until Argo interprets the IR. Est: < 0.5 eng-week (docs/contract only)."
    rationale: "These are the only debt items that misrepresent shipped reality — exactly the delivery-vs-narrative drift class the diligence exists to catch (Part 1 §1.10). Cheapest, highest-trust fix."
  - priority: "P1"
    action: "Bring the Argo engine to IRInterpreter parity (run the existing engine-neutral IRInterpreter as an Argo pod/operator) + add a cross-engine IR-parity test. Est: 4–8 eng-weeks (the single largest debt line; range is planning-grade)."
    rationale: "Converts the headline portability moat from structural to functional; today no test enforces cross-engine equivalence (Wave A 5.1)."
  - priority: "P1"
    action: "Add a real authz model (RBAC or OIDC) above the single shared bearer token, and land the deferred multi-namespace policy + policy-admin API. Est: 3–5 eng-weeks."
    rationale: "The single-token / single-namespace model blocks multi-tenant SaaS positioning; it is the largest control-plane architectural debt for a sale (auth.go:13-25; config.go:60-61)."
  - priority: "P2"
    action: "Wire fuzz + benchstat-vs-baseline into a scheduled CI job and make the domain coverage gate a global per-PR floor; add inline justifications to the 145 unjustified //nolint directives (or accept idiomatic ones via a config policy). Est: 1–2 eng-weeks total."
    rationale: "Closes the silent-regression windows (Wave A 5.7) and tightens lint hygiene; low risk since all domains are already ≥90%."
```

---

## (b) §6.2 Prose section

## 5.14 Technical Debt — Score: 8 (High) · debt is LOW (inverted scale)

**Mission recap:** Inventory technical debt, quantify remediation cost in eng-weeks, classify Now/Next/Later, and drift-test whether the M6 remediations C7 (in-memory compiler map) and C8 (cel-go guard) actually landed. Remember the scale is inverted — a high score means low debt.

**Verdict:** Zynax carries strikingly little technical debt, and what exists is honestly documented and tracked. Source-level debt-marker density is near zero — five raw TODO/FIXME/HACK/XXX hits in the entire tree, of which three are non-debt (regex detector strings, an `mktemp` template) and only two are genuine, both EPIC- and issue-tagged Helm placeholders. Lint suppression is scoped (zero bare `//nolint`), CVE pins are documented inline, and deferrals carry GitHub issue numbers and dates. The real debt is architectural, not hygienic: the multi-engine portability moat is functionally unbuilt (only Temporal interprets the IR; Argo is a non-interpreting stub), and the control-plane authz model is a single shared bearer token with multi-tenancy deferred to M7+. Both are inherited from Wave A 5.1/5.2's zones and counted here as remediation cost, not re-scored. An 8 (low debt) rather than a 9 because those two architectural gaps are material to a sale.

**Sub-dimension scores (inverted — high = low debt):**

| Sub-dimension | Score | Confidence | Evidence |
|---|---|---|---|
| Debt-marker density (TODO/FIXME/HACK/XXX) | 9 | High | grep → 5 hits, 2 genuine; helm values.yaml:10 (EPIC-tagged) |
| Lint suppression hygiene (//nolint) | 8 | High | 215 scoped, 0 bare; 70 justified / 145 idiomatic |
| Python type-ignore / noqa hygiene | 8 | High | 24 type:ignore (all code-scoped) + 5 noqa; agent.py:24 |
| Arch debt: deferred isolation/RBAC | 6 | High | auth.go:13-25 (single bearer); config.go:60-61 (M7+ defer) |
| Arch debt: single-engine reality (Argo stub) | 5 | High | argo-ir-interpreter.yaml:13-14; Wave A 5.1 |
| Test/lint debt: skips, exemptions, fail-open | 7 | High | 6 t.Skip (1 hard); 0 coverage exemptions; Wave A 5.7 |
| Dependency debt (renovate + CVE pins) | 9 | High | renovate.json:45-160; pyproject.toml:26,35,86-90 |
| Debt-tracking discipline (issue-tagged, dated) | 9 | High | current-milestone.md:26,130; adr/INDEX.md:44,46,47 |

**Drift test (C7 / C8 are the mandated targets):**
- *C7 — "stateless workflow-compiler, in-memory map removed (fixed M6)"* → **PARTIAL.** VERIFIED in code: the `Server` struct holds no map (`server.go:27-31`) and `GetCompiledWorkflow` unconditionally returns NOT_FOUND (`server.go:168-171`) — the map was removed outright, a stronger fix than a bounded cache. But the proto contract comment is **stale debt**: `workflow_compiler.proto:50-53` still documents the "unbounded in-memory map ... planned for M6." Code fixed; contract doc lagging.
- *C8 — "cel-go guard evaluator replaces bespoke fail-open evaluator (fixed M6)"* → **VERIFIED.** Real `github.com/google/cel-go` (`interpreter.go:17,203,250`), pinned `v0.28.1` (`go.mod:7`), and `evalGuard` is fail-**closed** on every path (`interpreter.go:216,220-259`). The fix fully landed.
- *"Tech debt is large but untracked"* → **CONTRADICTED.** Density is near-zero; every deferral and CVE suppression is issue-tagged and dated.

**Red flags (severity-ordered):**
1. **High** — Single-engine reality: only Temporal interprets the IR; Argo is a non-interpreting stub, parity "deliberately out of scope" (`argo-ir-interpreter.yaml:13-14`; Wave A 5.1). Largest debt line; threatens the portability sale narrative.
2. **Medium** — Deferred multi-tenancy / no RBAC: single shared static bearer token, disabled when empty (`auth.go:13-25`); multi-namespace policy + admin API deferred to M7+ (`config.go:60-61`).
3. **Medium** — Fuzz/bench never gate CI; domain coverage gate is per-changed-service — silent-regression windows (Wave A 5.7). Mitigated: all domains ≥90% today.
4. **Low** — Stale `workflow_compiler.proto:50-53` in-memory-map comment contradicts the shipped stateless compiler.
5. **Low** — 145 of 215 `//nolint` lack a same-line justification (all scoped, mostly idiomatic).

**Green flags:**
- Near-zero debt-marker density (5 hits, 2 genuine, both issue-tagged).
- Both M6 remediations landed in code (C8 fully; C7 the map was removed entirely).
- Best-in-class dependency-debt tracking — inline CVE-cited pin floors, a dated/re-evaluated pip-audit suppression, grouped digest-pinned renovate, CVE gates in CI (`pyproject.toml:26,35,86-90`; `renovate.json:45-160`).
- Deferrals carry GH issue numbers + dates and an honest xfail tripwire (`current-milestone.md:130`; `test_platform_readiness.py:52`); 0 bare `//nolint`.

**Open questions / unknowns:** Argo-parity eng-week cost (planning-grade only); whether RBAC/OIDC is planned; whether the renovate backlog is actually draining (live PR count not queryable offline); possible Tier-2 debt in gitignored private canvases.

**Recommendations:** **P0** — reconcile the stale proto comment + ADR-034 and re-label the portability claim (< 0.5 wk, docs only). **P1** — Argo IR-interpreter parity + cross-engine test (4–8 wk); real authz/RBAC + multi-namespace policy (3–5 wk). **P2** — gate fuzz/bench, global coverage floor, justify the idiomatic nolints (1–2 wk).

**Total remediation bill (Now/Next/Later):** Now (P0) ≈ 0.5 eng-week; Next (P1) ≈ 7–13 eng-weeks; Later (P2) ≈ 1–2 eng-weeks. **Total ≈ 8.5–15.5 eng-weeks** — modest for a 7-service platform, and concentrated in two architecture lines (Argo parity, authz) rather than spread as scattered hygiene debt.

**Cross-references:** 5.1 Architecture (Argo stub, ADR-034/stale-proto drift — inherited); 5.5 Engineering (ADR-010 SDK base-class drift, funlen parsers); 5.7 Testing (fuzz/bench not gated, coverage gate, engine-adapter BDD skip); 5.2 Security (single-bearer auth, fail-open quota, deferred multi-tenancy — security scoring deferred there).
<!-- Zynax Investment-Grade Due-Diligence — Agent 5.15 Maintainability — Wave B (derived technical) -->
<!-- Issue #1403 · target HEAD e3135a6 (2026-06-20) · READ-ONLY static audit (grep/read + simple bash; no go build/test) -->
<!-- Consumes Wave A: 5.5 Engineering, 5.1 Architecture, 5.24 Repo Health (docs/due-diligence/2026-06-20-dd-wave-a-findings.md ) -->

# Agent 5.15 — Maintainability (Wave B, derived technical)

## (a) §3.4 Handoff packet

```yaml
agent: "5.15 Maintainability"
wave: "B"
dimension_groups: ["D4", "D3", "D15"]
overall_score: 7
overall_confidence: "High"

sub_scores:
  - dimension: "Change amplification — does a typical change touch 1 module or many? (proto blast radius)"
    score: 8
    confidence: "High"
    justification: "Traced 3 recent feats: each stayed within ONE logical module. Even a proto change is bounded — additive fields + auto-regen stubs + buf-breaking gate mean a contract change touched only 4 files (proto + 2 generated stubs + .feature), zero service-code cascade."
    evidence:
      - "git show --stat 1be3ccb (#1339 data-context scoping) → 3 files, all engine-adapter/internal/domain/ (datacontext.go, datacontext_test.go, interpreter.go)"
      - "git show --stat 06e5bea (#1249 output/input binding IR proto change) → 4 files: workflow_compiler.proto + generated/go .pb.go + generated/python _pb2.py + .feature; NO service .go touched"
      - "protos/zynax/v1/workflow_compiler.proto:160,168 (output_bindings=5, input_bindings=6 — additive field numbers, no renumber)"
      - ".github/workflows/proto-generate.yml (stub auto-regen on main); pr-checks.yml:129-135 buf breaking gate (cited by 5.1) caps proto blast radius to additive-only"

  - dimension: "Modularity & coupling — service independence, shared-lib coupling, interface stability"
    score: 8
    confidence: "High"
    justification: "7 services + 2 libs are SEPARATE Go modules; Go's internal/ rule mechanically forbids cross-service internal imports (each service references only its own internal/). Shared libs are observability/config only — never imported into any internal/domain/. Inter-service coupling is gRPC-only (5.1). WorkflowEngine port introduced once and unreshaped since."
    evidence:
      - "find services libs cmd -name go.mod → 11 separate modules (each service + libs/zynaxobs + libs/zynaxconfig)"
      - "grep -rhoE 'zynax/services/[a-z-]+/internal' services → each count matches that service's OWN internal only (16-22); no cross-service internal import (Go internal/ visibility enforces this)"
      - "grep libs/zynaxobs|libs/zynaxconfig in services/*/internal/domain/ → EMPTY (shared libs never reach domain logic; used at cmd/ wiring + infrastructure only)"
      - "wc -l libs/*/*.go → ~700 LoC non-test lib code (small coupling surface)"
      - "git log -S 'WorkflowEngine interface' engine.go → single introducing commit 515f287 (#301/#308); port stable since (5.1 engine.go:17-41)"

  - dimension: "Readability & onboarding — can a NEW owner navigate via AGENTS.md + docs without the founder?"
    score: 8
    confidence: "High"
    justification: "21 AGENTS.md files (one per layer/service), a 35+-row root Knowledge Base Index, per-service Pre-Code Checklist + exact uniform directory layout, 37 ADRs, 8 docs/patterns guides, quickstart + developer-guide. Code documents WHY with ADR citations (5.5 corroborated). Self-serve navigation is strong. Minor: one onboarding-doc drift (CI-enforcement claim, see red flag)."
    evidence:
      - "find -name AGENTS.md → 21 files: root + services/ + 7 service dirs + agents tree + protos + spec + infra + cmd"
      - "AGENTS.md:222-258 (Knowledge Base Index — 35+ rows routing every concern to a doc)"
      - "services/AGENTS.md:9-23 (Pre-Code Checklist) + :26-44 (exact directory structure every service follows)"
      - "ls docs/adr | grep -c ADR → 37 ADRs; ls docs/patterns → 8 pattern guides; docs/quickstart.md (265 lines) + docs/developer-guide.md (167)"
      - "5.5 packet: comments explain WHY w/ ADR/canvas citations (service.go:83, interpreter.go:191) — corroborates human-navigability"

  - dimension: "Knowledge concentration / bus factor (cross-ref 5.8 / 5.24)"
    score: 3
    confidence: "High"
    justification: "Cross-reference, not re-scored: bus factor = 1 per 5.24 (one human committer, 772 commits, no second human author in shortlog or last-50). MAINTAINERS.md does NOT exist (issue #494 open). The single most material sustainability risk; process/docs mitigate but do not remove single-point-of-failure."
    evidence:
      - "docs/due-diligence/2026-06-20-dd-wave-a-findings.md:117-122 (5.24 red flag: bus factor=1, git shortlog → single human identity 772 commits)"
      - "ls MAINTAINERS.md → 'No existe el fichero' (does not exist); 5.24 cites open issue #494 'create MAINTAINERS.md'"
      - "§1.9 packet: CNCF gate is 'social, not technical' — single-maintainer bus factor"

  - dimension: "AI-authorship risk — is code human-ownable or only AI-regenerable?"
    score: 7
    confidence: "Medium"
    justification: "Strong mitigations: 62 REASONS Canvases capture per-feat WHY/DoD/decision-history with path:line refs (incl. Superseded/Rejected records); 1924 lines of ai-learnings tacit knowledge; ZERO AI-attribution scaffolding leaked into Go source; moderate ~10K LoC service-internal (human-comprehensible, not a sprawling generated mass). Residual risk: heavy AI-assisted authorship by one author means comprehension still concentrates in the founder + the prompt corpus rather than a human team."
    evidence:
      - "ls docs/spdd | wc -l → 62 canvas dirs (framework cites '~30' — favorable docs lag); each has canvas.md + SECURITY-REVIEW.md"
      - "docs/spdd/1359-zero-temporal-engine/canvas.md:1-50 (records problem, DoD, entities, cites main.go:73/186-196; Status: Superseded by #1456, ADR-037 Rejected — durable decision history)"
      - "grep 'Assisted-by|Generated with|claude' services libs --include=*.go → 0 (no AI scaffolding in source)"
      - "find services -name '*.go' -path '*internal*' not _test | wc -l total → 10048 LoC (moderate, human-ownable size)"
      - "wc -l docs/ai-learnings/*.md → 1924 lines across 7 expert-domain knowledge files"

drift_test:
  - claim: "A typical change touches one module; proto changes do not cascade (low change amplification)."
    result: "VERIFIED"
    evidence:
      - "#1339 (1be3ccb) → 3 files single service/domain; #1455 (4d582c2) → 15 files but all one CLI module + its spec/schema/docs, no cross-service edit; #1249 (06e5bea) proto change → 4 files (proto+2 stubs+.feature), 0 service code"
      - "additive-field discipline (protos/AGENTS.md:56-57 never remove/renumber) + buf-breaking gate (5.1) + auto-regen (proto-generate.yml) bound proto blast radius"
  - claim: "Layer/import boundaries are CI-enforced (services/AGENTS.md:22,46 'Import layering enforced (CI fails on violations)')."
    result: "CONTRADICTED"
    evidence:
      - "services/AGENTS.md:22 '10. Import layering enforced (CI fails on violations).' + :46 '(CI-enforced import analysis)'"
      - "grep depguard|import-boundary|importas|gomodguard|forbidigo in tools/golangci-lint.yml → EMPTY; grep import-layering in Makefile/cmd/zynax-ci/.github/workflows → EMPTY"
      - "boundary HOLDS in practice (grep internal/api|internal/infrastructure in services/*/internal/domain/ → EMPTY) but by convention + review only, not by any mechanical CI gate — corroborates 5.1 Medium red flag"
  - claim: "Services are independent modules with no cross-service code coupling (modular, gRPC-only)."
    result: "VERIFIED"
    evidence:
      - "11 separate go.mod; each service references only its OWN internal/ (Go visibility enforces); shared libs never in internal/domain/"
      - "5.1: 8 inter-service edges all gRPC, separate Postgres per stateful service, no cross-service internal import"

red_flags:
  - severity: "High"
    finding: "Bus factor = 1 (cross-ref 5.24/5.8). One human committer owns ~100% of non-bot history; MAINTAINERS.md does not exist (issue #494 open). The codebase is unusually well-documented and modular, which LOWERS but does not remove the single-point-of-failure: a 90-day acquirer hand-off is feasible on the artifacts, but day-to-day evolution velocity, the prompt/canvas corpus, and undocumented operational judgement still concentrate in one person."
    evidence:
      - "docs/due-diligence/2026-06-20-dd-wave-a-findings.md:117-122 (git shortlog → single human identity, 772 commits, no second human author)"
      - "ls MAINTAINERS.md → does not exist (open issue #494)"
  - severity: "Medium"
    finding: "The low-coupling invariant that underwrites the whole maintainability case — the domain/ layer never importing api/infrastructure — is DOCUMENTED as CI-enforced but is actually convention-only. services/AGENTS.md:22/46 claim 'CI fails on violations' / 'CI-enforced import analysis'; no depguard/import-linter/fitness function exists. A new owner trusting the onboarding doc would believe the boundary is mechanically defended; one careless import silently re-introduces coupling. This is doc-vs-reality drift (Part 1 §1.10 class) sitting in the onboarding contract itself."
    evidence:
      - "services/AGENTS.md:22, :46 (enforcement claim)"
      - "tools/golangci-lint.yml (no import-boundary linter) + Makefile/CI (no import-layering check) → both EMPTY on grep"
      - "5.1 architecture Medium red flag: layer-boundary separation has no automated enforcement"
  - severity: "Low"
    finding: "ADR-010 doc-vs-code drift (cross-ref 5.5): the Python SDK ships the base-class the ADR forbids; Protocol runtime unbuilt. Raises future change cost on the agent SDK and is a maintainability/onboarding trap (the spec misdescribes the shipped abstraction)."
    evidence:
      - "docs/due-diligence/2026-06-20-dd-wave-a-findings.md:102-106 (ADR-010 CONTRADICTED: agent.py:59 base class vs ADR-010:54 Protocol-never-base-class)"

green_flags:
  - strength: "Low change amplification — proven by trace, not asserted. Recent feats touched a single module each; a proto change cascaded to only 4 files (proto + auto-regenerated stubs + .feature), protected by additive-only rules + a buf-breaking CI gate + auto-regen."
    evidence:
      - "git show --stat 1be3ccb (3 files, 1 domain), 06e5bea (4 files, 0 service code), 4d582c2 (1 CLI module)"
      - "protos/AGENTS.md:56-57; proto-generate.yml; pr-checks.yml:129-135 (buf breaking)"
  - strength: "Hard modular boundaries enforced by the build system: 11 separate Go modules so Go's internal/ visibility MECHANICALLY forbids cross-service coupling; shared libs are small (~700 LoC) and confined to wiring/infra, never domain logic; inter-service is gRPC-only."
    evidence:
      - "11 go.mod; grep shared-lib in internal/domain → EMPTY; per-service internal references self-only"
  - strength: "Self-serve onboarding substrate: 21 layered AGENTS.md, a 35-row Knowledge Base Index, per-service Pre-Code Checklist + uniform directory layout, 37 ADRs, 8 pattern guides, quickstart + dev-guide. A new owner can navigate the architecture from docs alone."
    evidence:
      - "find -name AGENTS.md → 21; AGENTS.md:222-258; services/AGENTS.md:9-44; 37 ADRs; 8 docs/patterns"
  - strength: "Tacit knowledge is externalized, not in one head: 62 REASONS Canvases (with DoD, entity maps, path:line refs, and Superseded/Rejected decision history) + 1924 lines of ai-learnings + per-canvas security reviews. No AI scaffolding leaked into source — code reads as human-owned."
    evidence:
      - "62 canvas dirs; docs/spdd/1359/canvas.md:1-50 (records Superseded decision); 1924 LoC ai-learnings; 0 AI-attribution markers in Go"

open_questions:
  - "Could an acquirer's team OWN this in 90 days? Likely YES for comprehension/hand-off (docs + canvases + modular boundaries are unusually strong); UNCERTAIN for sustained velocity — the SPDD/prompt-driven workflow is itself a learned skill concentrated in one author."
  - "Is the CI-enforced-import-layering claim a stale aspiration (a linter was planned/removed) or never built? Needs maintainer confirmation; either way the onboarding doc currently misstates reality."
  - "Does the heavy AI-assisted authorship mean some modules are only economically MODIFIABLE via the same AI tooling? Not testable read-only; comment density + canvas coverage argue against, but unverified."

unknowns:
  - "Whether any module is 'AI-regenerable-only' — cannot be proven statically. Mitigations (WHY-comments, 62 canvases, moderate LoC, zero AI scaffolding) strongly suggest human-ownable, but the counterfactual (a human team modifying without AI) was not executed (Medium confidence on AI-authorship sub-score)."
  - "Exact onboarding TIME for a new owner — inferred from doc completeness, not measured."

cross_references:
  - to_agent: "5.8"
    note: "Bus factor = 1 is the dominant maintainability/sustainability risk; MAINTAINERS.md absent (issue #494). Recorded as cross-reference per §3.1 — owned/scored by 5.24/5.8, not re-scored here."
    evidence: ["docs/due-diligence/2026-06-20-dd-wave-a-findings.md:117-122", "ls MAINTAINERS.md → absent"]
  - to_agent: "5.1"
    note: "Confirms 5.1's Medium red flag: layer boundary has no CI enforcement. I add that the onboarding doc (services/AGENTS.md:22,46) actively CLAIMS CI enforcement — drift in the contract itself, raising new-owner risk."
    evidence: ["services/AGENTS.md:22,46", "tools/golangci-lint.yml (no import linter)"]
  - to_agent: "5.14"
    note: "Highest-leverage maintainability debt: (1) add an import-boundary fitness function (depguard) to make the convention real and match the doc; (2) reconcile ADR-010 (5.5)."
    evidence: ["services/AGENTS.md:22", "docs/due-diligence/2026-06-20-dd-wave-a-findings.md:102-106"]
  - to_agent: "5.12"
    note: "SPDD canvas corpus (62) + ai-learnings (1924 LoC) are the primary AI-authorship-risk mitigation; quality/coverage of that corpus is 5.12's zone."
    evidence: ["ls docs/spdd | wc -l → 62", "wc -l docs/ai-learnings → 1924"]

recommendations:
  - priority: "P0"
    action: "Recruit + document a second maintainer (close #494 MAINTAINERS.md). Distribute the SPDD/prompt-driven workflow knowledge, not just the code."
    rationale: "Bus factor = 1 is the single largest barrier to a new team owning this; docs reduce hand-off cost but do not remove the velocity single-point-of-failure."
  - priority: "P1"
    action: "Add an import-boundary fitness function (depguard or a custom analyzer) to the golangci-lint config / a CI step, so the domain→api/infra rule is mechanically enforced — OR correct services/AGENTS.md:22,46 to say 'convention, verified in review'."
    rationale: "The low-coupling invariant is the foundation of the maintainability story; today it is undefended AND the onboarding doc misrepresents it as defended. Cheap to fix; removes a real silent-regression path and a doc-drift."
  - priority: "P2"
    action: "Reconcile ADR-010 with the shipped SDK (implement Protocol or supersede the ADR) and refresh the '~30 canvases' references to the real 62."
    rationale: "Removes onboarding traps where the spec/docs misdescribe the shipped abstraction or scale."
```

---

## (b) §6.2 Prose section

## Maintainability — Score: 7 (High)

**Mission recap:** Assess how cheaply and safely a NEW team could evolve this codebase — change amplification, modularity/coupling, readability/onboarding, knowledge concentration (bus factor), and AI-authorship risk — and drift-test a recent feature's real blast radius.

**Verdict:** On the *technical* axes this is a genuinely low-change-cost, highly modular, self-documenting codebase that a new owner could comprehend and hand-off in well under 90 days. The change-amplification trace is the headline: recent features each stayed inside a single module, and even a contract change to a `.proto` rippled to only four files because additive-only field discipline, auto-regenerated stubs, and a `buf-breaking` CI gate cap the proto blast radius. The build system itself enforces modularity (11 separate Go modules; cross-service `internal/` imports are mechanically impossible), and the documentation substrate (21 layered `AGENTS.md`, a 35-row Knowledge Base Index, 37 ADRs, 62 REASONS Canvases capturing per-feature *why* and decision history) is unusually strong. Two things hold the score at 7 rather than 8–9: the **people** axis — bus factor = 1, no `MAINTAINERS.md` (cross-ref 5.24/5.8) — and a **doc-vs-reality drift in the onboarding contract itself**: `services/AGENTS.md` claims the low-coupling import boundary is CI-enforced, but no import linter exists; the invariant holds by convention only.

**Sub-dimension scores:**

| Sub-dimension | Score | Confidence | Evidence |
|---|---|---|---|
| Change amplification (proto blast radius) | 8 | High | #1339→3 files/1 domain; #1249 proto→4 files/0 service code; additive-only + buf-breaking + auto-regen |
| Modularity & coupling (independence, shared-lib, interface stability) | 8 | High | 11 go.mod; internal/ visibility blocks cross-service; libs absent from domain; WorkflowEngine port unreshaped |
| Readability & onboarding (AGENTS.md + docs self-serve) | 8 | High | 21 AGENTS.md; KB Index AGENTS.md:222-258; Pre-Code Checklist services/AGENTS.md:9-44; 37 ADRs; 8 patterns |
| Knowledge concentration / bus factor (cross-ref) | 3 | High | 5.24:117-122 (one human, 772 commits); MAINTAINERS.md absent (#494) |
| AI-authorship risk (human-ownable vs AI-only) | 7 | Medium | 62 canvases w/ path:line + decision history; 1924 LoC ai-learnings; 0 AI scaffolding in Go; ~10K LoC |

**Drift test:**
- "A typical change touches one module; proto changes do not cascade" → **VERIFIED**. `#1339` (data-context) = 3 files all in one service's `internal/domain/`; `#1249` (proto IR-binding fields) = 4 files (`.proto` + both auto-regenerated stubs + `.feature`), **zero service code touched**; `#1455` (context-injection) = 15 files but all one CLI module + its own spec/schema/docs.
- "Layer/import boundaries are CI-enforced" (`services/AGENTS.md:22,46`) → **CONTRADICTED**. No `depguard`/import-linter/fitness function exists in `tools/golangci-lint.yml`, the `Makefile`, `cmd/zynax-ci/`, or any workflow. The boundary holds in practice (no `domain→api/infra` import) but by convention + review only.
- "Services are independent, gRPC-only, no cross-service code coupling" → **VERIFIED**. 11 separate Go modules; each references only its own `internal/`; shared libs never reach `internal/domain/`.

**Red flags (severity-ordered):**
1. **High — Bus factor = 1** (cross-ref 5.24/5.8). One human committer owns ~100% of non-bot history; `MAINTAINERS.md` does not exist (issue #494). Excellent docs lower hand-off cost but the day-to-day evolution velocity and the SPDD/prompt corpus still concentrate in one person.
2. **Medium — the low-coupling invariant is claimed CI-enforced but is convention-only.** `services/AGENTS.md:22/46` says "CI fails on violations" / "CI-enforced import analysis"; no such linter exists. A new owner trusting the onboarding contract would over-rely on a defense that isn't there; one careless import silently re-couples the layers. Doc-vs-reality drift inside the onboarding doc — exactly the class Part 1 §1.10 exists to catch.
3. **Low — ADR-010 doc-vs-code drift** (cross-ref 5.5): the Python SDK ships the base class the ADR forbids; the spec misdescribes the shipped abstraction, an onboarding trap on the agent SDK.

**Green flags:**
- Low change amplification proven by trace (single-module features; 4-file proto change), protected by additive-only rules + `buf-breaking` gate + auto-regen.
- Hard modular boundaries enforced by the *build system* — 11 modules make cross-service `internal/` coupling impossible; shared libs are small (~700 LoC) and confined to wiring/infra.
- Self-serve onboarding: 21 layered `AGENTS.md`, a 35-row Knowledge Base Index, per-service checklist + uniform layout, 37 ADRs, 8 pattern guides.
- Tacit knowledge externalized: 62 REASONS Canvases (with DoD, entity maps, `path:line` refs, and recorded Superseded/Rejected decisions), 1924 lines of `ai-learnings`, and zero AI scaffolding in source — code reads as human-owned.

**Open questions / unknowns:**
- Could an acquirer own this in 90 days? Likely **yes** for comprehension/hand-off; **uncertain** for sustained velocity (the prompt-driven workflow is a concentrated skill).
- Is the CI-import-enforcement claim a stale aspiration or never built? Needs maintainer confirmation.
- Whether any module is economically modifiable only via the same AI tooling — cannot be proven statically (mitigations argue against; Medium confidence).

**Recommendations:**
- **P0** — Recruit + document a second maintainer (close #494); distribute the SPDD workflow knowledge, not just code.
- **P1** — Add an import-boundary fitness function (`depguard`) to make the convention real, OR correct `services/AGENTS.md:22,46` to "convention, verified in review."
- **P2** — Reconcile ADR-010 with the SDK; refresh the "~30 canvases" references to the real 62.

**Cross-references:** bus factor → 5.8/5.24; unenforced layer boundary → 5.1; debt items (depguard, ADR-010) → 5.14; canvas/ai-learnings corpus quality → 5.12.
# Agent 5.16 — Scalability Agent · Wave B (derived technical)

> Issue #1403 · HEAD `e3135a60e4abb20886d51f81d6448b22fe04cb64` · READ-ONLY audit.
> Every claim grounded in `path:line` / command-output, or marked `UNKNOWN`.
> Marketing/roadmap = `CLAIMED`; code/CI/contract/Helm-verified = `VERIFIED`.
> Consumes Wave A 5.1 Architecture (`docs/due-diligence/2026-06-20-dd-wave-a-findings.md`); 5.6 Performance treated as cross_reference, not blocking.

---

## (a) §3.4 Handoff packet

```yaml
agent: "5.16 Scalability"
wave: "B"
dimension_groups: ["D3", "D6"]   # contributes D14 (reliability) / D15 (maintainability) signals
overall_score: 5
overall_confidence: "High"

sub_scores:
  - dimension: "Service statelessness / horizontal scalability (7 services)"
    score: 6
    confidence: "High"
    justification: "api-gateway, workflow-compiler, event-bus are genuinely stateless; engine-adapter delegates fan-out to Temporal. BUT task-broker & agent-registry DEFAULT to in-memory maps (mutex-guarded) and the production umbrella leaves their DB DSN empty — i.e. the default deploy is NOT multi-replica-safe for the two stateful services."
    evidence:
      - "services/workflow-compiler/internal/api/server.go:21-24,46 — 'The compiler is stateless ... it is not stored anywhere'; GetCompiledWorkflow always NOT_FOUND (C7 'stateless workflow-compiler' now VERIFIED)."
      - "services/task-broker/cmd/task-broker/main.go:122-124 — 'if !cfg.DBEnabled { ... return infrastructure.NewMemoryRepo() }' (in-memory is the default branch)."
      - "helm/zynax-umbrella/values.yaml:46-48 — task-broker db.secretName: \"\" (no DSN wired → in-memory repo per values.yaml:31)."
      - "helm/zynax-umbrella/values.yaml:55-57 — agent-registry db.secretName: \"\" (same)."
      - "helm/zynax-task-broker/values.yaml:31 — 'Leave db.secretName empty to skip wiring ... (uses in-memory repo).'"
      - "services/api-gateway/templates/deployment.yaml:15-19 — RollingUpdate maxUnavailable:0/maxSurge:1; /readyz /livez /startupz probes (clean rollouts, no stickiness)."
  - dimension: "Helm horizontal-scale plumbing (HPA / PDB / replicas / resources)"
    score: 7
    confidence: "High"
    justification: "Every service chart ships HPA (CPU+mem), PDB, requests+limits, RollingUpdate maxUnavailable:0. Production override files raise replicas to 2-3 + enable HPA. Solid baseline; HPA scales on CPU/mem only (no queue-depth/custom metric for task-broker/event-bus); PDB minAvailable:1 is hardcoded (a 1-replica dev install blocks all voluntary evictions)."
    evidence:
      - "helm/zynax-api-gateway/templates/hpa.yaml:16-28 — Resource metrics cpu+memory only (no custom/queue metric)."
      - "helm/zynax-api-gateway/values-production.yaml:3,11-14 — replicaCount:3, autoscaling minReplicas:3 maxReplicas:10."
      - "helm/zynax-engine-adapter/values-production.yaml:3-10 — replicaCount:2 + HPA (improves on ADR-021's '1 replica' target; Temporal workers pull from task queue so N replicas are safe)."
      - "helm/zynax-task-broker/values-production.yaml:4-11,22-24 — replicaCount:2 + HPA + db.secretName:'task-broker-db' (production DOES wire Postgres → multi-replica-safe in the prod override)."
      - "helm/zynax-api-gateway/templates/pdb.yaml:8-9 — minAvailable:1 hardcoded (not value-driven)."
      - "helm/zynax-api-gateway/templates/deployment.yaml:122-123,49-55 — resources from values; requests 100m/64Mi, limits 500m/256Mi (dev), 250m-1000m/128-512Mi (prod)."
  - dimension: "Data-layer scale: Postgres single-instance vs HA, pooling, per-service ownership (ADR-021/026)"
    score: 4
    confidence: "High"
    justification: "Per-service schema ownership is real (ADR-008/021); BUT Postgres is a SINGLE-INSTANCE StatefulSet (replicas:1) with HA explicitly out of scope ('WON'T', EPIC #1073) — a deliberate, documented SPOF. Connection pools use pgxpool with NO tuning (default MaxConns)."
    evidence:
      - "helm/charts/postgres/templates/statefulset.yaml:1-5,13 — 'Single-instance Postgres StatefulSet ... HA / read-replicas are out of scope (EPIC #1073 \"WON'T\")'; spec.replicas:1."
      - "helm/charts/postgres/Chart.yaml:4-9 — 'Provisions a single cluster-level StatefulSet. task-broker, agent-registry, and memory-service each use a dedicated schema (ADR-008 + ADR-021).'"
      - "docs/adr/ADR-026-postgres-distribution.md:160-163,175 — 'No built-in HA ... high availability, failover, and managed backups are not provided'; out-of-scope table lists 'Postgres HA / multi-AZ / cross-region replication'."
      - "infra/docker-compose/postgres-zynax-init.sql:5-6 — separate DBs task_broker/agent_registry (per-service ownership confirmed by 5.1)."
      - "services/task-broker/internal/infrastructure/postgres/repository.go:32-33 — 'pgxpool.New(ctx, dsn)' with no Config tuning (MaxConns/MinConns default; ~4×CPU per pod) [Explore-verified, line approx]."
      - "docs/adr/ADR-021-horizontal-scale.md:133-142 — scale-target table (api-gw/wc/task-broker/agent-registry 2+, engine-adapter 1)."
  - dimension: "Event bus: NATS JetStream throughput, durable consumers, replay, backpressure, ordering (ADR-022)"
    score: 5
    confidence: "Medium"
    justification: "Durability semantics are solid (durable consumers, AckExplicit, MaxDeliver=5, exponential backoff, DLQ, DeliverLast, FileStorage). BUT every stream is created with Replicas:1 and the NATS subchart deploys a single (un-clustered) node — JetStream is a SPOF and not throughput-scaled. Ordering is per-subject FIFO; no MaxAckPending/AckWait tuning."
    evidence:
      - "services/event-bus/internal/infrastructure/nats.go:139-145 — StreamConfig Retention:LimitsPolicy, Storage:FileStorage, Replicas:1 (no stream replication)."
      - "helm/charts/nats/values.yaml:32-43 — nats.config.jetstream fileStore PVC enabled; NO cluster.enabled / replica override (community chart default = single node)."
      - "docs/adr/ADR-022-event-bus-architecture.md:40,53,71 — 'event-bus Go service is a stateless Deployment ... All durability ... lives entirely in JetStream'; 'NATS JetStream is a cluster-level StatefulSet' (claimed; chart ships single node)."
      - "services/event-bus/internal/infrastructure/nats.go:356-366 (Explore-verified) — nats.Durable, nats.AckExplicit, nats.MaxDeliver(5), nats.BackOff(RetryBackoff 1s/5s/30s/2m/5m), nats.DeliverLast; DLQ stream WorkQueuePolicy at nats.go:328-350."
      - "services/event-bus/internal/infrastructure/nats.go:188-189 — Publish subject = event.Type (per-subject ordering); MaxAckPending/AckWait not set (SDK default)."
  - dimension: "Multi-tenancy / namespace isolation (real vs cosmetic) + noisy-neighbor"
    score: 4
    confidence: "Medium"
    justification: "Mixed: memory-service enforces real namespace isolation (Redis key prefix + Postgres WHERE namespace) and engine-adapter datacontext keys on (RunID,Namespace); BUT event-bus namespace is cosmetic (subject derived from event.Type, all namespaces share one stream) and task-broker/agent-registry carry no namespace at all. No per-tenant quotas/isolation → noisy-neighbor risk. True multi-tenant isolation deferred post-M8 (Part-1 §1.8)."
    evidence:
      - "services/event-bus/internal/infrastructure/nats.go:178,188-189 — Namespace carried in envelope only; subject=event.Type (NOT namespaced) → no per-namespace stream/consumer isolation."
      - "services/memory-service/internal/infrastructure/postgres/pgvector.go:28 (Explore-verified) — 'Namespace isolation is enforced via WHERE namespace = $N on every query'; redis_kv.go:18 '{ns}:{key}'."
      - "services/engine-adapter/internal/domain/datacontext.go:26-34 (Explore-verified) — DataContextScope{RunID,Namespace}; cross-namespace read denied (ScopeError)."
      - "grep namespace in task-broker/agent-registry domain → NOT FOUND (Explore: no namespace field in those domain models)."
      - "ADR-022:52 — topic authorization / namespace isolation / rate limits 'planned for M7' (CLAIMED, gRPC chokepoint not yet enforcing)."
  - dimension: "Failure isolation: partition behavior, retries, idempotency, Temporal durability"
    score: 6
    confidence: "Medium"
    justification: "Strong where Temporal/Postgres back it: Temporal activity RetryPolicy (exp backoff, non-retryable types), task-broker recoverInFlight on restart, upsert idempotency, api-gateway manifest-hash idempotent apply, DLQ on the bus. Weaker elsewhere: inter-service gRPC clients enforce per-call timeout but NO retry; event dedup via ce-id header is tracing-only (not NATS Msg-Id dedup); event publish is best-effort (no retry)."
    evidence:
      - "services/engine-adapter/internal/infrastructure/temporal_workflow.go:77-86 (Explore-verified) — RetryPolicy InitialInterval 1s, Backoff 2.0, Max 30s, NonRetryableErrorTypes."
      - "services/task-broker/cmd/task-broker/main.go:112,150-160 — recoverInFlight re-launches non-terminal tasks on startup ('a broker restart never loses an in-flight fan-out')."
      - "services/api-gateway/internal/domain/apply.go:29-35,111-131 (per 5.1) — deterministic SHA-256 manifest id; running id reused (idempotent)."
      - "services/api-gateway/internal/infrastructure/clients.go (Explore) — per-call context.WithTimeout; NO grpc retry policy in dial options."
      - "services/event-bus/internal/infrastructure/nats.go:193-195 — ce-id header set for tracing, not used as Nats-Msg-Id dedup key."

drift_test:
  - claim: "K8s Production-Ready / Postgres-backed horizontal scale (M6, v0.5.0)."
    result: "PARTIAL"
    evidence:
      - "VERIFIED scale-OUT plumbing exists: HPA+PDB+resources on every chart; prod overrides set replicas 2-3 + HPA + wire Postgres DSN (values-production.yaml across all 7 charts; task-broker prod db.secretName:'task-broker-db')."
      - "CONTRADICTED 'production-ready' for data/event durability: Postgres is a single-instance StatefulSet with HA explicitly 'WON'T' (postgres/statefulset.yaml:1-5; ADR-026:160-163); NATS JetStream streams are Replicas:1 single-node (nats.go:144) — two acknowledged SPOFs."
      - "PARTIAL default-deploy safety: umbrella DEFAULT (values.yaml:46-48,55-57) leaves task-broker/agent-registry DSN empty → in-memory repos → multi-replica unsafe by default; only the prod override file fixes it."
      - "SELF-RATED: docs/product/strategy.md:449 rates Scalability 4.5/10 — consistent with this finding, NOT with a bare 'production-ready' reading."
  - claim: "Stateless workflow-compiler (Part-1 C7, claimed refactored M6)."
    result: "VERIFIED"
    evidence:
      - "services/workflow-compiler/internal/api/server.go:21-24,46 — compiler is stateless, IR not stored, GetCompiledWorkflow returns NOT_FOUND. (Resolves 5.1's note that the proto comment at workflow_compiler.proto:50-53 is STALE — code is already stateless.)"
  - claim: "Event bus has durable consumers with replay & backpressure (ADR-022)."
    result: "PARTIAL"
    evidence:
      - "VERIFIED durable semantics: nats.Durable + AckExplicit + MaxDeliver(5) + BackOff + DLQ (nats.go:356-366,328-350)."
      - "WEAKENED: DeliverLast (not replay-from-sequence/time for new consumers); no MaxAckPending/AckWait set (default flow control); Replicas:1 single-node JetStream — durable to disk but NOT fault-tolerant or throughput-replicated."
  - claim: "Namespace provides multi-tenant isolation."
    result: "CONTRADICTED (partial)"
    evidence:
      - "Real only in memory-service (pgvector.go:28 WHERE namespace) & engine-adapter datacontext (datacontext.go:26-34); COSMETIC in event-bus (subject=event.Type, nats.go:188-189) and ABSENT in task-broker/agent-registry. No per-tenant quota → noisy-neighbor. Isolation deferred post-M8 (Part-1 §1.8)."

red_flags:
  - severity: "High"
    finding: "Two acknowledged single-points-of-failure in the data/event tier: Postgres runs as a single-instance StatefulSet (replicas:1, HA 'WON'T', no failover/backups in-chart) and NATS JetStream streams are Replicas:1 on a single un-clustered node. The compute tier scales out (HPA on 5 services) but bottoms out on a non-HA shared Postgres and a non-replicated JetStream. 'Production-ready' is true for stateless compute, not for the stateful substrate every workflow depends on."
    evidence:
      - "helm/charts/postgres/templates/statefulset.yaml:1-5,13"
      - "docs/adr/ADR-026-postgres-distribution.md:160-163,175"
      - "services/event-bus/internal/infrastructure/nats.go:139-145 (Replicas:1)"
      - "helm/charts/nats/values.yaml:32-43 (no cluster override)"
  - severity: "High"
    finding: "Default umbrella deploy is NOT horizontally safe for the two stateful services: task-broker & agent-registry fall back to in-memory mutex-guarded maps when no DB DSN is wired, and the umbrella DEFAULT leaves db.secretName empty. Scaling those to >1 replica without setting the DSN gives each replica a disjoint state view (the exact failure ADR-021 was written to fix). Correct config exists in values-production.yaml but the safe path is opt-in, not default."
    evidence:
      - "helm/zynax-umbrella/values.yaml:46-48,55-57 (db.secretName empty by default)"
      - "services/task-broker/cmd/task-broker/main.go:122-124 (in-memory default branch)"
      - "docs/adr/ADR-021-horizontal-scale.md:18-24 (in-memory blocks horizontal scale: 'each replica holds a disjoint view')"
  - severity: "Medium"
    finding: "Helm/doc drift on event-bus & memory-service: both production charts still carry appVersion:'placeholder' and 'Do not deploy until EPIC I/J merges', and the umbrella defaults them enabled:false — yet the services are fully implemented and e2e-tested (images ship since #1089/#1090). The shipped scalability story (durable bus, memory KV) is gated OFF in the default production chart and mislabelled as not-yet-built."
    evidence:
      - "helm/zynax-event-bus/Chart.yaml (appVersion 'placeholder'; 'Implementation pending EPIC I (#772). Do not deploy')"
      - "helm/zynax-umbrella/values.yaml:59-65 (event-bus & memory-service enabled:false)"
      - "scripts/e2e/values-e2e.yaml:56-83 (both enabled:true with real image tag:main; 'images ship since #1089')"
  - severity: "Medium"
    finding: "No backpressure/queue-aware autoscaling and no connection-pool tuning. HPA scales only on CPU/memory — task-broker fan-out and event-bus publish load are not reflected in CPU, so a deep task/event backlog won't trigger scale-out. pgxpool is created with defaults (no MaxConns), so N autoscaled replicas × default pool can exhaust the single Postgres's connection slots."
    evidence:
      - "helm/zynax-api-gateway/templates/hpa.yaml:16-28 (cpu+memory metrics only)"
      - "services/task-broker/internal/infrastructure/postgres/repository.go:32-33 (pgxpool.New, no Config)"
      - "services/event-bus/internal/infrastructure/nats.go (no MaxAckPending/AckWait set)"
  - severity: "Medium"
    finding: "Multi-tenancy is half-real and there are no per-tenant quotas: namespace isolates memory-service and engine-adapter data-context, but is cosmetic on the event bus (single shared stream per event type) and absent in task-broker/agent-registry. A heavy namespace can starve others on the shared Postgres/JetStream (noisy-neighbor). ADR-022's namespace-isolation + rate-limit chokepoint is 'planned M7', not enforced."
    evidence:
      - "services/event-bus/internal/infrastructure/nats.go:178,188-189"
      - "docs/adr/ADR-022-event-bus-architecture.md:52 (isolation/quota 'planned for M7')"
  - severity: "Low"
    finding: "PDB minAvailable:1 is hardcoded (not value-driven). On a single-replica install (dev/umbrella default replicaCount:1) this blocks ALL voluntary disruptions (node drains/upgrades) for that pod. Harmless at prod replicas≥2 but a footgun for the default topology."
    evidence:
      - "helm/zynax-api-gateway/templates/pdb.yaml:8-9 (minAvailable:1 literal)"
      - "helm/zynax-umbrella/values.yaml:16,25,31 (replicaCount:1 default)"

green_flags:
  - strength: "Genuinely stateless compute services with clean rollout hygiene: workflow-compiler stores no IR (C7 closed), api-gateway/event-bus hold no per-request state, engine-adapter delegates durability+fan-out to Temporal; RollingUpdate maxUnavailable:0 + readiness/liveness/startup probes + graceful NOT_SERVING drain mean zero-downtime horizontal scaling of the stateless tier."
    evidence: ["services/workflow-compiler/internal/api/server.go:21-24", "helm/zynax-api-gateway/templates/deployment.yaml:15-19,102-121", "services/task-broker/cmd/task-broker/main.go:163-164"]
  - strength: "Production-grade per-service Helm scale plumbing: every chart ships HPA (CPU+mem), PDB, requests+limits, and a values-production overlay raising replicas to 2-3, enabling HPA, and wiring Postgres DSN + mTLS secrets — a coherent, repeatable scale-out package across all 7 services."
    evidence: ["helm/zynax-task-broker/values-production.yaml:4-27", "helm/zynax-engine-adapter/values-production.yaml:3-10", "helm/zynax-api-gateway/values-production.yaml:1-22"]
  - strength: "Durable, idempotent failure-isolation primitives where it counts: Temporal activity RetryPolicy with non-retryable types, task-broker in-flight recovery on restart, Postgres upsert + manifest-hash idempotent apply, NATS DLQ with 5-step exponential backoff — restart/transient-fault safety is engineered, not assumed."
    evidence: ["services/engine-adapter/internal/infrastructure/temporal_workflow.go:77-86", "services/task-broker/cmd/task-broker/main.go:150-160", "services/event-bus/internal/infrastructure/nats.go:328-366"]
  - strength: "Clean per-service data ownership (ADR-008/021): dedicated schemas, no shared tables, repository behind a domain port with memory/postgres adapters swappable at startup — the persistence layer is structured to scale per-service once the single-instance Postgres is made HA."
    evidence: ["helm/charts/postgres/Chart.yaml:4-9", "docs/adr/ADR-021-horizontal-scale.md:50-85"]

open_questions:
  - "What is the scaling ceiling today, and where does it fall over first? Inference: the single-instance Postgres connection-slot limit (untuned pgxpool × autoscaled replicas) and the single-node JetStream throughput/disk are the first break points; the stateless tier (api-gateway/compiler) scales well past them. No load test observed to quantify the ceiling."
  - "Is the umbrella's in-memory default for task-broker/agent-registry an intentional 'safe-by-default-single-replica' posture, or an un-tightened default that will silently break a naive `replicas: 3` umbrella scale-up?"
  - "When event-bus/memory-service charts lose their 'placeholder' label, will the umbrella default flip to enabled:true, and will NATS get a clustered (R3) JetStream profile for production durability?"

unknowns:
  - "Live throughput/latency under load — no benchmark or load test in-repo for task-broker fan-out, event-bus publish, or Postgres; all scale claims are design-level inference (E2/E3), not E1 executed proof. (Cross-ref 5.6 Performance.)"
  - "Whether any production reference deployment actually wires HA-external Postgres/NATS (ADR-026 says operators must 'arrange backups/replication separately'); not evidenced in-repo."
  - "Exact pgxpool default MaxConns at runtime and AckWait/MaxAckPending effective values — code sets none; depends on library/SDK defaults, not independently measured."

cross_references:
  - to_agent: "5.6 Performance"
    note: "Same-wave. No in-repo load test to quantify the scaling ceiling; the single-instance Postgres conn-slots and single-node JetStream are the likely first bottlenecks — performance should measure them."
    evidence: ["helm/charts/postgres/templates/statefulset.yaml:13", "services/event-bus/internal/infrastructure/nats.go:144"]
  - to_agent: "5.1 Architecture"
    note: "Confirms 5.1's cross-ref to me: process-per-engine + engine_hint dead field constrains multi-engine horizontal routing; and the workflow_compiler.proto:50-53 in-memory-map comment is STALE — code (server.go:21-24) is already stateless (C7 closed)."
    evidence: ["services/workflow-compiler/internal/api/server.go:21-24", "docs/due-diligence/2026-06-20-dd-wave-a-findings.md:188-190"]
  - to_agent: "5.10 Documentation / 5.13 Governance"
    note: "Helm drift: event-bus & memory-service charts still labelled 'placeholder / do not deploy' and umbrella-disabled though the services ship and are e2e-tested. Doc-vs-implementation reconciliation debt (same class as Part-1 §1.10)."
    evidence: ["helm/zynax-event-bus/Chart.yaml", "helm/zynax-umbrella/values.yaml:59-65", "scripts/e2e/values-e2e.yaml:56-64"]
  - to_agent: "5.2 Security / 5.20 Enterprise"
    note: "Multi-tenant isolation is partial/cosmetic on the event bus and absent in task-broker/agent-registry; no per-tenant quota. Relevant to enterprise tenancy and to ADR-022's deferred M7 authz/rate-limit chokepoint."
    evidence: ["services/event-bus/internal/infrastructure/nats.go:188-189", "docs/adr/ADR-022-event-bus-architecture.md:52"]

recommendations:
  - priority: "P0"
    action: "Re-label the M6 scalability claim precisely: 'stateless compute horizontally scales (HPA); the data/event substrate (Postgres + JetStream) is single-instance, non-HA — production HA deferred (EPIC #1073 WON'T)'. Stop any bare 'production-ready / horizontally scalable' reading that omits the two acknowledged SPOFs."
    rationale: "Compute scale-out is real but the stateful tier is a documented SPOF; this is the delivery-vs-narrative drift class the diligence exists to catch (Part-1 §1.10). strategy.md:449 already self-rates 4.5/10 — align the headline."
  - priority: "P0"
    action: "Make the safe path the default OR guard it: either wire task-broker/agent-registry DB DSN in the umbrella default (and ship a Postgres-backed default), or add a startup refusal/HPA-guard that forbids replicas>1 when running the in-memory repo."
    rationale: "A naive umbrella `replicas: 3` today silently gives each replica a disjoint in-memory state view — the precise bug ADR-021 set out to eliminate, re-introduced via an opt-in default."
  - priority: "P1"
    action: "Adopt an HA Postgres path (CloudNativePG, already named in ADR-026 as the successor) and a clustered JetStream profile (R3 streams + ≥3 NATS nodes) for production values; add connection-pool sizing (pgxpool MaxConns) tied to replica count."
    rationale: "Removes the two SPOFs and the connection-exhaustion failure mode under autoscaling; ADR-026 explicitly leaves the door open to CloudNativePG."
  - priority: "P1"
    action: "Add queue-depth / custom-metric autoscaling for task-broker and event-bus (e.g. KEDA on NATS pending or task backlog) instead of CPU-only HPA."
    rationale: "Fan-out and event backlog don't surface as CPU; CPU-only HPA won't react to the real scaling signal for the async tier."
  - priority: "P2"
    action: "Promote event-bus & memory-service charts out of 'placeholder' status, flip umbrella defaults, and make namespace a real isolation boundary on the bus (per-namespace subject/stream) with per-tenant quotas (the ADR-022 M7 chokepoint)."
    rationale: "Closes the Helm/doc drift and the cosmetic-multi-tenancy / noisy-neighbor gap before any multi-tenant adopter."
```

---

## (b) §6.2 Prose section

## 5.16 Scalability — Score: 5 (High)

**Mission recap:** Assess horizontal/vertical scaling, statefulness, data-layer scale, multi-tenancy, and failure behavior under load and partition; drift-test "production-ready / horizontally scalable."

**Verdict:** Zynax's *stateless compute tier* genuinely scales out — workflow-compiler holds no IR (the C7 limitation is closed), api-gateway and event-bus carry no per-request state, engine-adapter delegates fan-out and durability to Temporal, and every service chart ships HPA + PDB + resource limits with a coherent `values-production` overlay that raises replicas and enables autoscaling. The failure-isolation primitives (Temporal retry policy, task-broker in-flight recovery, idempotent manifest-hash apply, NATS DLQ with backoff) are engineered, not assumed. But the *stateful substrate every workflow depends on is non-HA by design*: Postgres is a single-instance StatefulSet with HA explicitly "WON'T" (EPIC #1073), and NATS JetStream streams are `Replicas:1` on an un-clustered node — two acknowledged SPOFs. Worse, the umbrella **default** leaves task-broker/agent-registry without a DB DSN, so they fall back to in-memory mutex-guarded maps; a naive multi-replica umbrella deploy re-creates the exact disjoint-state bug ADR-021 was written to fix. Multi-tenancy is half-real (enforced in memory-service/engine-adapter, cosmetic on the bus, absent in task-broker/agent-registry) with no per-tenant quotas. The compute-tier excellence and the non-HA data tier net to an adequate-5: the architecture is *designed* to scale but is *single-instance where it counts*, exactly matching the project's own 4.5/10 self-rating.

**Sub-dimension scores:**

| Sub-dimension | Score | Confidence | Evidence |
|---|---|---|---|
| Statelessness / horizontal scalability | 6 | High | `workflow-compiler server.go:21-24`; `task-broker main.go:122-124`; `umbrella values.yaml:46-48,55-57` |
| Helm HPA/PDB/replicas/resources | 7 | High | `api-gateway hpa.yaml:16-28`; `*-values-production.yaml`; `pdb.yaml:8-9` |
| Data-layer scale (Postgres HA/pooling/ownership) | 4 | High | `postgres/statefulset.yaml:1-5,13`; `ADR-026:160-163,175`; `pgxpool.New` untuned |
| Event bus (JetStream durability/replay/backpressure) | 5 | Medium | `nats.go:139-145,356-366,328-350`; `nats/values.yaml:32-43` |
| Multi-tenancy / namespace isolation | 4 | Medium | `nats.go:188-189`; `pgvector.go:28`; `datacontext.go:26-34`; `ADR-022:52` |
| Failure isolation / retries / idempotency / Temporal | 6 | Medium | `temporal_workflow.go:77-86`; `task-broker main.go:150-160`; `apply.go:29-35` |

**Drift test:**
- *"K8s Production-Ready / Postgres-backed horizontal scale (M6)"* → **PARTIAL.** Scale-out plumbing VERIFIED (HPA/PDB/prod overlays wire replicas 2-3 + Postgres); CONTRADICTED for durability (single-instance Postgres, HA "WON'T"; JetStream `Replicas:1`); default deploy unsafe for the two stateful services (in-memory by default). Self-rated 4.5/10 (`strategy.md:449`).
- *"Stateless workflow-compiler (C7)"* → **VERIFIED** (`server.go:21-24,46`; the stale proto comment 5.1 flagged is contradicted by the code, which is already stateless).
- *"Durable consumers with replay & backpressure (ADR-022)"* → **PARTIAL.** Durable+AckExplicit+MaxDeliver+DLQ VERIFIED; but DeliverLast (not replay), no MaxAckPending/AckWait, single-node JetStream.
- *"Namespace = multi-tenant isolation"* → **CONTRADICTED (partial).** Real in memory-service/engine-adapter; cosmetic on the bus; absent in task-broker/agent-registry; isolation deferred post-M8.

**Red flags (severity-ordered):**
1. **High** — Two acknowledged SPOFs: single-instance non-HA Postgres + single-node `Replicas:1` JetStream under a scaling compute tier (`postgres/statefulset.yaml:1-5`, `nats.go:144`, `ADR-026:160-163`).
2. **High** — Default umbrella deploy is not multi-replica-safe: task-broker/agent-registry default to in-memory maps with empty DSN (`umbrella values.yaml:46-48,55-57`, `task-broker main.go:122-124`).
3. **Medium** — Helm/doc drift: event-bus & memory-service still "placeholder / do not deploy" and umbrella-disabled though shipped and e2e-tested (`event-bus Chart.yaml`, `values-e2e.yaml:56-64`).
4. **Medium** — CPU-only HPA + untuned pgxpool: async backlog won't trigger scale-out; autoscaled replicas can exhaust the single Postgres's connection slots (`hpa.yaml:16-28`, `repository.go:32-33`).
5. **Medium** — Cosmetic event-bus multi-tenancy + no per-tenant quota → noisy-neighbor (`nats.go:188-189`, `ADR-022:52`).
6. **Low** — Hardcoded PDB `minAvailable:1` blocks voluntary disruptions on single-replica default topology (`pdb.yaml:8-9`).

**Green flags:**
- Genuinely stateless compute with zero-downtime rollout hygiene (`server.go:21-24`, `deployment.yaml:15-19,102-121`, graceful NOT_SERVING drain).
- Coherent per-service Helm scale package: HPA+PDB+limits + prod overlay across all 7 charts (`task-broker values-production.yaml:4-27`).
- Engineered failure-isolation/idempotency: Temporal RetryPolicy, in-flight recovery, manifest-hash apply, NATS DLQ+backoff (`temporal_workflow.go:77-86`, `main.go:150-160`, `nats.go:328-366`).
- Clean per-service data ownership behind swappable repo ports (`postgres/Chart.yaml:4-9`, `ADR-021:50-85`).

**Open questions / unknowns:** Scaling ceiling unquantified (no load test in-repo) — inferred first break points are single-instance Postgres connection slots and single-node JetStream throughput; whether the in-memory umbrella default is intentional safe-single-replica vs an un-tightened footgun; whether prod actually wires external HA Postgres/NATS (ADR-026 pushes this to the operator); effective pgxpool/AckWait defaults not measured.

**Recommendations:** P0 — re-label the scale claim to separate stateless-compute scale-out from the non-HA data tier, and make the multi-replica-safe (Postgres-backed) path the default or guard replicas>1 on the in-memory repo. P1 — adopt HA Postgres (CloudNativePG, already named in ADR-026) + clustered JetStream (R3) and size pgxpool to replica count; add queue-depth/custom-metric autoscaling for the async tier. P2 — promote event-bus/memory-service out of "placeholder," flip umbrella defaults, and make namespace a real bus isolation boundary with per-tenant quotas.

**Cross-references:** 5.6 Performance (quantify the ceiling: Postgres conn-slots, JetStream throughput); 5.1 Architecture (confirms process-per-engine routing limit; closes the stale stateless-compiler proto comment); 5.10/5.13 (event-bus/memory-service placeholder Helm drift); 5.2/5.20 (partial/cosmetic multi-tenancy, deferred ADR-022 M7 authz/quota chokepoint).
# Agent 5.22 — OpenSSF Readiness — Wave B (derived technical) — Issue #1403

> REPO: the repository root · HEAD `e3135a6` (branch `main`, 2026-06-20)
> READ-ONLY audit. Evidence cited as `path:line` (E2/E3/E4), `cmd→output` (E1), external URL (E7), or marked UNKNOWN / CLAIMED.
> Wave B (derived). Consumes Wave A 5.2 Security & 5.9 DevOps supply-chain packets (cited, not re-scored — framework §3.1).
> Builds the OpenSSF Scorecard view ON TOP of Wave A; primary supply-chain zones (cosign/SBOM/SLSA, digest pinning, gate-blocking) remain owned by 5.2/5.9.

---

## (a) §3.4 Handoff packet

```yaml
agent: "5.22 OpenSSF Readiness"
wave: "B"
dimension_groups: ["D5", "D7"]
overall_score: 6
overall_confidence: "High"

# ── Per-Scorecard-check sub_scores (the 15 checks from §5.22 checklist) ──
# Mapping convention: Scorecard 0-10 estimate folded into the 0-10 DD scale.
# VERIFIED = config/cmd proves it; PARTIAL = present-but-gapped; FAIL = absent/contradicted.
sub_scores:
  - dimension: "Scorecard: Branch-Protection"
    score: 8
    confidence: "High"
    result: "VERIFIED (with one tier-4 gap)"
    justification: "Live ruleset 17547241 enforces required_linear_history, required_signatures, squash-only, 12 strict required status checks, required_review_thread_resolution. Tier-4 gap: required_approving_review_count=0, require_code_owner_review=false."
    evidence:
      - "cmd→`gh api repos/zynax-io/zynax/rulesets/17547241 --jq .rules`→ {type:required_linear_history}, {type:required_signatures}, pull_request{allowed_merge_methods:[squash], required_approving_review_count:0, require_code_owner_review:false, required_review_thread_resolution:true}, required_status_checks{12 contexts, strict_required_status_checks_policy:true}"
      - "docs/adr/ADR-023 (squash-only merge — cross-ref Part 1 §1.9)"
  - dimension: "Scorecard: Signed-Releases"
    score: 8
    confidence: "Medium"
    result: "VERIFIED (git) / PARTIAL (cosign artifact UNKNOWN)"
    justification: "Release git tags are SSH/ED25519-signed and verify locally; cosign keyless image signing + SLSA L2 provenance + SPDX SBOM wired in release.yml (Wave A 5.2/5.9). GHCR signature EXISTENCE not independently verifiable (no cosign/registry access)."
    evidence:
      - "cmd→`git tag -v v0.5.0`→'Good \"git\" signature for ogomezmanresa-at-gmail-dot-com with ED25519 key SHA256:muS51myW5vmlAq...' (E1)"
      - ".github/workflows/release.yml:201  (cosign sign --yes per promoted digest — Wave A 5.2)"
      - ".github/workflows/release.yml:510  (actions/attest-build-provenance — SLSA L2 — Wave A 5.9)"
      - ".github/workflows/release.yml:527  (syft SPDX SBOM per digest — Wave A 5.2, C3 VERIFIED)"
      - "Wave A 5.2: `cosign verify` UNRUNNABLE (cosign absent, no registry) → C4 PARTIAL, GHCR signature existence UNKNOWN"
  - dimension: "Scorecard: Pinned-Dependencies"
    score: 9
    confidence: "High"
    result: "VERIFIED"
    justification: "All external GitHub Actions pinned by 40-char SHA; zero @main/@master/@vN/@latest tag-pins across all 21 workflows. Container images digest-pinned via images.yaml SoT with pre-merge drift gate (Wave A)."
    evidence:
      - "cmd→`grep -rhnE 'uses: .+@(main|master|v[0-9]+|latest)' .github/workflows/*.yml`→ (empty — no tag-pinned external actions)"
      - ".github/workflows/pr-checks.yml:70  (actions/dependency-review-action@a1d282b36b6f3519aa1f3fc636f609c47dddb294 — SHA-pinned example)"
      - ".github/workflows/ci.yml:891  (github/codeql-action/upload-sarif@03e4368ac7daa2bd82b3e85262f3bf87ee112f57 — SHA-pinned)"
      - "images/images.yaml  (sha256 digest SoT — Wave A 5.2/5.9); .github/workflows/pr-checks.yml:336 (images check drift gate)"
      - "renovate.json:135  (github-actions group pinDigests:true — keeps action SHAs current)"
  - dimension: "Scorecard: Token-Permissions"
    score: 9
    confidence: "High"
    result: "VERIFIED"
    justification: "Every one of the 21 workflows declares a top-level permissions: block; the default is least-privilege contents: read. Write scopes are narrow and job-scoped (packages:write for image cleanup; security-events:write only for SARIF upload)."
    evidence:
      - "cmd→`grep -rn '^permissions:' .github/workflows/*.yml`→ 21/21 workflows declare top-level permissions; ci.yml:41 `contents: read`, pr-checks.yml:35 `contents: read`, release.yml:55 `contents: read`"
      - ".github/workflows/ci.yml:802  (security-events: write — narrowly scoped to SARIF-upload job only)"
      - ".github/workflows/pr-image-cleanup.yml:42  (packages: write — scoped to cleanup workflow)"
  - dimension: "Scorecard: Dangerous-Workflow"
    score: 9
    confidence: "High"
    result: "VERIFIED"
    justification: "No pull_request_target anywhere; sole workflow_run (release.yml) is the trusted post-merge retag. Untrusted github.event.pull_request.title is passed via env: and referenced as a quoted shell var (injection-safe pattern), not interpolated into run: directly."
    evidence:
      - "cmd→`grep -rln pull_request_target .github/workflows/`→ (empty)"
      - ".github/workflows/pr-checks.yml:218  (PR_TITLE via env:); :220 `echo \"$PR_TITLE\" | grep` (quoted var — safe)"
      - ".github/workflows/release.yml  (only workflow_run; trusted promote path — Wave A 5.9:1140)"
  - dimension: "Scorecard: SAST"
    score: 5
    confidence: "High"
    result: "PARTIAL"
    justification: "No CodeQL ANALYZE/init (no real CodeQL SAST run) — codeql-action is used ONLY to upload Trivy SARIF. Scorecard SAST credits Trivy SARIF upload + govulncheck/bandit, but the absence of a CodeQL analysis run caps the check. SAST signal is CVE/lint-oriented, not deep code SAST."
    evidence:
      - "cmd→`grep -rn 'codeql-action/analyze|codeql-action/init' .github/workflows/`→ (empty — NO real CodeQL analysis)"
      - ".github/workflows/ci.yml:891  (github/codeql-action/upload-sarif — used to push Trivy SARIF only)"
      - ".github/workflows/ci.yml:689  (govulncheck — Wave A 5.2); ci.yml:716 (bandit + pip-audit — Wave A 5.2)"
      - ".github/workflows/ci.yml:870  (Trivy CRITICAL,HIGH fail; :893 SARIF→Security tab — Wave A 5.2)"
  - dimension: "Scorecard: Dependency-Update-Tool"
    score: 9
    confidence: "High"
    result: "VERIFIED"
    justification: "renovate.json present with config:recommended, custom managers for CI env-var versions, and pinDigests:true for both Docker images and GitHub Actions. 7 renovate[bot] commits in history confirm it runs live."
    evidence:
      - "renovate.json:3  (\"extends\": [\"config:recommended\"]); :104 (docker pinDigests:true); :135 (github-actions pinDigests:true)"
      - "SECURITY.md:38  (\"Renovate dependency updates (weekly, patch auto-merge)\")"
      - "cmd→`git shortlog -sne --all`→ 7 commits by renovate[bot] (E1 — tool runs)"
  - dimension: "Scorecard: Fuzzing"
    score: 4
    confidence: "High"
    result: "PARTIAL"
    justification: "Native Go fuzz harnesses EXIST (FuzzEvalGuard, FuzzParseManifest) with a Makefile fuzz target, but NO CI/scheduled fuzz job and NO OSS-Fuzz integration. Scorecard Fuzzing credits the presence of native go fuzz functions (low-but-nonzero), not continuous fuzzing."
    evidence:
      - "services/engine-adapter/internal/domain/interpreter_test.go:577  (func FuzzEvalGuard)"
      - "services/workflow-compiler/internal/domain/manifest_fuzz_test.go:40  (func FuzzParseManifest)"
      - "Makefile:374  (go test -fuzz target, manual DURATION)"
      - "cmd→`grep -rn 'fuzztime|-fuzz' .github/workflows/`→ (empty — no fuzz job in CI; no OSS-Fuzz)"
  - dimension: "Scorecard: CI-Tests"
    score: 9
    confidence: "High"
    result: "VERIFIED"
    justification: "test-unit AND test-integration are REQUIRED branch-protection status checks; integration runs the real //go:build integration suite via testcontainers; SKIPPED-bypass footgun explicitly defended (if: always())."
    evidence:
      - "cmd→ruleset 17547241 required_status_checks includes 'test-unit' and 'test-integration' (E1)"
      - ".github/workflows/ci.yml:573  (test-unit required-check wrapper); :614 (test-integration testcontainers)"
      - "Wave A 5.9:1132  (if: always() defeats SKIPPED-bypass — #986)"
  - dimension: "Scorecard: Vulnerabilities"
    score: 8
    confidence: "High"
    result: "VERIFIED"
    justification: "Layered blocking vuln gates: govulncheck (Go), pip-audit (Python), Trivy CRITICAL/HIGH=fail, dependency-review blocks new HIGH CVEs. Two documented/time-boxed suppressions (PYSEC-2026-196, trivy DS002) are standing exceptions (Wave A 5.2 Low flag)."
    evidence:
      - ".github/workflows/pr-checks.yml:70  (dependency-review-action@a1d282b — blocks new HIGH CVEs)"
      - ".github/workflows/ci.yml:689  (govulncheck); :870 (Trivy CRITICAL,HIGH=fail) — Wave A 5.2"
      - ".trivyignore  (DS002 accepted-until 2026-11-01); ci.yml:730 (pip-audit --ignore-vuln PYSEC-2026-196) — Wave A 5.2 Low flag"
  - dimension: "Scorecard: Code-Review"
    score: 2
    confidence: "High"
    result: "FAIL"
    justification: "Branch protection sets required_approving_review_count=0 and the last 15 merged PRs show 0 reviews / empty reviewDecision. Single-maintainer self-merge on green CI — commits are NOT human-reviewed. Scorecard Code-Review scores the fraction of reviewed changes ≈ 0."
    evidence:
      - "cmd→`gh pr list --state merged --limit 15 --json reviewDecision,reviews`→ all 15 PRs (#1455–#1471): reviews:0, reviewDecision:'' (E1)"
      - "cmd→ruleset pull_request: required_approving_review_count:0, require_code_owner_review:false (E1)"
      - "Wave A 5.9:1206  ('required_approving_review_count:0 → NO human approval required; automation can self-merge on green')"
  - dimension: "Scorecard: Maintained"
    score: 9
    confidence: "High"
    result: "VERIFIED"
    justification: "824 commits in the last 90 days; HEAD dated 2026-06-20 (today). Highly active. (Caveat: activity is single-human — bus factor 1, Wave A 5.24 — which Scorecard Maintained does not penalize but governance does.)"
    evidence:
      - "cmd→`git log --since='90 days ago' --oneline | wc -l`→ 824 (E1)"
      - "cmd→`git log -1 --format='%ci'`→ 2026-06-20 (E1)"
      - "Wave A 5.24: bus factor = 1 (one human identity) — Maintained-active but socially concentrated"
  - dimension: "Scorecard: Security-Policy"
    score: 9
    confidence: "High"
    result: "VERIFIED"
    justification: "SECURITY.md present with private-disclosure channel (GitHub Security Advisories), explicit 'do NOT open a public issue', and response SLAs (48h ack / 5d assessment / severity-based fix timeline)."
    evidence:
      - "SECURITY.md:14  ('Do NOT open a public GitHub issue for security vulnerabilities')"
      - "SECURITY.md:17  (private GitHub Security Advisories link)"
      - "SECURITY.md:22-25  (Response SLAs: 48h ack, 5d assessment, Critical 7d / High 30d / Medium 90d)"
  - dimension: "Scorecard: License"
    score: 1
    confidence: "High"
    result: "FAIL"
    justification: "NO top-level LICENSE file exists in the repo. README badge and footer LINK to a non-existent LICENSE file. Only ADR-005 (decision record) + 347 SPDX headers declare Apache-2.0. Scorecard License check requires a recognized top-level license file → FAIL. This is also a Part 1 §1.10-class doc-vs-reality drift (broken README link)."
    evidence:
      - "cmd→`ls LICENSE* COPYING*`→ 'No existe el fichero' (no license file at root) (E1)"
      - "cmd→`git ls-files | grep -i license`→ only 'docs/adr/ADR-005-apache-license.md' (no LICENSE file tracked) (E1)"
      - "README.md:13  ([![License: Apache 2.0]...](LICENSE) — badge links to MISSING file)"
      - "README.md:527  ('Apache License 2.0 — see [LICENSE](LICENSE).' — link target absent)"
      - "cmd→`grep -rl 'SPDX-License-Identifier: Apache-2.0' --include=*.go | wc -l`→ 347 (license declared in headers, but no canonical file)"
  - dimension: "Scorecard: CII/Best-Practices"
    score: 3
    confidence: "High"
    result: "FAIL (no badge registered)"
    justification: "No OpenSSF/CII Best Practices badge anywhere in the repo. Supporting docs (CONTRIBUTING, GOVERNANCE, CODE_OF_CONDUCT) exist — a self-certification path is partly walkable — but no badge is registered. MAINTAINERS.md also absent (Wave A 5.24, #494 open)."
    evidence:
      - "cmd→`grep -rln 'bestpractices.coreinfrastructure|bestpractices.dev' . --include=*.md`→ (empty — no badge)"
      - "cmd→`ls CONTRIBUTING.md GOVERNANCE.md CODE_OF_CONDUCT.md MAINTAINERS.md`→ first 3 present; MAINTAINERS.md 'No existe'"
      - "README.md:13,16  (only License + Scorecard badges; NO Best-Practices badge)"
  - dimension: "Scorecard badge drift (displayed vs actual published score)"
    score: 2
    confidence: "High"
    result: "FAIL / CONTRADICTED"
    justification: "README displays an OpenSSF Scorecard badge, but there is NO scorecard.yml workflow to produce a result and the live Scorecard API has NO published record for the project (HTTP 404). The badge renders 'no data', not a real grade — a displayed-vs-actual drift."
    evidence:
      - "README.md:16  ([![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/zynax-io/zynax/badge)]...)"
      - "cmd→`ls .github/workflows/scorecard.yml`→ 'No existe el fichero' (no Scorecard workflow) (E1)"
      - "cmd→`curl https://api.securityscorecards.dev/projects/github.com/zynax-io/zynax`→ HTTP 404, empty body (no published Scorecard) (E1/E7)"
      - "cmd→badge redirect→ https://img.shields.io/ossf-scorecard/... (renders no-data without a scan)"

drift_test:
  - claim: "README displays an OpenSSF Scorecard badge reflecting an actual Scorecard grade (Part 1 §1.9: 'OpenSSF Scorecard badge')"
    result: "CONTRADICTED"
    evidence:
      - "README.md:16 (badge present)"
      - "cmd→`curl .../projects/github.com/zynax-io/zynax`→ HTTP 404 empty (NO published Scorecard) (E1)"
      - "cmd→`ls .github/workflows/scorecard.yml`→ absent (no workflow generates a score) (E1)"
  - claim: "License: Apache-2.0 (`LICENSE`) — Part 1 §1.9 / ADR-005; README links to LICENSE"
    result: "CONTRADICTED"
    evidence:
      - "cmd→`ls LICENSE*`→ absent; `git ls-files|grep -i license`→ only ADR-005 (E1)"
      - "README.md:13,527 (badge + footer link to non-existent LICENSE file)"
  - claim: "GitHub Actions are pinned by SHA + workflow tokens are least-privilege (supply-chain hygiene, §5.22)"
    result: "VERIFIED"
    evidence:
      - "cmd→`grep -rhnE 'uses: .+@(main|master|v[0-9]+|latest)' .github/workflows/*.yml`→ empty (all SHA-pinned)"
      - "cmd→`grep -rn '^permissions:' .github/workflows/*.yml`→ 21/21 declare top-level perms; default contents: read"

red_flags:
  - severity: "High"
    finding: "Scorecard BADGE is decorative, not real: README.md:16 shows an OpenSSF Scorecard badge but there is NO scorecard.yml workflow and the live Scorecard API returns HTTP 404 (no published score). An evaluator clicking the badge gets 'no data', not a grade — a Part 1 §1.10-class displayed-vs-actual drift that undermines the project's own CNCF-credibility narrative."
    evidence:
      - "README.md:16"
      - "cmd→`curl .../projects/github.com/zynax-io/zynax`→404 empty"
      - "cmd→`ls .github/workflows/scorecard.yml`→absent"
  - severity: "High"
    finding: "License check FAILS at the simplest bar: no top-level LICENSE file exists, yet README.md:13 (badge) and :527 (footer) LINK to LICENSE. Apache-2.0 is only declared in ADR-005 + 347 SPDX headers. Scorecard License = FAIL, GitHub repo-license auto-detect fails, and the README link is broken — trivially fixable but currently a verified contradiction."
    evidence:
      - "cmd→`ls LICENSE*`→absent"
      - "cmd→`git ls-files|grep -i license`→only ADR-005"
      - "README.md:13"
      - "README.md:527"
  - severity: "Medium"
    finding: "Code-Review check ≈ 0: required_approving_review_count=0 and the last 15 merged PRs (#1455–#1471) all show 0 reviews. Every change self-merges on green CI with no second human. Scorecard Code-Review scores near-zero; also a governance/bus-factor red flag (echoes Wave A 5.9/5.24)."
    evidence:
      - "cmd→`gh pr list --state merged --limit 15 --json reviews,reviewDecision`→all reviews:0"
      - "cmd→ruleset required_approving_review_count:0"
  - severity: "Medium"
    finding: "SAST check is shallow: no CodeQL analysis run (codeql-action used only to upload Trivy SARIF). Scorecard SAST is capped without a real CodeQL/SonarCloud analyze step; current coverage is CVE/lint-grade (govulncheck/bandit/trivy), not code-flow SAST."
    evidence:
      - "cmd→`grep -rn 'codeql-action/analyze|codeql-action/init' .github/workflows/`→empty"
      - ".github/workflows/ci.yml:891 (upload-sarif only)"
  - severity: "Low"
    finding: "Fuzzing is harness-only, not continuous: native go fuzz funcs exist (FuzzEvalGuard, FuzzParseManifest) + a manual Makefile target, but no CI/scheduled fuzz job and no OSS-Fuzz. Scorecard credits presence but not coverage."
    evidence:
      - "services/workflow-compiler/internal/domain/manifest_fuzz_test.go:40"
      - "cmd→`grep -rn 'fuzztime|-fuzz' .github/workflows/`→empty"
  - severity: "Low"
    finding: "SECURITY.md:'Multi-arch container images (linux/amd64 + linux/arm64) for all service images (#489)' is CONTRADICTED by Wave A 5.9: production service images are amd64-only (ci.yml:861). A standing doc-vs-config drift in the security policy itself (cross-ref 5.9; not re-scored here)."
    evidence:
      - "SECURITY.md (Multi-arch line)"
      - "Wave A 5.9: ci.yml:861 platforms: linux/amd64 ONLY"

green_flags:
  - strength: "Pinned-Dependencies is best-in-class: ALL external GitHub Actions SHA-pinned (zero tag-pins), container images digest-pinned via images.yaml SoT with a pre-merge drift gate, and renovate pinDigests:true keeps both current."
    evidence:
      - "cmd→`grep -rhnE 'uses: .+@(main|master|v[0-9]+|latest)' .github/workflows/*.yml`→empty"
      - ".github/workflows/pr-checks.yml:70 (SHA-pinned dependency-review)"
      - "renovate.json:104,135 (pinDigests)"
  - strength: "Token-Permissions is exemplary: 21/21 workflows declare top-level permissions; least-privilege contents: read default; write scopes narrowly job-scoped. This is the single highest-confidence Scorecard win."
    evidence:
      - "cmd→`grep -rn '^permissions:' .github/workflows/*.yml`→21/21"
      - ".github/workflows/ci.yml:41 (contents: read)"
  - strength: "Dangerous-Workflow clean: no pull_request_target, sole workflow_run is the trusted retag, untrusted inputs flow through env: with quoted-var references (injection-safe)."
    evidence:
      - "cmd→`grep -rln pull_request_target .github/workflows/`→empty"
      - ".github/workflows/pr-checks.yml:218 (env: PR_TITLE pattern)"
  - strength: "Signed-Releases strong on the git side: release tags are ED25519-signed and verify locally; cosign keyless image signing + SLSA L2 + SBOM wired in release.yml (Wave A)."
    evidence:
      - "cmd→`git tag -v v0.5.0`→Good git signature ED25519"
      - ".github/workflows/release.yml:201,510,527 (Wave A 5.2/5.9)"
  - strength: "Branch-Protection, CI-Tests, Vulnerabilities, Maintained, Security-Policy, Dependency-Update-Tool all VERIFIED via live ruleset + config — a solid Scorecard backbone of ~6 strong checks."
    evidence:
      - "cmd→ruleset 17547241 (required_signatures, linear history, 12 strict checks)"
      - "SECURITY.md:14-25; renovate.json:3; cmd→824 commits/90d"

open_questions:
  - "What is the REALISTIC Scorecard score? Estimate ~6.0–6.8/10: strong Token-Permissions(~10), Pinned-Deps(~9-10), Dangerous-Workflow(~10), Branch-Protection(~7-8), CI-Tests(10), Vulnerabilities(~9-10), Dependency-Update-Tool(10), Maintained(10), Security-Policy(~9-10), Signed-Releases(~8-10), dragged down by License(0), Code-Review(~0-2), SAST(~5-7 SARIF-only, no CodeQL), Fuzzing(~1-2), CII-Best-Practices(0)."
  - "Top-3 fixes to raise the score: (1) ADD a top-level LICENSE file (Apache-2.0) — flips License 0→10 and fixes the broken README link; (2) add a scorecard.yml workflow so the badge reflects a REAL published score (removes the badge drift); (3) require ≥1 review (or CODEOWNERS review) so Code-Review rises off the floor."
  - "Does GitHub auto-detect the repo license without a LICENSE file? Almost certainly NO (ADR + SPDX headers are not a license file) — confirm in the GitHub UI 'About' panel."

unknowns:
  - "Exact live Scorecard numeric grade — no published record (API 404) and no scorecard.yml to generate one; estimate above is derived per-check, not an official run."
  - "cosign/SLSA signature EXISTENCE on published GHCR images — inherited UNKNOWN from Wave A 5.2/5.9 (cosign absent, no registry access). Signing is config-VERIFIED, artifact-UNKNOWN."
  - "Whether a CII/OpenSSF Best Practices self-certification was ever started off-repo (no badge in-repo to confirm)."

cross_references:
  - to_agent: "5.2 Security"
    note: "Consumed (not re-scored) the supply-chain trifecta cosign+SBOM+SLSA (release.yml:201,510,527), digest pinning + drift gate, and CVE/SAST/secret gates. Cosign GHCR artifact existence remains UNKNOWN per their packet."
    evidence: [".github/workflows/release.yml:201", ".github/workflows/pr-checks.yml:336"]
  - to_agent: "5.9 DevOps"
    note: "Consumed branch-protection ruleset, build-once/promote-by-retag, required_approving_review_count:0, and the amd64-only multi-arch CONTRADICTION (which contradicts SECURITY.md). Did not re-score D8."
    evidence: ["ruleset 17547241", "Wave A 5.9 ci.yml:861"]
  - to_agent: "5.21 CNCF Readiness"
    note: "This packet FEEDS 5.21. OpenSSF posture is technically strong (pinning/tokens/signing) but two simple checks FAIL (License file, real Scorecard badge) and Code-Review≈0 — exactly the kind of supply-chain/governance hygiene a CNCF Sandbox review and external security audit (M8 gate) will flag."
    evidence: ["cmd→`ls LICENSE*`→absent", "cmd→Scorecard API 404", "cmd→PRs 0 reviews"]
  - to_agent: "5.24 Repo Health / Governance"
    note: "Code-Review=0, no MAINTAINERS.md, single-human bus factor reinforce their bus-factor=1 finding; Maintained-active masks social concentration."
    evidence: ["cmd→merged PRs 0 reviews", "cmd→`ls MAINTAINERS.md`→absent"]

recommendations:
  - priority: "P0"
    action: "Add a top-level LICENSE file containing the full Apache-2.0 text. Fixes Scorecard License (0→10), repairs the broken README.md:13/:527 link, and enables GitHub license auto-detection."
    rationale: "Lowest-effort, highest-leverage fix; a missing LICENSE is an embarrassing FAIL for a CNCF-aspiring Apache-2.0 project and a §1.10-class drift."
  - priority: "P0"
    action: "Add a scorecard.yml workflow (ossf/scorecard-action) so the README badge reflects a REAL published score; either that or remove the badge until a score exists."
    rationale: "Eliminates the decorative-badge drift; converts a marketing claim into a verifiable, continuously-monitored artifact."
  - priority: "P1"
    action: "Raise Code-Review off the floor: require ≥1 approving review (or enable CODEOWNERS review) on main, or add a documented second bypass actor. Adds a CodeQL analyze step for real SAST."
    rationale: "Code-Review≈0 and SAST-SARIF-only are the two largest remaining Scorecard depressors and a governance liability at the M8 CNCF gate."
  - priority: "P2"
    action: "Add a scheduled CI fuzz job (go test -fuzz with a fuzztime budget) over the existing FuzzEvalGuard/FuzzParseManifest harnesses, and consider OSS-Fuzz onboarding. Register an OpenSSF Best Practices badge via self-certification (docs already exist)."
    rationale: "Converts harness-only fuzzing and a missing Best-Practices badge into Scorecard credit; both are incremental over existing assets."
```

---

## (b) §6.2 Prose section

## OpenSSF Readiness — Score: 6 (High)

**Mission recap:** Score Zynax against the 15 OpenSSF Scorecard checks and the Best-Practices badge, verify supply-chain integrity, and drift-test the displayed Scorecard/badge level against actual config — building on Wave A's Security (5.2) and DevOps (5.9) supply-chain ground truth.

**Verdict:** Zynax has a genuinely strong *supply-chain-hygiene* core — best-in-class **Pinned-Dependencies** (all actions SHA-pinned, images digest-pinned with a drift gate), exemplary **Token-Permissions** (21/21 workflows least-privilege), a clean **Dangerous-Workflow** posture, signed releases, blocking vuln gates, an active maintenance cadence, and a real Security-Policy. That backbone alone would earn a respectable Scorecard. But the headline result is undermined by two trivially-fixable FAILs and one governance FAIL: there is **no LICENSE file** (the README badge and footer link to a non-existent file; Apache-2.0 lives only in ADR-005 + SPDX headers), the **README Scorecard badge is decorative** (no `scorecard.yml`, live API returns HTTP 404 — no published score), and **Code-Review ≈ 0** (required approvals = 0; the last 15 merged PRs had zero reviews — single-maintainer self-merge). SAST is SARIF-only (no CodeQL analyze) and fuzzing is harness-only (no CI fuzz job). Net: a technically-credible supply chain wrapped in two embarrassing presentation/hygiene gaps that an external audit (M8) will flag first. Estimated realistic Scorecard ≈ **6.0–6.8/10**.

**Sub-dimension scores:** see the 16-row per-Scorecard-check table in the YAML packet above (15 checks + the badge-drift row). Strongest: Token-Permissions (9), Pinned-Dependencies (9), Dangerous-Workflow (9), Dependency-Update-Tool (9), Maintained (9), Security-Policy (9). Weakest: License (1), Code-Review (2), badge-drift (2), Fuzzing (4), SAST (5).

**Drift test (mandatory):**
- *"README shows an OpenSSF Scorecard badge reflecting a real grade"* (Part 1 §1.9) → **CONTRADICTED** — no `scorecard.yml`, Scorecard API HTTP 404, badge renders "no data" (README.md:16; `cmd→curl→404`).
- *"License: Apache-2.0 (`LICENSE`)"* (Part 1 §1.9 / ADR-005; README links LICENSE) → **CONTRADICTED** — no LICENSE file exists; README.md:13/:527 link a missing target (`cmd→ls LICENSE*→absent`).
- *"Actions SHA-pinned + tokens least-privilege"* (§5.22) → **VERIFIED** — zero tag-pinned actions; 21/21 workflows least-privilege.

**Red flags (severity-ordered):**
1. **High** — Scorecard badge is decorative, not backed by any published score (README.md:16; API 404; no workflow).
2. **High** — License check FAILS: no LICENSE file, README links broken (README.md:13/:527; `ls LICENSE*`→absent).
3. **Medium** — Code-Review ≈ 0: 0 required approvals, last 15 PRs unreviewed (ruleset; `gh pr list`).
4. **Medium** — SAST shallow: no CodeQL analyze, SARIF-upload only (ci.yml:891; analyze grep empty).
5. **Low** — Fuzzing harness-only, no CI job (manifest_fuzz_test.go:40; fuzz grep empty).
6. **Low** — SECURITY.md multi-arch claim contradicted by amd64-only services (cross-ref Wave A 5.9).

**Green flags:** Pinned-Dependencies best-in-class; Token-Permissions exemplary (21/21); Dangerous-Workflow clean; signed (ED25519) release tags + cosign/SLSA/SBOM config; a 6-check VERIFIED backbone (Branch-Protection, CI-Tests, Vulnerabilities, Maintained, Security-Policy, Dependency-Update-Tool).

**Open questions / unknowns:** Realistic Scorecard grade (no published run, estimated per-check); cosign/SLSA GHCR artifact existence (inherited UNKNOWN from Wave A — cosign/registry unavailable); whether any off-repo Best-Practices self-cert exists.

**Recommendations:** **P0** add a top-level LICENSE file (Apache-2.0) and a `scorecard.yml` workflow (or remove the badge); **P1** require ≥1 review and add a CodeQL analyze step; **P2** add a scheduled CI fuzz job over existing harnesses and register an OpenSSF Best-Practices badge.

**Cross-references:** Consumes 5.2 Security (cosign/SBOM/SLSA, digest pinning, CVE/SAST gates) and 5.9 DevOps (branch-protection ruleset, build-once/promote, review-count=0, amd64-only contradiction) without re-scoring them; **feeds 5.21 CNCF Readiness** (the License/badge/Code-Review FAILs are exactly the supply-chain/governance hygiene a Sandbox review + M8 external audit will surface); reinforces 5.24 Repo-Health bus-factor=1.
<!-- SPDX-License-Identifier: Apache-2.0 -->
<!-- Agent 5.26 — Innovation · Wave B (derived technical) · GitHub issue #1403 -->
<!-- READ-ONLY audit of the repository root at HEAD (main @ e3135a6). -->
<!-- Every claim carries path:line / command-output, or is marked UNKNOWN. External prior art = E7 (cited). Roadmap/marketing = CLAIMED. -->
<!-- Consumes Wave A (docs/due-diligence/2026-06-20-dd-wave-a-findings.md): Architecture (5.1) and AI-Workflow (5.12) packets cited; their zones NOT re-scored (§3.1). -->

# Agent 5.26 — Innovation / Technical & Process IP

## (a) §3.4 Handoff packet

```yaml
agent: "5.26 Innovation"
wave: "B"
dimension_groups: ["D16", "D10"]   # D16 innovation/IP defensibility; D10 AI-native process IP (cross-ref 5.12)
overall_score: 6
overall_confidence: "Medium"

sub_scores:
  - dimension: "Engine-agnostic Workflow IR — novelty vs prior art"
    score: 7
    confidence: "High"
    justification: "The IR itself is a clean, genuinely engine-neutral state-machine contract embodied in proto + a pure-Go interpreter with zero engine types — defensible and real. But an engine-agnostic workflow IR is established prior art (Argo Workflows IR, Temporal's internal command model, Apache Beam's portable runner IR, Dapr Workflow); the novelty is the *application* to multi-engine AI capability dispatch, not the IR concept. Embodied in code, so not rhetorical — but commoditizable."
    evidence:
      - "protos/zynax/v1/workflow_compiler.proto:205-241 — WorkflowIR(workflow_id/states/initial_state/ir_version); no Temporal/Argo type in the contract"
      - "protos/zynax/v1/workflow_compiler.proto:120-135 — StateType enum {NORMAL,TERMINAL,HUMAN_IN_THE_LOOP} — engine-neutral, no engine binding"
      - "services/engine-adapter/internal/domain/interpreter.go:46-101 — IRInterpreter.Run is a plain Go struct/loop; side effects only via ActivityExecutor/EventPublisher ports (no Temporal SDK import in domain)"
      - "docs/adr/ADR-012-workflow-ir.md:6-7,20-23 — 'engine-agnostic IR … no Temporal/Argo/LangGraph concepts in IR types'"
      - "E7 prior art: Apache Beam portable runner IR; Argo Workflows is itself a YAML→workflow IR; Temporal compiles to an internal command model; Dapr Workflow abstracts a durable engine — engine-portable workflow IR is not category-new."
  - dimension: "Multi-engine portability — operational embodiment (the boldest claim)"
    score: 4
    confidence: "High"
    justification: "The portability moat is real at the CONTRACT/submission boundary (same YAML→same IR→same 5-method WorkflowEngine port; Temporal+Argo both satisfy it with compile-time assertions) but NOT at the EXECUTION boundary — only Temporal interprets the IR; the Argo path serialises IR to a cluster stub that asserts non-empty and exits 0. Cross-ref 5.1 (this is their primary finding; not re-scored as architecture, scored here as the durability/embodiment of the innovation claim)."
    evidence:
      - "5.1 (Wave A) docs/due-diligence/2026-06-20-dd-wave-a-findings.md:168-176,238-244 — Argo 'interpreter' is a stub; argo_engine.go:62-98 never calls IRInterpreter.Run; capability-dispatch parity 'deliberately out of scope'"
      - "services/engine-adapter/internal/domain/interpreter.go:46-101 — only this Temporal-wired interpreter actually traverses states + dispatches"
      - "docs/adr/ADR-015-pluggable-workflow-engines.md:6-7,19-25 — 'Zynax does NOT build a workflow engine … engine selected via config, no code change required' (real at interface)"
      - "docs/adr/ADR-037-zero-temporal-evaluation-engine.md:26-30 — confirms IRInterpreter is engine-agnostic + I/O-free (the moat's true asset), but ADR-037 itself is Status: Rejected (superseded by #1456)"
  - dimension: "Event-driven state-machine model (ADR-014, not-DAGs) — novelty + embodiment"
    score: 6
    confidence: "High"
    justification: "Choosing event-driven state machines over DAGs IS a meaningful, well-argued differentiator for agentic loops/HITL/long-running — and it is embodied: TransitionIR has no acyclic constraint (target_state may reference any/prior state), the interpreter loops unboundedly, HITL is a first-class state type. But the model is explicitly inspired by XState / AWS Step Functions / Temporal (ADR's own 'Inspiration' section); it is a sound architecture choice, not a novel mechanism. The differentiation is framing (vs LangGraph/Airflow DAGs), not invention."
    evidence:
      - "docs/adr/ADR-014-event-driven-state-machine.md:5-21 — 'workflows are event-driven state machines … NOT DAGs'; loops/HITL/long-running rationale"
      - "docs/adr/ADR-014-event-driven-state-machine.md:22-25 — 'Inspiration: XState, AWS Step Functions, Temporal' (self-declared prior art — E5/E7)"
      - "protos/zynax/v1/workflow_compiler.proto:171-185,196-203 — TransitionIR.target_state is any state name; no acyclic check → loops expressible (only ORPHAN_STATE/UNKNOWN_STATE_REFERENCE validated, proto:79-86)"
      - "services/engine-adapter/internal/domain/interpreter.go:67-100 — for{} loop follows ec.CurrentState=transition.GetTargetState() with no cycle guard → back-edges run"
      - "protos/zynax/v1/workflow_compiler.proto:132-134 — STATE_TYPE_HUMAN_IN_THE_LOOP is a first-class state type (HITL embodied, not bolted on)"
      - "E7: AWS Step Functions (ASL) and XState are exactly state-machine workflow engines; LangGraph itself supports cycles (5.1 open_q notes LangGraph adapter is provider-only)."
  - dimension: "Adapter-first no-SDK capability routing (ADR-013) — novelty + embodiment"
    score: 7
    confidence: "High"
    justification: "Strongly embodied: a single 2-RPC AgentService gRPC contract is THE capability boundary, and 5 real adapters (Go: llm/ci/git/http; Python: langgraph) implement the servicer directly with NO SDK import — even the LangGraph (framework) adapter does not import zynax_sdk. The SDK is genuinely optional. But 'any-gRPC-service-is-a-plugin' is well-trodden prior art (gRPC reflection ecosystems, Envoy ext-authz/ext-proc, Dapr's pluggable-component gRPC contract, Backstage). Novelty is in disciplined minimalism (one streaming RPC + schema introspection), not a new mechanism."
    evidence:
      - "protos/zynax/v1/agent.proto:31-47 — AgentService = 2 RPCs only (ExecuteCapability stream + GetCapabilitySchema); contract IS the SDK"
      - "protos/zynax/v1/agent.proto:5-7 — 'Any system that serves this single RPC becomes a first-class capability: no SDK required, no framework required, no language requirement.'"
      - "cmd: grep -rln AgentServiceServicer|AgentServiceServer|RegisterAgentServiceServer agents/adapters services → 5 adapters implement it (llm,ci,git,http Go; langgraph Py) + task-broker"
      - "cmd: grep -rln zynax_sdk agents/adapters/langgraph → (empty) — even the LangGraph adapter does NOT import the SDK; no-SDK holds for the framework case"
      - "docs/adr/ADR-013-adapter-first-no-sdk.md:20-23 — 'The SDK is OPTIONAL … never required. Zero features require SDK adoption.'"
      - "E7 prior art: Dapr pluggable components (gRPC), Envoy ext-authz/ext-proc, OpenFaaS/Knative function contracts — 'implement-a-gRPC-contract-to-extend' is an established pattern."
  - dimension: "SPDD AI-native development methodology — process IP (novelty + transferability)"
    score: 6
    confidence: "Medium"
    justification: "SPDD (Canvas-before-code + REASONS structure + 3-tier KB security + closed learnings loop + expert-prompt substrate) is the single strongest novelty candidate AND partly embodied in executable tooling (a real Go canvas validator, gitleaks-ai-context, CI freshness gate). It is a coherent, transferable methodology — more than ceremony. But (a) it is process/discipline IP, inherently copyable; (b) Wave A 5.12 shows the headline gate is SOFT (any-canvas-passes, Draft-only-warns) so the enforcement is thinner than the ADR advertises; (c) the productivity multiplier is CLAIMED (E5), never measured (no DORA baseline). Cross-ref 5.12 — not re-scoring their enforcement zone; scoring its value AS innovation/IP."
    evidence:
      - "docs/adr/ADR-019-spdd-prompt-governance.md:29-69 — Canvas-before-code core rule + REASONS structure + Draft→Aligned→Synced lifecycle + Tier-1/2/3 context security"
      - "cmd/zynax-ci/validate/canvas.go:36-56,103-170 — SPDD is embodied in executable Go (validates 7 REASONS sections, status enum, security marker, no committed private file) — process is partly code, not only docs"
      - "5.12 (Wave A) docs/due-diligence/2026-06-20-dd-wave-a-findings.md:1607-1613,1626-1631 — drift_test: 'Canvas-before-code enforced gate' = PARTIAL; soft gate (any-canvas-passes; Draft warns only)"
      - "5.12 (Wave A) docs/due-diligence/2026-06-20-dd-wave-a-findings.md:1668-1670 — productivity-multiplier magnitude = CLAIMED (E5), no committed DORA baseline"
      - "docs/ai-learnings/APPLY_LOG.md — closed learnings loop (cited by 5.12:1650-1653: Draft→applied/rejected, rejects structural-workaround patterns)"
      - "E7 prior art: GitHub spec-kit / spec-driven-development, AWS Kiro, Anthropic/industry 'spec-first' AI-coding patterns are converging on the same idea concurrently → first-mover, not sole-inventor; commoditizing fast."
  - dimension: "Innovation-vs-execution split (is the value in the idea or the disciplined build?)"
    score: 7
    confidence: "High"
    justification: "The value is overwhelmingly in the DISCIPLINED BUILD, not in any single novel idea. Each candidate innovation re-frames established prior art (IR/state-machine/gRPC-plugin/spec-first-AI); none is mechanism-novel. What is genuinely strong and harder to replicate is the *integrated execution*: a coherent IR→port→adapter→dispatch substrate with compile-time conformance, plus an AI-native delivery process embodied in tooling. That is real engineering capital, but it is execution capital, not protectable IP."
    evidence:
      - "5.1 (Wave A) docs/due-diligence/2026-06-20-dd-wave-a-findings.md:42-43,265-275 — 'Genuinely engine-neutral IR … textbook WorkflowEngine port' (execution quality is the asset)"
      - "5.5 (Wave A) docs/due-diligence/2026-06-20-dd-wave-a-findings.md:45 — 14-linter config passes 0 issues; disciplined build is the verified strength"
      - "docs/adr/ADR-014-event-driven-state-machine.md:22-25 + docs/adr/ADR-015-pluggable-workflow-engines.md:9-17 — ADRs themselves cite the prior art they build on (Temporal/Argo/XState/Step Functions) → reframing, not invention"
      - "docs/adr/ADR-033-expert-agent-substrate.md:3,28 — expert substrate Status: Proposed; full 14-expert library deferred to M-dx → process IP not yet fully built"

drift_test:
  - claim: "Engine-agnostic — the same workflow runs on Temporal OR Argo without a rewrite (the boldest 'novel moat')."
    result: "PARTIAL"
    evidence:
      - "VERIFIED at the IR/contract/port boundary: one WorkflowIR with zero engine types (workflow_compiler.proto:205-241); a 5-method WorkflowEngine port both engines satisfy with compile-time assertions (5.1:159-167)."
      - "CONTRADICTED at the execution boundary: only Temporal interprets the IR; the Argo leg runs a stub that asserts non-empty IR and exits 0 — capability-dispatch parity 'deliberately out of scope' (5.1 docs/due-diligence/2026-06-20-dd-wave-a-findings.md:168-176; argo_engine.go:62-98 never calls IRInterpreter.Run)."
      - "Net: the innovation is real as an interface/portability *design*, not yet as a *proven, running* multi-engine capability — exactly the delivery-vs-narrative drift class (Part 1 §1.10)."
  - claim: "Event-driven state machines are a novel mechanism that DAG engines cannot match."
    result: "PARTIAL"
    evidence:
      - "VERIFIED embodiment: loops + HITL are first-class and run (interpreter.go:67-100 unbounded transition loop; STATE_TYPE_HUMAN_IN_THE_LOOP proto:132-134)."
      - "CONTRADICTED as novelty: ADR-014:22-25 self-cites XState / AWS Step Functions / Temporal as inspiration; ASL and XState are pre-existing state-machine workflow engines; LangGraph supports cycles. It is a sound, differentiating *choice*, not a new mechanism (E7)."
  - claim: "Adapter-first 'no SDK required' is a real, embodied capability boundary (not marketing)."
    result: "VERIFIED"
    evidence:
      - "AgentService is a 2-RPC contract that IS the plugin boundary (agent.proto:31-47); 5 adapters implement the servicer directly (grep AgentServiceServicer → llm/ci/git/http/langgraph)."
      - "Even the LangGraph framework adapter imports no SDK (grep zynax_sdk agents/adapters/langgraph → empty); ADR-013:20-23 'SDK is OPTIONAL … zero features require it'. Embodied, not rhetorical."
  - claim: "SPDD is a transferable methodology embodied in tooling, not bespoke ceremony."
    result: "PARTIAL"
    evidence:
      - "VERIFIED partial embodiment: real Go validator (cmd/zynax-ci/validate/canvas.go:36-170) + gitleaks-ai-context + CI freshness gate + closed learnings loop (5.12:1650-1656)."
      - "WEAKENED: the headline canvas-before-code gate is SOFT (any-canvas-passes, Draft-only-warns — 5.12:1626-1631); the productivity claim is CLAIMED/E5, never measured (5.12:1668-1670); concurrent industry prior art (spec-kit, Kiro) is commoditizing the idea (E7)."

red_flags:
  - severity: "High"
    finding: "The boldest 'novel moat' — engine-agnostic portability — is innovation at the INTERFACE but is NOT operationally embodied: only Temporal interprets the IR; the second engine (Argo) is a non-interpreting stub. An innovation that is structurally wired but not functionally proven is exactly the 'novelty that is easily replicated / not embodied in execution' red flag. The defensible moat is 'a clean IR + port', which a competent team can rebuild in << 6 months (see Question)."
    evidence:
      - "5.1 (Wave A) docs/due-diligence/2026-06-20-dd-wave-a-findings.md:238-244 — Argo path never calls IRInterpreter.Run; parity 'deliberately out of scope'"
      - "services/engine-adapter/internal/infrastructure/argo_engine.go:62-98 (cited by 5.1)"
      - "docs/adr/ADR-037-...:3 — even the lightweight in-process engine ADR is Status: Rejected/superseded → the third-engine story is still in flux"
  - severity: "Medium"
    finding: "Every candidate technical innovation is a REFRAMING of well-established prior art (engine-portable IR: Beam/Argo/Dapr; event-driven state machine: XState/Step Functions/Temporal — self-cited; gRPC-contract plugin: Dapr/Envoy/Knative). None is mechanism-novel. The patentable/durable-IP surface is thin; the protection is execution quality and integration, which is copyable, not a moat."
    evidence:
      - "docs/adr/ADR-014-event-driven-state-machine.md:22-25 (self-cited inspiration); docs/adr/ADR-015-...:9-17; docs/adr/ADR-012-...:6-7"
      - "E7: Dapr Workflow + pluggable components; Apache Beam portable runner; AWS Step Functions ASL; Envoy ext-proc"
  - severity: "Medium"
    finding: "SPDD (the strongest process-IP candidate) is being commoditized in real time by convergent industry efforts (GitHub spec-kit, AWS Kiro, spec-driven-development), and Zynax's own enforcement is softer than advertised (soft canvas gate) with the value-prop unmeasured (no DORA baseline). Time-to-commoditize is short; the durable edge is the accumulated learnings corpus, which is single-maintainer-authored (bus factor 1)."
    evidence:
      - "5.12 (Wave A) docs/due-diligence/2026-06-20-dd-wave-a-findings.md:1626-1631 (soft gate), 1668-1670 (multiplier CLAIMED not measured), 1646-1648 (single-maintainer IP; ADR-033 Proposed)"
      - "E7: github/spec-kit, AWS Kiro spec-driven IDE, broad 2025-26 'spec-first AI coding' trend"
  - severity: "Low"
    finding: "Process-IP substrate is partly aspirational: the expert-agent substrate (ADR-033) is still 'Proposed' and the full 14-expert library is deferred to M-dx, so the compounding-knowledge flywheel is established but not yet at the scale the methodology envisions."
    evidence:
      - "docs/adr/ADR-033-expert-agent-substrate.md:3,28"

green_flags:
  - strength: "The engine-neutral IR + 5-method WorkflowEngine port is genuinely well-built and embodied in code, not docs: a pure I/O-free interpreter behind two ports, with engine identity as a free-text string and no engine type in the contract. ADR-037 confirms the interpreter is so cleanly engine-agnostic that a 3rd in-process engine reuses it verbatim — that reusability is real, defensible execution capital."
    evidence:
      - "protos/zynax/v1/workflow_compiler.proto:205-241; services/engine-adapter/internal/domain/interpreter.go:46-101; docs/adr/ADR-037-...:26-30,44-48"
  - strength: "Adapter-first no-SDK is the cleanest, most-defensible embodiment of any candidate: a minimal 2-RPC contract is THE extension boundary, proven by 5 heterogeneous adapters (4 Go + 1 Python framework) that implement the servicer with zero SDK import. This is 'novel mechanism proven in code' in the sense that matters — low-friction extensibility that actually works across languages."
    evidence:
      - "protos/zynax/v1/agent.proto:31-47; grep AgentServiceServicer → 5 adapters; grep zynax_sdk agents/adapters/langgraph → empty"
  - strength: "SPDD is partly embodied in executable tooling (a real Go canvas validator + AI-context gitleaks config + CI freshness gate + a closed Draft→applied/rejected learnings loop that rejects bad patterns). As an INTEGRATED AI-native delivery system it is ahead of most open-source projects and is the most credible source of compounding, hard-to-copy process advantage."
    evidence:
      - "cmd/zynax-ci/validate/canvas.go:36-170; docs/adr/ADR-019-...:29-81; 5.12 (Wave A) docs/due-diligence/2026-06-20-dd-wave-a-findings.md:1650-1656"
  - strength: "Event-driven state-machine choice with first-class loops + HITL is a sound, differentiating architecture decision vs DAG-only competitors (LangGraph/Airflow framing), and it is embodied (loops run, HITL is a state type) — execution-credible even if not mechanism-novel."
    evidence:
      - "docs/adr/ADR-014-...:5-21; protos/zynax/v1/workflow_compiler.proto:132-134,171-185; interpreter.go:67-100"

open_questions:
  - "What could a competent team NOT rebuild in 6 months? On this evidence: nothing in the technical innovations (IR + port + gRPC contract are all reproducible patterns). The non-reproducible asset is the accumulated learnings corpus + integrated SPDD discipline — but that is process capital, single-maintainer-authored, and being commoditized by spec-kit/Kiro. The honest answer is the 6-month-defensible surface is near-zero on idea, modest on integrated execution."
  - "Is there ANY patent/trade-secret/network-effect protecting any candidate, or is the entire moat execution speed + first-mover in the 'control-plane-for-AI-workflows' niche? No IP-protection mechanism found in repo."
  - "Will the second/third engine ever reach IR-interpretation parity (making portability operationally real), or is process-per-engine the permanent model (5.1 open_q)? Until then the headline innovation is a design, not a proven capability."

unknowns:
  - "Productivity multiplier of SPDD — no committed DORA/velocity baseline; the 'AI-native methodology compounds value' claim rests on E5 docs, not measured E1 (cross-ref 5.12:1668-1670). Marked CLAIMED."
  - "Whether any external party has adopted SPDD or the IR/adapter pattern (network-effect / ecosystem evidence) — no external adopters found in repo; Wave A 5.24 confirms bus factor 1, zero named adopters."
  - "Patent landscape / freedom-to-operate around 'engine-portable AI workflow IR' and 'gRPC capability contract' — not assessable from repo (no IP filings, E7 patent search out of scope/offline)."

cross_references:
  - to_agent: "5.1 Architecture"
    note: "The portability-embodiment gap (Argo stub, only Temporal interprets IR) is 5.1's primary finding; I consume it as the durability/embodiment ceiling on the IR innovation and do not re-score the architecture zone (§3.1)."
    evidence: ["docs/due-diligence/2026-06-20-dd-wave-a-findings.md:168-176,238-244", "services/engine-adapter/internal/infrastructure/argo_engine.go:62-98"]
  - to_agent: "5.12 AI Workflow"
    note: "SPDD process-IP scoring derives from 5.12's enforcement findings (soft gate, CLAIMED multiplier, closed learnings loop, single-maintainer). 5.12 explicitly hands SPDD to 5.26 as the primary novelty/IP candidate (their cross_ref:1671-1674). I score its VALUE-as-innovation; I do not re-score their enforcement zone."
    evidence: ["docs/due-diligence/2026-06-20-dd-wave-a-findings.md:1671-1674,1607-1613,1668-1670"]
  - to_agent: "5.17 Risk / 5.19 Investment"
    note: "Net innovation verdict: value is in disciplined integrated EXECUTION, not protectable IP; durable moat ~near-zero on idea, modest on execution + first-mover; SPDD commoditizing (spec-kit/Kiro). Feeds the 'defensibility / moat' risk and valuation discount."
    evidence: ["docs/adr/ADR-014-...:22-25", "E7: github/spec-kit, AWS Kiro"]
  - to_agent: "5.24 Repo Health"
    note: "Process-IP durability is capped by bus factor 1 (single-maintainer-authored ADRs/learnings) — confirmed in 5.24. The compounding-knowledge flywheel has a single point of failure."
    evidence: ["docs/due-diligence/2026-06-20-dd-wave-a-findings.md:50", "docs/adr/ADR-033-...:3"]

recommendations:
  - priority: "P0"
    action: "Stop positioning the four candidates as 'novel IP / moat'; position them as a well-executed integration of known patterns. Reserve any 'novel' language for the one operationally-proven, defensible embodiment (adapter-first no-SDK routing), and re-label engine-portability as 'engine-neutral IR, Temporal reference interpreter' until a second engine interprets the IR."
    rationale: "Three of four candidates are reframings of self-cited prior art; the portability claim is CONTRADICTED at execution (Part 1 §1.10 drift class). Overstating novelty is the fastest way to fail technical diligence."
  - priority: "P1"
    action: "Make the strongest real innovation defensible by depth: bring a second engine to IR-interpretation parity + add a cross-engine parity test, so 'runs on N engines without rewrite' is a PROVEN running capability, not a design."
    rationale: "Converts the headline moat from interface-only to operationally embodied — the only candidate that, if proven, would be genuinely hard to replicate quickly."
  - priority: "P1"
    action: "Measure SPDD's value: commit a pre/post-SPDD DORA or defect-rate baseline, and harden the canvas gate (issue-matched canvas, fail-on-not-Aligned-at-merge per 5.12) so the process IP is evidenced (E1) and enforced, racing the spec-kit/Kiro commoditization curve."
    rationale: "Turns the strongest process-IP candidate from CLAIMED multiplier + soft gate into measured, enforced, defensible methodology before the window closes."
  - priority: "P2"
    action: "Reduce process-IP bus factor: socialize SPDD/ADR authorship beyond one maintainer and accelerate ADR-033 from Proposed toward the fuller expert substrate, so the compounding-learnings flywheel survives a single departure."
    rationale: "The only durable, hard-to-copy asset (accumulated learnings + discipline) is single-maintainer-authored; that is the binding constraint on process-IP durability."
```

---

## (b) §6.2 Prose section

## 5.26 Innovation — Score: 6 (Medium)

**Mission recap:** Identify what in Zynax is genuinely novel and defensible (technical and process IP) versus competent-but-commodity, judge each candidate against both the code and the prior-art landscape, and rate innovation versus disciplined execution.

**Verdict:** Zynax's value is overwhelmingly in **disciplined, integrated execution, not in protectable invention**. All four candidate innovations are well-built *reframings of established prior art*: the engine-agnostic Workflow IR echoes Apache Beam's portable runner, Argo's IR, and Dapr Workflow; the event-driven state-machine model is self-cited as inspired by XState, AWS Step Functions, and Temporal (ADR-014:22-25); adapter-first no-SDK routing is the same "implement-a-gRPC-contract-to-extend" pattern as Dapr pluggable components and Envoy ext-proc; and SPDD is one of several convergent 2025-26 "spec-first AI coding" efforts (GitHub spec-kit, AWS Kiro). None is mechanism-novel. What is real is that the integration is genuinely well-engineered and largely embodied in code rather than docs — which is execution capital, not a moat.

The **boldest "novel" claim — engine-agnostic portability — is the weakest at embodiment.** It is real at the interface (one engine-neutral `WorkflowIR` with zero engine types, a clean 5-method `WorkflowEngine` port both engines satisfy with compile-time assertions) but only Temporal actually *interprets* the IR; the Argo path is a non-interpreting stub (consumed from 5.1). An innovation wired structurally but not proven functionally is precisely the "easily replicated / not embodied" red flag. The **most-defensible candidate is adapter-first no-SDK routing**: a minimal 2-RPC `AgentService` contract that is genuinely THE extension boundary, proven by five heterogeneous adapters — including a Python LangGraph adapter — that implement the servicer with zero SDK import. SPDD is the strongest *process*-IP candidate and is partly embodied in executable tooling (a real Go canvas validator, AI-context leak controls, a closed learnings loop), but its headline gate is soft, its productivity claim is unmeasured, and the idea is commoditizing in real time.

**Sub-dimension scores:**

| Sub-dimension | Score | Confidence | Evidence |
|---|---|---|---|
| Engine-agnostic IR — novelty vs prior art | 7 | High | `workflow_compiler.proto:205-241`; `interpreter.go:46-101`; `ADR-012:6-23`; E7 Beam/Argo/Dapr |
| Multi-engine portability — operational embodiment | 4 | High | 5.1 `docs/due-diligence/2026-06-20-dd-wave-a-findings.md:168-176,238-244`; `interpreter.go:46-101`; `ADR-037:26-30` |
| Event-driven state machine (ADR-014, not-DAGs) | 6 | High | `ADR-014:5-25`; `workflow_compiler.proto:132-134,171-185`; `interpreter.go:67-100`; E7 XState/ASL |
| Adapter-first no-SDK routing (ADR-013) | 7 | High | `agent.proto:31-47`; grep `AgentServiceServicer`→5 adapters; grep `zynax_sdk` langgraph→empty; `ADR-013:20-23` |
| SPDD AI-native methodology — process IP | 6 | Medium | `ADR-019:29-81`; `canvas.go:36-170`; 5.12 `:1607-1613,1668-1670`; E7 spec-kit/Kiro |
| Innovation-vs-execution split | 7 | High | 5.1 `:42-43,265-275`; 5.5 `:45`; `ADR-014:22-25`; `ADR-033:3,28` |

**Drift test:**
- *"Same workflow runs on Temporal OR Argo without rewrite" (boldest novel moat)* → **PARTIAL.** VERIFIED at the IR/contract/port boundary; CONTRADICTED at execution — only Temporal interprets the IR, the Argo leg is a stub (5.1 `:168-176`; `argo_engine.go:62-98`). The innovation is a portability *design*, not a proven multi-engine *capability*.
- *"Event-driven state machines are a novel mechanism DAGs can't match"* → **PARTIAL.** Loops + HITL are embodied and run (`interpreter.go:67-100`; `proto:132-134`), but ADR-014:22-25 self-cites XState/Step Functions/Temporal — a differentiating choice, not an invention.
- *"Adapter-first no-SDK is real, not marketing"* → **VERIFIED.** 2-RPC contract is the plugin boundary; 5 adapters implement it, the LangGraph adapter imports no SDK.
- *"SPDD is transferable methodology embodied in tooling, not ceremony"* → **PARTIAL.** Real validator + leak controls + learnings loop, but the gate is soft, the multiplier is CLAIMED (E5), and the idea is commoditizing (E7).

**Red flags (severity-ordered):**
1. **High** — The boldest novel moat (engine portability) is innovation at the interface but not embodied in execution; only Temporal interprets the IR (5.1 `:238-244`; `argo_engine.go:62-98`; even ADR-037's lightweight engine is Rejected/superseded).
2. **Medium** — Every technical candidate reframes self-cited prior art (Beam/Argo/Dapr; XState/Step Functions/Temporal; Dapr/Envoy); the durable-IP surface is thin, protection is copyable execution (`ADR-014:22-25`, `ADR-015:9-17`).
3. **Medium** — SPDD, the strongest process-IP candidate, is commoditizing (spec-kit/Kiro), its enforcement is softer than advertised, and the value-prop is unmeasured (5.12 `:1626-1631,1668-1670`).
4. **Low** — Expert-agent substrate (ADR-033) still "Proposed"; full library deferred to M-dx — the compounding flywheel is established but sub-scale (`ADR-033:3,28`).

**Green flags:**
- Engine-neutral IR + I/O-free interpreter behind two ports is genuinely well-built and reusable verbatim by a third engine — real execution capital (`workflow_compiler.proto:205-241`; `interpreter.go:46-101`; `ADR-037:26-30`).
- Adapter-first no-SDK is the cleanest embodied innovation: a 2-RPC contract proven by 5 cross-language adapters with zero SDK import (`agent.proto:31-47`; grep evidence).
- SPDD is partly embodied in executable tooling + a closed, self-correcting learnings loop — an integrated AI-native delivery system ahead of most OSS (`canvas.go:36-170`; 5.12 `:1650-1656`).
- Event-driven state machine with first-class loops/HITL is a sound, differentiating, embodied architecture choice (`ADR-014:5-21`; `proto:132-134`; `interpreter.go:67-100`).

**Open questions / unknowns:** What could a competent team *not* rebuild in 6 months? On this evidence, nothing in the technical candidates — the non-reproducible asset is the accumulated learnings corpus + integrated SPDD discipline, which is single-maintainer-authored and commoditizing. SPDD's productivity multiplier is unmeasured (no DORA baseline). No patent/trade-secret/network-effect protection found in repo; no external adopters.

**Recommendations:** P0 — stop positioning the candidates as novel IP/moat; reserve "novel" for the one operationally-proven embodiment (no-SDK routing) and re-label portability honestly. P1 — bring a second engine to IR-interpretation parity + a cross-engine parity test to make the moat operationally real; and measure SPDD (DORA baseline) + harden the canvas gate while racing the commoditization curve. P2 — reduce process-IP bus factor and advance ADR-033 so the flywheel survives a single departure.

**Cross-references:** 5.1 Architecture (portability-embodiment gap is their primary finding; consumed, not re-scored); 5.12 AI Workflow (SPDD process-IP enforcement findings; consumed, not re-scored); 5.17 Risk / 5.19 Investment (moat ~near-zero on idea, modest on execution + first-mover → valuation discount); 5.24 Repo Health (process-IP durability capped by bus factor 1).
