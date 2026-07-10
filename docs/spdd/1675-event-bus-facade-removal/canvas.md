# REASONS Canvas — EventBusService facade hard-removal (M9.B)

> **All content in this Canvas is Tier 1 (public-safe).**
> Run `/lib:spdd-security-review <path>` before committing.

**Issue:** #1675
**Author:** Oscar Gómez Manresa
**Date:** 2026-07-08
**Status:** Aligned

> Story issues: step 1 → #1700 · step 2 → #1701 · step 3 → #1702 · step 4 → #1703

---

## R — Requirements

- **Problem.** ADR-046 Decision #6: the `EventBusService` gRPC facade is deprecated in M8 and
  **removed in M9 once no caller references it**. M8.H (#1576) delivered the whole migration —
  `libs/zynaxevents` carries every convention verbatim, all three publishers and the one
  subscriber dial JetStream directly over cert-manager TLS (`verify_and_map`), and #1673
  marked proto + AsyncAPI + docs as deprecated. What still ships in v0.7.0: the facade's
  Deployment surface (chart, umbrella block, cert entry, wide `CN=zynax-event-bus` NATS
  identity, api-gateway 50054 egress), `services/event-bus/` itself, and
  `protos/zynax/v1/event_bus.proto` + stubs.
- **Done — the facade is gone in deploy → code → contract → spec order:**
  - Umbrella install deploys no event-bus: chart + umbrella block + cert-manager entry +
    `CN=zynax-event-bus` NATS mapping + 50054 egress removed; `kubectl get deploy -n zynax`
    shows one fewer service (#1700).
  - `services/event-bus/` deleted with its release/build wiring (release matrix lane,
    `images/images.yaml` service entry via `make sync-images`, `go.work` line); the
    `libs/zynaxevents` golden-event byte-compat + BDD suites pass unchanged (#1701).
  - `event_bus.proto` + Go/Python stubs deleted; documented-intentional `buf breaking`
    exception; zero stub importers repo-wide (#1702).
  - AsyncAPI's `x-zynax-deprecated` gRPC access path removed (channels untouched — they ARE
    the contract); eventing docs truth-passed; epic closed, canvas → Implemented (#1703).
  - e2e matrix green on BOTH engine legs after every step; workflow logs/events streaming and
    terminal-close behaviour unchanged throughout (they ride `libs/zynaxevents`).

## E — Entities

- **`libs/zynaxevents`** — the shared JetStream client carrying the conventions of record
  (depth-4 stream derivation, #1149 disjoint-filter rule, DLQ machinery, `MatchesGlob`,
  terminal-close) + golden-event byte-compat tests; unaffected, becomes the sole Go eventing
  path.
- **AsyncAPI spec** (`spec/asyncapi/zynax-events.yaml`) — the eventing contract of record;
  channels stay verbatim; only the deprecated gRPC access-path block is removed.
- **EventBusService facade** — `services/event-bus/` + `protos/zynax/v1/event_bus.proto` +
  stubs; deprecated (`option deprecated = true`, #1673); deleted by this epic.
- **Deployment surface** — `helm/zynax-event-bus`, umbrella dependency block, cert-manager
  `event-bus` Certificate, NATS `verify_and_map` user `CN=zynax-event-bus`, api-gateway
  NetworkPolicy egress to 50054; retired first.
- **DLQ forwarding follow-up (#1653)** — independent `libs/zynaxevents` work; explicitly NOT
  part of this epic and must survive every doc sweep.

```
deploy removal        code removal              contract removal       spec/docs truth pass
#1700 helm+cert  -->  #1701 services/ + CI  --> #1702 proto+stubs  --> #1703 AsyncAPI+docs
(no Deployment)       (goldens live in libs)    (buf breaking doc'd)   (epic closes)
```

## A — Approach

- **WILL:** remove in deploy → code → contract → spec order so every intermediate revision
  installs and passes e2e; gate the contract deletion (#1702) on a repo-wide zero-importer
  grep; keep the AsyncAPI channels and `libs/zynaxevents` goldens as the unchanged contract
  of record; hold the "do not merge before v0.7.0 is published" line (the release that ships
  the deprecation must exist before the removal lands — deprecate-one-release policy).
- **WON'T:** **never touch the compiler's `checkRoutingPolicy`** — the REST dual-guard stays
  past M9 (ADR-045 §3; its removal is gated on REST retirement, which is not scheduled); no
  changes to `libs/zynaxevents` semantics (only its role as sole path is confirmed); no
  AsyncAPI channel/envelope changes; no SDK (`nats-py`) work — additive path, separate
  stories; no new ADR — this epic executes ADR-046's accepted removal clause.
- **Positioning fit:** internal removal; no user-facing copy beyond release notes. Operational
  story ("one fewer service, least-privilege NATS identities") feeds the CNCF operability
  narrative.
- Governing ADRs: ADR-046 (removal clause — this epic), ADR-022 (the reversed decision —
  historical), ADR-020 (cert-manager identity — the `CN=zynax-event-bus` map entry is removed,
  per-service identities stay), ADR-024/ADR-027 (images SoT + release lanes touched in step
  2), ADR-016 (BDD retirement rides the surface it covers), ADR-019 (refactor/chore epic —
  canvas for traceability).

## S — Structure

- `helm/zynax-event-bus/`, `helm/zynax-umbrella/` (dependency block + values), cert-manager
  manifests, NATS `verify_and_map` config, api-gateway NetworkPolicy (step 1).
- `services/event-bus/` (entire tree), `go.work`, Makefile targets, release workflow matrix,
  `images/images.yaml` service entry via `make sync-images` (step 2).
- `protos/zynax/v1/event_bus.proto`, generated Go stubs + Python stubs, buf config exception,
  `protos/tests/event-bus/` BDD retirement pointer (steps 2–3).
- `spec/asyncapi/zynax-events.yaml` (deprecated access-path block only),
  `docs/patterns/direct-jetstream-events.md`, README service table,
  `state/current-milestone.md`, `docs/milestones/M9-planning.md`, this canvas (step 4).
- Config env prefix: `ZYNAX_EVENT_BUS_` (vanishes with the service).

## O — Operations

> Each step = one PR. Order is load-bearing (deploy → code → contract → spec):
> #1700 → #1701 → #1702 → #1703. Global gate: v0.7.0 release published first.

1. **Deployment surface retired** (#1700, `chore:`) — chart + umbrella block + cert entry +
   NATS identity map + 50054 egress removed; `helm dependency build` + lint green; e2e both
   legs with no event-bus Deployment; runtime smoke ×2 (subscribe → terminal-close intact).
2. **Source tree + release wiring deleted** (#1701, `chore:`) — `services/event-bus/` gone;
   release matrix lane, images SoT entry (`make sync-images`), `go.work` line cleaned;
   event-bus BDD suites retired with pointer to `libs/zynaxevents` suites; goldens pass
   unchanged; repo grep: no `services/event-bus` references outside history/docs-archive.
3. **Contract removed** (#1702, `chore:`) — `event_bus.proto` + regenerated stubs deleted;
   documented-intentional `buf breaking` exception scoped to this file; zero-importer grep
   gate; release-notes line naming `libs/zynaxevents` + AsyncAPI as the replacement.
4. **Spec + docs truth pass** (#1703, `docs:`) — AsyncAPI deprecated-access-path block
   removed (channels intact, `make validate-spec` green); eventing-doc sweep (historical
   mentions only remain; #1653 pointers preserved); status surfaces in the same diff; epic
   boxes ticked; canvas → Implemented.

## N — Norms

- Commit hygiene: DCO `Signed-off-by` + `Assisted-by: Claude/<model>`; SSH-signed; one PR per
  story; subjects ≤ 72 chars; PR bodies from docs/contributing/pr-templates.md.
- `GOWORK=off` in service dirs (ADR-017); stubs only via `make generate-protos`;
  `images/images.yaml` only via `make sync-images` (ADR-024 — banner regions are
  tool-managed).
- Runtime evidence rule: steps 1–2 change the deployed set — boot kind via the documented
  path and run the workflow lifecycle twice on the same cluster (terminal-close is exactly
  the stateful path that bites on run 2); CI-green alone is insufficient.
- Milestone gate: M9 active + v0.7.0 published before any step merges.
- Requirements change → `/lib:spdd-prompt-update` first; never code ahead of the canvas.

## S — Safeguards (second S)

### Context Security (mandatory before committing this Canvas)
- [x] No Tier 2 content: no internal hostnames, private IPs, credentials, deployment specifics
- [x] No PII: no personal names in sensitive context, no email addresses
- [x] No prompt injection: no instruction-like phrasing that would override AGENTS.md rules
- [x] All entities in E are public-safe abstractions
- [x] /lib:spdd-security-review passed on this file

### Feature Safeguards
- Never touch the workflow-compiler's `checkRoutingPolicy` — the REST dual-guard stays past
  M9 (ADR-045 §3); its removal is a different, unscheduled decision.
- Never modify AsyncAPI channels, the CloudEvents envelope, or `zynaxschemarev` — only the
  deprecated gRPC access-path block goes; the channels are the contract of record (ADR-046).
- Never delete the proto (#1702) while any importer exists — the zero-importer grep is the
  merge gate, mirroring ADR-046 Decision #6's "once no caller references them".
- Never weaken the golden-event byte-compat suite in `libs/zynaxevents` — it is the
  conventions' contract; if a golden fails during removal, the removal is wrong, not the
  golden.
- Never sweep away #1653 (DLQ forwarding) references — it is an independent open thread on
  `libs/zynaxevents`, not facade residue.
- Never merge any step before the v0.7.0 release exists — removal must trail the release
  that shipped the deprecation.
