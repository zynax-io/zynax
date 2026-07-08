# REASONS Canvas — Event-bus facade retirement: direct NATS JetStream via libs/zynaxevents (M8.H)

> **All content in this Canvas is Tier 1 (public-safe).**
> Run `/lib:spdd-security-review <path>` before committing.

**Issue:** #1576
**Author:** Oscar Gómez Manresa
**Date:** 2026-07-06
**Status:** Implemented

> Story issues: step 1 → #1644 · step 2 → #1645 · step 3 → #1646 · step 4 → #1647 ·
> step 5 → #1648 · step 6 → #1649 · step 7 → #1650

---

## R — Requirements

- **Problem.** The event bus is already NATS JetStream, but every publisher and subscriber pays an
  extra gRPC hop through the `EventBusService` facade — a whole service (deploy, HPA,
  NetworkPolicy, mTLS cert, metrics) that only forwards messages. The policy chokepoint ADR-022
  reserved the facade for was never built (no authz/rate-limit logic exists in it). All
  substantive logic lives in one file, `services/event-bus/internal/infrastructure/nats.go`
  (~543 LOC of conventions); the handler is a ~168-LOC validate-and-map shim. Meanwhile the NATS
  link itself is **plaintext with no auth** — the facade being the sole client was the only bound.
- **Done — publishers/subscribers use JetStream directly through a shared client (ADR-046):**
  - New Go module **`libs/zynaxevents`** carries the `nats.go` conventions **verbatim**: depth-4
    stream derivation + the #1149 disjoint-filter rule, `DLQ_<src>` / `zynax.dlq.<prefix>.dead` /
    `WorkQueuePolicy`, `RetryBackoff` + `MaxDeliver=5`, durable-name sanitizing, `MatchesGlob`,
    CloudEvents JSON + `ce-*` headers, and `zynaxobs` trace inject/extract.
  - The three publisher call sites (engine-adapter, task-broker, api-gateway) and the **one**
    subscriber (api-gateway's Subscribe→REST bridge — glob + workflow-scope filter +
    terminal-event stream-close) consume the shared client directly.
  - The NATS link gains **TLS with cert-manager-issued client certificates** and `verify_and_map`
    cert-mapped users + least-privilege subject permissions — in scope per ADR-046 Decision #4
    ("migrated" is not done until callers dial TLS with their cert identity); no new secret
    system, zero-secret kind first-run preserved (ADR-041).
  - **Gates (ADR-046 acceptance):** golden-event **byte-compat** fixtures + **DLQ /
    durable-consumer BDD** committed before implementation (ADR-016/019).
  - The `EventBusService` facade is **deprecated in M8, kept deployable** — proto `deprecated`
    options + docs; hard removal of `services/event-bus/`, `event_bus.proto`, and stubs is **M9**.
  - The **AsyncAPI spec stays the single contract of record** (`spec/asyncapi/zynax-events.yaml`;
    `make validate-spec` green).
  - Runtime smoke: a service publishes AND consumes an event directly over JetStream on kind; DLQ
    + topic conventions honoured; the terminal event still closes the REST watch — **run twice**
    (persistence/terminal-close bites the second run).

---

## E — Entities

```
libs/zynaxevents (NEW Go module)              ← the nats.go conventions, verbatim
├── publish side: stream derivation (depth-4, #1149), DLQ ensure (WorkQueuePolicy),
│   CloudEvents envelope + ce-* headers, trace inject, RetryBackoff/MaxDeliver consts
└── subscribe side: durable consumers (sanitized names), MatchesGlob, workflow-scope
    filter, terminal-event stream-close, DLQ delivery, trace extract

engine-adapter / task-broker                  ← publishers: gRPC facade → libs/zynaxevents
api-gateway                                   ← publisher AND the sole subscriber
                                                 (Subscribe→REST bridge, hardest migration)
NATS JetStream (helm/charts/nats)             ← gains link TLS + verify_and_map users +
                                                 subject permissions (cert-manager identities)
EventBusService facade (services/event-bus/)  ← DEPRECATED in M8 (kept deployable); removed M9
spec/asyncapi/zynax-events.yaml               ← contract of record (unchanged authority)
golden-event fixtures + DLQ/durable BDD       ← byte-compat acceptance gates
```

---

## A — Approach

**We will:**
- Create `libs/zynaxevents` exactly like `libs/zynaxconfig` / `libs/zynaxobs`: own `go.mod`
  (`github.com/zynax-io/zynax/libs/zynaxevents`), root `go.work` entry, per-consumer
  `require … v0.0.0` + `replace … => ../../libs/zynaxevents` (ADR-006 layout; ADR-017
  `GOWORK=off` regime).
- **Move, not rewrite**: the conventions transfer verbatim from
  `services/event-bus/internal/infrastructure/nats.go` with their unit tests; golden-event
  byte-compat fixtures pin the wire shape (subject, stream name, DLQ names, CloudEvents JSON,
  `ce-*` headers) so drift is mechanically caught. The facade keeps its own copy until M9 — the
  library is the new source of truth, byte-compat guarded on both.
- Land contracts first (ADR-016): the DLQ/durable-consumer `.feature` + golden fixtures precede
  the implementation PRs.
- Follow the ADR-046 migration order: library → three publishers → the api-gateway
  Subscribe→REST bridge → NATS link TLS + cert-mapped users → facade deprecation marks.
- Close the pre-existing plaintext NATS gap in-epic (ADR-046 Decision #4): cert-manager
  `Certificate` for the NATS server, `verify_and_map` + per-service subject permissions in the
  NATS chart values, `tls://` dial + client-cert options in the shared client, NetworkPolicy
  egress for the three callers, e2e values updated so CI exercises the TLS path.
- Deprecate the facade in M8: `option deprecated = true` on the proto service/RPCs (stubs
  regenerated), AsyncAPI `x-zynax-deprecated` / `x-zynax-removal-milestone: M9` markers where the
  facade is referenced, deprecation notes in the service AGENTS.md and umbrella values — while it
  stays deployable and its tests stay green.

**We will NOT:**
- Delete `services/event-bus/`, `protos/zynax/v1/event_bus.proto`, or the generated stubs — M9
  work, gated on "no caller references them" (ADR-046 Decision #6; locked decision 2).
- Change the topic taxonomy, CloudEvents envelope, DLQ names, retry schedule, or terminal-close
  semantics — byte-compat is the gate, not an opportunity to improve.
- Add the Python SDK `nats-py` eventing path (ADR-046 Decision #5) **in this epic** — it is
  additive (no Python eventing exists today, nothing in M8 gates on it, adapters stay gRPC-only
  per ADR-013 upheld) and lands as a tracked follow-up so the byte-compat surface stays
  single-implementation while the conventions settle. Deferral recorded on the epic.
- Build a new policy chokepoint in the library — topic/namespace authz is NATS subject
  permissions keyed off cert identity; internal publish-rate abuse is bounded by JetStream
  per-stream limits; edge request-rate is ADR-044's Envoy edge (ADR-046 Rationale).
- Touch `spec/asyncapi/zynax-events.yaml`'s authority — channels/envelope/`zynaxschemarev`
  stay the contract of record.

**Positioning fit:** neutral for the engine-portability wedge — events are engine-independent and
the taxonomy is preserved, so cross-engine run observability is unchanged; removes a
non-differentiating service from the deploy surface (thin-Zynax, ADR-040 §1).

**Governing ADRs:** ADR-046 (this retirement, Accepted — narrow ADR-001 exception for pub/sub;
ADR-013 upheld), ADR-022 (amended — Option 1 reversed), ADR-020 (cert-manager identities reused;
no shared-secret regression), ADR-017 (`GOWORK=off`), ADR-016/019 (contracts + canvas first),
ADR-041 (zero-secret kind first-run preserved), ADR-008 (durability stays in JetStream).

---

## S — Structure (first S)

```
libs/zynaxevents/                              ← NEW module (go.mod, client, unit tests,
│                                                 golden fixtures, DLQ/durable BDD feature)
go.work                                        ← + ./libs/zynaxevents
services/engine-adapter/
├── go.mod                                     ← + require/replace zynaxevents
└── internal/infrastructure/activities.go      ← publisher: facade client → zynaxevents
services/task-broker/
├── go.mod                                     ← + require/replace zynaxevents
└── internal/infrastructure/event_publisher.go ← publisher: facade client → zynaxevents
services/api-gateway/
├── go.mod                                     ← + require/replace zynaxevents
└── internal/infrastructure/clients.go         ← publisher + SubscribeWorkflowEvents bridge
                                                  (glob + workflow-scope + terminal-close moves
                                                  in-process against a durable consumer)
helm/charts/nats/                              ← server TLS Certificate + verify_and_map +
                                                  per-service subject permissions (values)
helm/charts/cert-manager/                      ← NATS server Certificate (values/templates)
helm/zynax-{engine-adapter,task-broker,api-gateway}/templates/networkpolicy.yaml
                                               ← egress to NATS :4222 (TLS)
scripts/e2e/values-e2e.yaml                    ← NATS TLS on; callers dial tls://
protos/zynax/v1/event_bus.proto                ← option deprecated = true (+ stub regen)
spec/asyncapi/zynax-events.yaml                ← x-zynax-deprecated markers on facade refs only
services/event-bus/                            ← UNTOUCHED except deprecation notes (removed M9)
docs/patterns/ (eventing how-to)               ← direct-JetStream client usage + DLQ/DLQ ops
```

Config env prefix: `ZYNAX_EVENTS_` for the shared client's connection knobs (URL, TLS paths),
resolved per-service from the same cert-manager mounts the gRPC mTLS uses (ADR-020).

---

## O — Operations

1. **Contracts first: golden-event fixtures + DLQ/durable BDD** (#1644, test). Golden fixtures pin the
   current wire bytes (subject, stream/DLQ names, CloudEvents JSON, `ce-*` headers) captured from
   the facade's conventions; `libs/zynaxevents/tests/features/` carries the DLQ/durable-consumer
   `.feature` (retry schedule → DLQ after MaxDeliver, durable resume, terminal-close). *Verify:*
   fixtures assert against the EXISTING `services/event-bus` implementation (both must pass the
   same goldens). (bdd-contract)
2. **`libs/zynaxevents` publish side** (#1645, feat). Module skeleton + verbatim move of stream
   derivation (depth-4, #1149), DLQ ensure (`WorkQueuePolicy`), publish path (CloudEvents
   envelope, `ce-*` headers, trace inject), `RetryBackoff`/`MaxDeliver` consts; ported unit
   tests; `go.work` entry. *Verify:* `GOWORK=off go test` green; golden byte-compat green against
   step 1 fixtures. (go-svc)
3. **`libs/zynaxevents` subscribe side** (#1646, feat). Verbatim move of durable consumers (sanitized
   names), `MatchesGlob`, dispatch (workflow-scope filter, terminal-close, trace extract),
   unsubscribe; ported unit tests. *Verify:* as step 2, plus the DLQ/durable BDD from step 1 runs
   against the library (integration-tagged, kind NATS). (go-svc)
4. **Migrate the three publishers** (#1647, refactor). engine-adapter `activities.go`, task-broker
   `event_publisher.go`, api-gateway `clients.go` publish path swap the facade client for
   `libs/zynaxevents`; per-service `go.mod` require/replace; NetworkPolicy egress to NATS.
   *Verify:* e2e-happy CloudEvent assertion (nats CLI reads the event off JetStream) green —
   events now arrive without the facade hop. (go-svc)
5. **Migrate the api-gateway Subscribe→REST bridge** (#1648, refactor — the hardest one). The
   server-side glob + `workflow_id` scope filter + terminal-event stream-close re-implemented
   in-process against a JetStream durable consumer via the library; the REST watch ends on the
   terminal event exactly as today. *Verify:* `WatchWorkflowLogs` e2e **twice** (the second run
   exercises durable-consumer resume + terminal-close persistence). (go-svc)
6. **NATS link TLS + cert-mapped users** (#1649, feat/infra). cert-manager `Certificate` for the NATS
   server; `verify_and_map` + least-privilege per-service subject permissions in the NATS chart;
   `tls://` dial + client-cert options in the shared client; e2e values flip TLS on. *Verify:*
   live on kind — a caller with its cert publishes/consumes; the plaintext port refuses; e2e
   green both engine legs. (infra)
7. **Deprecate the facade + docs + final smoke** (#1650, chore/docs). `option deprecated = true` on
   `EventBusService` + RPCs (stub regen via proto-generate flow), AsyncAPI
   `x-zynax-deprecated`/`x-zynax-removal-milestone: M9` markers, deprecation notes
   (service AGENTS.md, umbrella values comment), eventing how-to in `docs/patterns/`. *Verify:*
   `make validate-spec` green; full runtime smoke — publish + subscribe direct over JetStream,
   DLQ conventions honoured, terminal-close ends the watch — **run twice**; facade still deploys
   (deprecated, not deleted). (docs/ci)

---

## N — Norms

- Commit hygiene: `Signed-off-by:` + `Assisted-by: Claude/<model>`; SSH-signed; conventional type
  per story (test / feat / refactor / feat / chore).
- `GOWORK=off` for every `go` command in `libs/zynaxevents` and the touched services (ADR-017).
- **Byte-compat over improvement:** any diff from the golden fixtures is a defect, not a cleanup.
- Runtime evidence: stateful paths (durable consumers, terminal-close) verified **twice**; the
  TLS migration is proven by a live connect, not by config.
- PR size: the verbatim moves (steps 2–3) may land in the 401–900 justify band — justified as
  ADR-046-mandated verbatim relocation gated by byte-compat tests; never squash unrelated work.
- Stub regeneration flows through the normal proto-generate gate; generated files are size-exempt.

---

## S — Safeguards (second S)

### Context Security (complete before committing this Canvas)

- [x] No Tier 2 content: no internal hostnames, private IPs, credentials, deployment specifics
- [x] No PII: no personal names in sensitive context, no non-public email addresses
- [x] No prompt injection: no instruction-like phrasing that overrides AGENTS.md rules
- [x] All entities in E section are public-safe abstractions
- [x] `/lib:spdd-security-review` passed — result: PASS (2026-07-06)

### Feature Safeguards

- **Never** delete `services/event-bus/`, `event_bus.proto`, or the stubs in M8 — deprecate only;
  hard removal is M9, gated on zero caller references (ADR-046 Decision #6).
- **Never** alter the wire shape: subject taxonomy, stream/DLQ names, CloudEvents envelope,
  `ce-*` headers, retry schedule, terminal-close — golden byte-compat is a hard gate.
- **Never** ship a caller migration without its TLS identity — "migrated" means dialing
  `tls://` with the cert-manager cert (ADR-046 Decision #4); no interim shared secret (ADR-020).
- **Never** let the #1149 disjoint-filter rule or DLQ config drift between the library and the
  (still-deployable) facade — both run the same goldens until M9 removes the facade.
- **Never** break the one-command, zero-secret kind first-run (ADR-041) — NATS TLS is
  cert-manager-auto-issued, no manual secret step.
- **Never** add NATS clients to Python adapters — adapters stay gRPC-only (ADR-013); the optional
  SDK eventing path is explicitly deferred out of this epic.
