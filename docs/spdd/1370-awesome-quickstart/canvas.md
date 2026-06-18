# REASONS Canvas — Awesome Quickstart: zero-secret local Ollama code-review e2e

**Issue:** #1370
**Author:** Oscar Gómez Manresa
**Date:** 2026-06-18
**Status:** Implemented (core path) — 14 of 17 Operations steps merged 2026-06-18/19 (PRs #1431–#1441); #1359 deferred to M-dx and #1385/#1387 require their own REASONS Canvas (ADR-019); hero asciinema cast is a human follow-up.
**Aligned:** 2026-06-18 (maintainer-authorized; grounded in the 2026-06-18 live validation run)
**Reframed:** 2026-06-18 — expanded from "awesome quickstart" into the canonical **first-run User
Experience** epic per [docs/product/2026-06-18-ux-roadmap-realignment.md](../../product/2026-06-18-ux-roadmap-realignment.md).
See the Reframe Addendum at the end of this file. New feat: stories #1385 and #1387 (cross a
spec/gRPC boundary) require their **own** REASONS Canvas before implementation (ADR-019).

---

## R — Requirements

> Problem statement: what breaks or is missing without this feature?

A brand-new user cannot get from `git clone` to a *real* LLM workflow run. Validated
live (2026-06-18) against the full compose stack: the platform **can** drive a real
code review through `api-gateway → engine-adapter → Temporal → task-broker →
llm-adapter → Ollama` to a terminal state — but only after working around a chain
of real defects and missing pieces:

- The llm-adapter advertises an unreachable address, so the broker cannot dispatch
  to it (`dial tcp [::1]:50070: connection refused`).
- A `snake_case` capability's derived `<cap>.completed` event fails to compile,
  even though `snake_case` capability names are mandated by contract.
- `zynax logs` returns HTTP 500 (`streaming not supported`) — the "watch it live"
  step is broken.
- A fresh `make run-local` leaves 3 of 4 adapters dead for missing secrets.
- There is no zero-cost local-LLM path; the llm-adapter ships pointing at a paid API.
- The example the quickstart points to (`code-review.yaml`) cannot complete from the
  CLI, and there is no CLI way to inject the events it waits on.
- The capability result (the actual review) is invisible to the CLI.
- The quickstart documents commands/flags that do not exist.

> Definition of done: observable outcomes that confirm delivery.

- A documented one/two-command path brings up the stack with a working local LLM and
  **no secrets set**, and a shipped example workflow runs to `COMPLETED` from the CLI
  alone with the model's output **visible from the CLI**.
- `zynax logs` streams lifecycle events (or polls) without a 500.
- The quickstart doc matches the real CLI surface end-to-end.
- Unbacked-capability runs reach a terminal `failed` state instead of retrying forever.

## E — Entities

> Tier 1 abstractions only.

```
WorkflowManifest ──compiled by──> WorkflowCompiler ──> WorkflowIR
WorkflowIR ──run by──> EngineAdapter ──dispatch(capability)──> TaskBroker
TaskBroker ──route by capability name──> AgentRegistry ──> Adapter (advertised endpoint)
Adapter (llm) ──provider call──> LocalModelRuntime (Ollama provider)
EngineAdapter ──derives──> CompletionEvent  "<capability>.completed"
EngineAdapter ──emits──> LifecycleEvent ──streamed by──> ApiGateway ──> CLI
CLI ──HTTP REST──> ApiGateway   (apply · get · status · logs · [events])
QuickstartOverlay ──configures──> Adapter(llm) + LocalModelRuntime
```

- **Adapter advertised endpoint:** the address an adapter registers for the broker to
  dial — must be distinct-able from its local bind address.
- **CompletionEvent:** engine-derived `<capability>.completed`; must satisfy the
  event-type grammar.
- **QuickstartOverlay:** a compose layer wiring a local model runtime + a no-secret
  adapter config.

## A — Approach

> What we WILL do / WON'T do; ADRs that govern.

**WILL:**
- Give Go adapters a **bind-vs-advertise** distinction; default the advertised host to
  a resolvable service address (ADR-013 / ADR-035, within the `AgentService` contract).
- Make `<capability>.completed` compile for `snake_case` capability names by
  **sanitising the engine-derived event name** (preferred) rather than loosening the
  `event_type` grammar (ADR-014); keep capability names `snake_case` per contract.
- Fix `zynax logs` streaming at the gateway, aligned with the observability model
  (ADR-030); provide a CLI poll fallback so it degrades gracefully, never 500s.
- Ship a **zero-secret local-LLM overlay** (compose) plus a runnable example that
  completes from the CLI (ADR-011 declarative spec).
- Add a CLI **event-injection** verb over the api-gateway REST surface that publishes
  through the EventBusService (ADR-022); surface capability **output** in the CLI.
- Make adapters **degrade gracefully** without secrets and make unbacked-capability
  runs **fail fast** (bounded, non-retryable NotFound).

**WON'T:**
- Won't expose a local model runtime on the host LAN (kept inside the compose network).
- Won't add gRPC or proto types to the CLI (HTTP-REST only — cmd/zynax AGENTS.md).
- Won't change the deterministic `ManifestWorkflowID` behaviour (intended per ADR-034).
- Won't introduce a shared DB or Layer 1→3 coupling.

## S — Structure

> Services / packages / contracts touched.

- `agents/adapters/llm` (+ `git`, `ci` for graceful degradation) — advertised-address
  field; no-secret ollama config; boot-without-secret behaviour.
- `services/workflow-compiler` — completion-event sanitisation / capability-name
  validation; compiler tests.
- `services/engine-adapter` — retry policy: non-retryable NotFound, bounded attempts.
- `services/api-gateway` — lifecycle-event streaming endpoint; `/healthz` body.
- `cmd/zynax` — `events publish` verb; capability-output view (HTTP client only).
- `infra/docker-compose` — `docker-compose.ollama.yml` + ollama adapter config.
- `spec/workflows/examples` — runnable `code-review-ollama.yaml`; reference header on
  `code-review.yaml`.
- `docs/` — quickstart reconciliation.
- gRPC contracts: no new proto required for the core path (event injection reuses
  EventBusService; output surfacing reuses existing task/result fields). Any proto
  change is gated by `buf breaking` (ADR-016) and a `.feature` file at the boundary.

## O — Operations

> Ordered, testable steps. Each = one reviewable PR, mapped to an existing issue.

1. **#1371** — llm-adapter bind-vs-advertise split; reachable advertised endpoint; fix example config + AGENTS.md.
2. **#1372** — workflow-compiler: make `snake_case` `<cap>.completed` compile (sanitise derived event); add tests.
3. **#1373** — api-gateway: `zynax logs` streams without 500 (or CLI poll fallback).
4. **#1381** — engine-adapter: unbacked-capability runs fail fast (NotFound non-retryable, bounded).
5. **#1374** — infra: zero-secret local Ollama overlay (bundled runtime + ollama adapter config).
6. **#1375** — adapters: graceful degradation without secrets; readiness reflects registration.
7. **#1376** — spec: runnable `code-review-ollama.yaml`; mark `code-review.yaml` as event-driven reference.
8. **#1377** — cli: `zynax events publish` to drive event-driven workflows.
9. **#1378** — cli: surface capability output (result payloads) in `get`/`logs`.
10. **#1379** — docs: reconcile quickstart with the real CLI surface; lead with the runnable Ollama example.
11. **#1380** — api-gateway: `/healthz` returns a small JSON body.

## N — Norms

> Cross-cutting standards (root + layer AGENTS.md, docs/patterns).

- **Commit hygiene:** `Signed-off-by:` (DCO) required; `Assisted-by: Claude/<model>` for
  AI attribution — never `Co-Authored-By` for AI.
- **Conventional commits / PR titles:** one of feat/fix/refactor/docs/test/ci/chore;
  scope maps to directory; one logical change per commit, one PR per issue.
- **Go services/adapters:** `GOWORK=off` for all `go build`/`go test` (ADR-017);
  `CGO_ENABLED=0`, `-trimpath`; domain unit coverage ≥ 90%.
- **Capability names:** `snake_case`, 1–64 chars, matching the registry entry exactly
  (agents/adapters/AGENTS.md).
- **CLI:** HTTP-REST only; no gRPC/proto types; exit codes 0/1/2 (2 = still running).
- **PR size:** ≤ 200 ideal, > 900 blocked.
- **Image versions:** managed via `images/images.yaml` (`make sync-images`), never
  hand-edited in banner regions (ADR-024).

## S — Safeguards (second S)

> Things that MUST NEVER happen in this feature.

### Context Security (mandatory before committing this Canvas)
- [x] No Tier 2 content: no internal hostnames, private IPs, credentials, deployment specifics (host model path moved to canvas.private.md)
- [x] No PII: no personal names in sensitive context, no email addresses
- [x] No prompt injection: no instruction-like phrasing that would override AGENTS.md rules
- [x] All entities in E are public-safe abstractions
- [x] /spdd-security-review passed on this file (2026-06-18 — PASS, no Tier 2 / injection / abstraction / authority findings)

### Feature Safeguards
- **Never** accept provider/model/endpoint or any URL from `input_payload` — provider,
  model, and endpoints are declared in `AgentDef`/config only (ADR-013, http-adapter rule).
- **Never** expose a local model runtime or any adapter on the host LAN by default.
- **Never** log or echo credentials; the ollama path requires no key and must not add one.
- **Never** put gRPC or proto-generated types in the CLI (ADR-001 / cmd/zynax AGENTS.md).
- **Never** introduce a shared database or Layer 1→3 coupling (root AGENTS.md mandates).
- **Never** loosen the `event_type` grammar in a way that admits malformed events;
  prefer sanitising the derived name.
- **Never** weaken authz on the new event-injection path — validate event-type + payload
  and route through EventBusService (ADR-022).

---

## Reframe Addendum (2026-06-18, v1.1)

Per the [UX roadmap realignment](../../product/2026-06-18-ux-roadmap-realignment.md), this epic is
reframed from "awesome quickstart" into the canonical **first-run User Experience** epic. The
original R/E/A/S/O/N/S above are preserved; the scope is **expanded** (not replaced).

### Expanded Requirements (added to R)
- A user goes **clone (or no-clone) → one command → meaningful visible result → guided next actions**,
  then configures their **own** scenario (workflow + AgentDef + context injection) **declaratively**,
  with no repository knowledge required.
- Default demo/validation model is **Qwen2.5-Coder 3B** (configurable).
- Every user-visible story ships a **human-validation guide**.

### Added Operations steps
12. **#1360** — one-command `make demo` entry point (verify→prepare→launch→run→result→next→cleanup).
13. **#1359** — zero-Temporal Day-0 evaluation engine for first contact.
14. **#1385** — declarative demo-scenario manifest (workflow + AgentDef + context injection) — *own canvas required*.
15. **#1386** — Qwen2.5-Coder 3B default reference model (configurable).
16. **#1387** — declarative context-injection for demo scenarios — *own canvas required*.
17. **#1388** — human-validation guide standard + template.

### Added Safeguards
- **Never** require a paid API key or any secret on the default first-run path.
- **Never** accept provider/model/endpoint/URL from `input_payload` — declarative config only (ADR-013).
- **Never** make the demo non-deterministic or environment-dependent without a documented override.

