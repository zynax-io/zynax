<!-- SPDX-License-Identifier: Apache-2.0 -->

# Eventing: direct JetStream via the shared client

Zynax services publish and consume platform events **directly over NATS
JetStream** through the shared client — `libs/zynaxevents` (Go) — not through
a broker-fronting service (ADR-046). The `EventBusService` gRPC facade was
removed in M9 (EPIC #1675); this is now the only eventing path. The **AsyncAPI
spec (`spec/asyncapi/zynax-events.yaml`) remains the single contract of
record**, realised by the client library.

## How it fits together

```
engine-adapter ─┐
task-broker    ─┼─ libs/zynaxevents ──TLS──▶ NATS JetStream
api-gateway    ─┘   (publish + the /logs      · verify_and_map client identities
                     Subscribe→REST bridge)   · per-identity subject permissions
                                              · streams derived on demand
```

- **Conventions ride the client, verbatim from the facade** (golden
  byte-compat gated in `libs/zynaxevents/testdata/golden/`): depth-4 stream
  derivation with the #1149 disjoint-filter rule, `DLQ_<src>` /
  `zynax.dlq.<prefix>.dead` / WorkQueuePolicy dead-letter provisioning,
  `MaxDeliver=5` with the ascending retry backoff, durable-name sanitizing,
  `*`/`**` glob subscription matching, the CloudEvents v1.0 JSON envelope with
  `ce-*` headers, and trace inject/extract over the async hop.
- **Workflow-scoped terminal-close:** a subscription scoped to a
  `workflow_id` delivers the run's terminal lifecycle event and then closes —
  this is what ends a `zynax logs` REST watch.
- The `zynax.dlq.` prefix is **reserved**; end-to-end DLQ forwarding is
  delivered by the opt-in `DLQForwarder` (#1653, below) — the conventions
  provision the DLQ stream and stop redelivery at MaxDeliver, and the forwarder
  moves the exhausted message into it.

## Publishing (Go)

```go
import "github.com/zynax-io/zynax/libs/zynaxevents"

opts := []nats.Option{nats.RetryOnFailedConnect(true), nats.MaxReconnects(-1)}
if tlsCert != "" { // the service's cert-manager identity (same PEMs as gRPC mTLS)
    opts = append(opts, zynaxevents.TLSIdentity(tlsCert, tlsKey, tlsCA)...)
}
client, err := zynaxevents.New(natsURL, opts...)
// ...
ack, err := client.Publish(ctx, zynaxevents.CloudEvent{
    ID: id, Source: "/zynax/<service>/...", SpecVersion: "1.0",
    Type: "zynax.v1.<service>.<entity>.<verb>", Data: payload,
})
```

Publishes are **best-effort by convention**: log-and-continue on error, never
fail the caller's state machine. `RetryOnFailedConnect` keeps startup
broker-independent — profiles without NATS (ADR-041 lite) still boot.

## Dead-lettering: provisioning vs forwarding (#1653)

Two halves, deliberately separate:

1. **Provisioning happens for everyone.** `Subscribe` idempotently creates
   `DLQ_<src>` (WorkQueuePolicy, one exact `zynax.dlq.<prefix>.dead` subject)
   and caps redelivery at `MaxDeliver=5`.
2. **Moving the exhausted message is opt-in.** `Subscribe` does NOT start a
   mover. The max-deliveries advisory subject is server-global, so a mover
   started implicitly per subscription would act on streams its caller has
   nothing to do with, and would need grants ordinary publishers do not have —
   turning a library upgrade into a permission-error regression for every
   subscriber. Forwarding is a platform role, taken on explicitly:

```go
fwd, err := client.StartDLQForwarder(ctx, logger) // logger may be nil
defer fwd.Stop()
```

The forwarder consumes `$JS.EVENT.ADVISORY.CONSUMER.MAX_DELIVERIES.>`, fetches
the exhausted message from its source stream by sequence, and republishes it
byte-for-byte on that stream's reserved deliver subject, stamped with
`Zynax-Dlq-Source-*` forensic headers. Its safety properties:

- **It never deletes the source message** — a failed rescue leaves the message
  exactly where it was, so the mover cannot lose what it exists to save.
- **It is idempotent.** The DLQ publish carries a deterministic
  `DLQ_<stream>:<sequence>` `Nats-Msg-Id`, so JetStream's duplicate window
  collapses a replayed advisory, a retried publish, and a second forwarder onto
  one message. Replicas additionally share one NATS queue group.
- **It only moves what the taxonomy derived.** An advisory whose stream is not
  `StreamName(<message subject>)` is skipped rather than published to a
  computed — and therefore wrong — subject.

Running it in-cluster needs the identity's NATS permissions widened beyond the
publisher baseline: subscribe `$JS.EVENT.ADVISORY.CONSUMER.MAX_DELIVERIES.>`
and publish `zynax.dlq.>` (see the users block in `scripts/e2e/values-e2e.yaml`).
No service opts in yet — the contract is exercised by the
`libs/zynaxevents/tests` BDD suite against a real broker.

## Transport security (ADR-046 Decision #4)

The broker requires TLS client certificates and maps each to a NATS user by
subject DN (`verify_and_map`); per-identity **subject permissions** scope each
service to its own taxonomy prefix plus the JetStream API. Identities are the
same cert-manager certificates the gRPC mTLS mesh mounts (ADR-020) — no second
secret system, no manual secret step on kind first-run (ADR-041). The policy
lives in the NATS chart values (see `scripts/e2e/values-e2e.yaml`); a caller
without a certificate fails the TLS handshake.

## Topic taxonomy

`zynax.<version>.<service>.<entity>.<event_type>` — streams are derived from
the first four segments (`StreamName`), so all events under one entity share
one stream and subject filters can never overlap (#1149). New event types must
follow the taxonomy and be added to the AsyncAPI spec (`make validate-spec`).

## Related

- ADR-046 — the retirement decision, migration order, M9 removal clause
- ADR-022 — the original facade architecture (amended by ADR-046)
- Canvas: `docs/spdd/1576-direct-nats-jetstream/canvas.md`
- Troubleshooting: `docs/troubleshooting.md` §5–6
