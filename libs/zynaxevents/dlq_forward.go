// SPDX-License-Identifier: Apache-2.0

package zynaxevents

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	nats "github.com/nats-io/nats.go"
)

// maxDeliveriesAdvisorySubject is the JetStream system subject on which the
// server announces that a consumer exhausted MaxDeliver for one message. It is
// core NATS (not itself a stream), so a plain subscription is enough.
const maxDeliveriesAdvisorySubject = "$JS.EVENT.ADVISORY.CONSUMER.MAX_DELIVERIES.>"

// dlqForwarderQueueGroup makes every forwarder instance a member of one NATS
// queue group: with N replicas running the forwarder the server hands each
// advisory to exactly ONE of them. Combined with the deterministic publish
// message id (see dlqMessageID) this keeps a rescue single-shot even when the
// group membership changes mid-advisory.
const dlqForwarderQueueGroup = "zynax-dlq-forwarder"

// dlqStreamPrefix is the name prefix of every dead-letter stream. Advisories
// about consumers ON a DLQ stream are ignored — a DLQ is a terminus, never a
// source, and forwarding one would build a cycle.
const dlqStreamPrefix = "DLQ_"

// natsHeaderPrefix marks the JetStream-reserved header namespace. Source
// headers under it are dropped when copying a message to the DLQ so a
// publisher-set "Nats-Msg-Id" can never displace the forwarder's own
// deduplication key.
const natsHeaderPrefix = "Nats-"

// Forensic headers stamped onto every message the forwarder moves. They record
// where the message came from and why it was moved, so an operator draining a
// DLQ can trace it back without correlating logs. The CloudEvent body itself is
// copied byte-for-byte and is never rewritten.
const (
	HeaderDLQSourceStream   = "Zynax-Dlq-Source-Stream"
	HeaderDLQSourceSubject  = "Zynax-Dlq-Source-Subject"
	HeaderDLQSourceSequence = "Zynax-Dlq-Source-Sequence"
	HeaderDLQConsumer       = "Zynax-Dlq-Consumer"
	HeaderDLQDeliveries     = "Zynax-Dlq-Deliveries"
)

const (
	// dlqAdvisoryBuffer bounds how many advisories may queue ahead of the
	// forwarding worker before NATS reports a slow consumer.
	dlqAdvisoryBuffer = 256
	// dlqPublishAttempts is the bounded retry budget for the DLQ publish. The
	// source message is never deleted, so exhausting the budget leaves the
	// message where it already was — a failed rescue never loses data.
	dlqPublishAttempts = 3
	// dlqPublishRetryDelay is the pause between DLQ publish attempts.
	dlqPublishRetryDelay = 250 * time.Millisecond
)

// maxDeliveriesAdvisory is the JetStream max-deliveries advisory payload. Note
// that the subject says MAX_DELIVERIES while the schema type says
// "io.nats.jetstream.advisory.v1.max_deliver" — the mover therefore routes on
// the SUBJECT and only records the type, so a schema rename cannot silently
// stop it. Only the fields the forwarder acts on are decoded; unknown fields
// are ignored by design so a newer server does not break the mover.
type maxDeliveriesAdvisory struct {
	Type       string `json:"type"`
	Stream     string `json:"stream"`
	Consumer   string `json:"consumer"`
	StreamSeq  uint64 `json:"stream_seq"`
	Deliveries uint64 `json:"deliveries"`
}

// DLQForwarderStats reports what a running forwarder has done. Counters are
// cumulative since Start and safe to read concurrently.
type DLQForwarderStats struct {
	// Forwarded counts messages the DLQ stream accepted as new.
	Forwarded uint64
	// Deduplicated counts rescues JetStream collapsed onto a message already in
	// the DLQ — a replayed advisory, a retried publish, or a second forwarder.
	Deduplicated uint64
	// Skipped counts advisories deliberately not acted on: DLQ-stream
	// advisories, a source message that no longer exists, or a stream whose
	// topology these conventions did not provision.
	Skipped uint64
	// Failed counts advisories whose rescue could not be completed. The source
	// message is untouched in every such case.
	Failed uint64
}

// DLQForwarder moves messages that exhausted MaxDeliver into their stream's
// dead-letter queue.
//
// It is deliberately OPT-IN: Subscribe does not start one. The advisory subject
// is server-global, so a forwarder started implicitly per subscription would
// (a) act on streams its caller has nothing to do with, and (b) require the
// advisory-subscribe and "zynax.dlq.>" publish grants that the per-identity
// NATS policy (ADR-046 Decision #4) does not give ordinary publishers — turning
// a library upgrade into a permission-error regression for every subscriber.
// Forwarding is a platform role: the deployment that takes it on is the one
// that gets the extra grants.
//
// Guarantees:
//   - It never deletes the source message. A failed forward leaves the message
//     exactly where it was; it cannot lose the message it exists to rescue.
//   - Forwarding is idempotent. The DLQ publish carries a deterministic
//     "<stream>:<sequence>" message id, so JetStream deduplicates a repeated
//     advisory, a retried publish, and a concurrent second forwarder alike.
type DLQForwarder struct {
	client    *Client
	sub       *nats.Subscription
	msgs      chan *nats.Msg
	log       *slog.Logger
	stop      chan struct{}
	done      chan struct{}
	stopOnce  sync.Once
	forwarded atomic.Uint64
	deduped   atomic.Uint64
	skipped   atomic.Uint64
	failed    atomic.Uint64
}

// StartDLQForwarder subscribes to the JetStream max-deliveries advisory and
// begins moving exhausted messages into their DLQ_<source> stream. It returns
// once the subscription is live; forwarding runs on its own goroutine until
// ctx is cancelled or Stop is called. A nil logger falls back to slog.Default.
//
// The caller needs a NATS identity permitted to subscribe to
// "$JS.EVENT.ADVISORY.CONSUMER.MAX_DELIVERIES.>" and to publish on
// "zynax.dlq.>" in addition to the usual "$JS.API.>".
func (c *Client) StartDLQForwarder(ctx context.Context, logger *slog.Logger) (*DLQForwarder, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("context: %w", err)
	}
	if logger == nil {
		logger = slog.Default()
	}

	f := &DLQForwarder{
		client: c,
		msgs:   make(chan *nats.Msg, dlqAdvisoryBuffer),
		log:    logger,
		stop:   make(chan struct{}),
		done:   make(chan struct{}),
	}

	sub, err := c.conn.ChanQueueSubscribe(maxDeliveriesAdvisorySubject, dlqForwarderQueueGroup, f.msgs)
	if err != nil {
		return nil, fmt.Errorf("dlq forwarder: subscribe %s: %w", maxDeliveriesAdvisorySubject, err)
	}
	f.sub = sub

	go f.run(ctx)
	return f, nil
}

// Stop unsubscribes from the advisory subject and waits for the in-flight
// forward to finish. It is idempotent and safe to call after ctx cancellation.
func (f *DLQForwarder) Stop() {
	f.stopOnce.Do(func() { close(f.stop) })
	<-f.done
}

// Stats returns a snapshot of the forwarder's cumulative counters.
func (f *DLQForwarder) Stats() DLQForwarderStats {
	return DLQForwarderStats{
		Forwarded:    f.forwarded.Load(),
		Deduplicated: f.deduped.Load(),
		Skipped:      f.skipped.Load(),
		Failed:       f.failed.Load(),
	}
}

// run drains advisories until the context is cancelled or Stop is called.
func (f *DLQForwarder) run(ctx context.Context) {
	defer close(f.done)
	defer func() { _ = f.sub.Unsubscribe() }()

	for {
		select {
		case <-ctx.Done():
			return
		case <-f.stop:
			return
		case msg := <-f.msgs:
			f.handle(msg)
		}
	}
}

// handle decodes one advisory and forwards the message it refers to.
func (f *DLQForwarder) handle(msg *nats.Msg) {
	var adv maxDeliveriesAdvisory
	if err := json.Unmarshal(msg.Data, &adv); err != nil {
		f.skipped.Add(1)
		f.log.Warn("dlq forwarder: undecodable max-deliveries advisory", "error", err)
		return
	}
	if reason, ok := advisoryNotActionable(adv); !ok {
		f.skipped.Add(1)
		f.log.Debug("dlq forwarder: advisory ignored",
			"reason", reason, "stream", adv.Stream, "advisory_type", adv.Type)
		return
	}

	if err := f.forward(adv); err != nil {
		f.failed.Add(1)
		f.log.Error("dlq forwarder: rescue failed — source message left in place",
			"stream", adv.Stream, "sequence", adv.StreamSeq, "consumer", adv.Consumer,
			"advisory_type", adv.Type, "error", err)
	}
}

// advisoryNotActionable reports whether an advisory describes a message this
// forwarder should move, plus the reason when it does not.
func advisoryNotActionable(adv maxDeliveriesAdvisory) (string, bool) {
	switch {
	case adv.Stream == "":
		return "advisory carries no stream", false
	case adv.StreamSeq == 0:
		return "advisory carries no stream sequence", false
	case strings.HasPrefix(adv.Stream, dlqStreamPrefix):
		return "source is itself a dlq stream", false
	default:
		return "", true
	}
}

// forward fetches the exhausted message from its source stream and republishes
// it on the stream's reserved DLQ deliver subject.
func (f *DLQForwarder) forward(adv maxDeliveriesAdvisory) error {
	raw, err := f.client.js.GetMsg(adv.Stream, adv.StreamSeq)
	if err != nil {
		// The message aged out of the source stream, or a WorkQueue source
		// already consumed it: there is nothing left to rescue and re-driving
		// would be wrong.
		if errors.Is(err, nats.ErrMsgNotFound) {
			f.skipped.Add(1)
			f.log.Debug("dlq forwarder: source message gone", "stream", adv.Stream, "sequence", adv.StreamSeq)
			return nil
		}
		return fmt.Errorf("get source message %s:%d: %w", adv.Stream, adv.StreamSeq, err)
	}

	if reason, ok := sourceForwardable(adv.Stream, raw.Subject); !ok {
		f.skipped.Add(1)
		f.log.Debug("dlq forwarder: source not forwardable",
			"reason", reason, "stream", adv.Stream, "subject", raw.Subject)
		return nil
	}

	// DLQ_<src> is a WorkQueuePolicy stream whose single subject is exact:
	// publishing before it exists is rejected, so provision it idempotently
	// from the SAME subject the deliver subject is derived from.
	if err = f.client.ensureDLQStream(adv.Stream, raw.Subject); err != nil {
		return fmt.Errorf("ensure dlq stream for %s: %w", adv.Stream, err)
	}

	return f.publishToDLQ(adv, raw)
}

// sourceForwardable reports whether a source subject belongs to a stream these
// conventions provisioned, plus the reason when it does not.
func sourceForwardable(streamName, subject string) (string, bool) {
	switch {
	case strings.HasPrefix(subject, reservedDLQPrefix):
		return "subject is already under the reserved dlq prefix", false
	// Both the DLQ stream name and its exact deliver subject are derived from
	// the source subject. If that derivation does not reproduce the advisory's
	// stream, the stream was not provisioned by these conventions and any
	// subject we computed would be the wrong one.
	case StreamName(subject) != streamName:
		return "stream not derived from the zynax subject taxonomy", false
	default:
		return "", true
	}
}

// publishToDLQ republishes raw on the DLQ deliver subject with a bounded retry
// budget. Every attempt carries the same deterministic message id, so a retry
// after a lost acknowledgement is deduplicated rather than duplicated.
func (f *DLQForwarder) publishToDLQ(adv maxDeliveriesAdvisory, raw *nats.RawStreamMsg) error {
	out := &nats.Msg{
		Subject: dlqDeliverSubject(raw.Subject),
		Data:    raw.Data,
		Header:  dlqHeader(raw.Header, adv, raw.Subject),
	}
	msgID := dlqMessageID(adv.Stream, adv.StreamSeq)

	var lastErr error
	for attempt := range dlqPublishAttempts {
		if attempt > 0 {
			select {
			case <-f.stop:
				return fmt.Errorf("publish to %s: stopped after %d attempts: %w", out.Subject, attempt, lastErr)
			case <-time.After(dlqPublishRetryDelay):
			}
		}
		pubAck, err := f.client.js.PublishMsg(out, nats.MsgId(msgID))
		if err != nil {
			lastErr = err
			continue
		}
		// The server reports when it collapsed this publish onto a message the
		// DLQ already holds — that is the idempotency guarantee observed, not
		// assumed.
		if pubAck.Duplicate {
			f.deduped.Add(1)
			f.log.Debug("dlq forwarder: rescue already in the dlq",
				"stream", adv.Stream, "sequence", adv.StreamSeq, "msg_id", msgID)
			return nil
		}
		f.forwarded.Add(1)
		f.log.Info("dlq forwarder: message rescued",
			"stream", adv.Stream, "sequence", adv.StreamSeq,
			"consumer", adv.Consumer, "dlq_subject", out.Subject, "msg_id", msgID)
		return nil
	}
	return fmt.Errorf("publish to %s after %d attempts: %w", out.Subject, dlqPublishAttempts, lastErr)
}

// dlqMessageID is the JetStream deduplication key for a rescued message: a pure
// function of the source stream and sequence, so every advisory replay and
// every forwarder instance computes the same id for the same message.
func dlqMessageID(streamName string, seq uint64) string {
	return dlqStreamName(streamName) + ":" + strconv.FormatUint(seq, 10)
}

// dlqHeader copies the source headers (minus the JetStream-reserved namespace)
// and stamps the forensic trail. The CloudEvent body is not touched.
func dlqHeader(src nats.Header, adv maxDeliveriesAdvisory, subject string) nats.Header {
	h := nats.Header{}
	for k, v := range src {
		if strings.HasPrefix(k, natsHeaderPrefix) {
			continue
		}
		h[k] = append([]string(nil), v...)
	}
	h.Set(HeaderDLQSourceStream, adv.Stream)
	h.Set(HeaderDLQSourceSubject, subject)
	h.Set(HeaderDLQSourceSequence, strconv.FormatUint(adv.StreamSeq, 10))
	h.Set(HeaderDLQConsumer, adv.Consumer)
	h.Set(HeaderDLQDeliveries, strconv.FormatUint(adv.Deliveries, 10))
	return h
}
