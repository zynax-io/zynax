// SPDX-License-Identifier: Apache-2.0

//go:build integration

// Package zynaxevents_bdd_test provides BDD contract tests for the shared
// JetStream events client (libs/zynaxevents, ADR-046 M8.H) driven by godog
// against a real NATS JetStream server (testcontainers-go). The suite is the
// ported event-bus facade suite — same scenarios, same semantics — plus the
// #1149 disjoint-stream and workflow-scoped terminal-close contracts.
// Run with: GOWORK=off go test -tags integration -v -timeout 300s ./tests/...
package zynaxevents_bdd_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/cucumber/godog"
	nats "github.com/nats-io/nats.go"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/zynax-io/zynax/libs/zynaxevents"
)

// ── test harness ─────────────────────────────────────────────────────────────

type ebSuite struct {
	natsURL    string
	bus        *zynaxevents.Client
	channels   map[string]<-chan zynaxevents.CloudEvent
	cancelFn   map[string]context.CancelFunc
	topic      string
	event      zynaxevents.CloudEvent
	ackStreams []string
	wfCh       <-chan zynaxevents.CloudEvent
	advCh      chan *nats.Msg
	advSub     *nats.Subscription
	advMsg     *nats.Msg
	nc         *nats.Conn
	js         nats.JetStreamContext
}

func startNATS(t *testing.T) (string, func()) {
	t.Helper()
	ctx := context.Background()
	ctr, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "nats:2.10-alpine",
			Cmd:          []string{"-js"},
			ExposedPorts: []string{"4222/tcp"},
			WaitingFor:   wait.ForLog("Server is ready"),
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("start nats: %v", err)
	}
	host, err := ctr.Host(ctx)
	if err != nil {
		t.Fatalf("nats host: %v", err)
	}
	port, err := ctr.MappedPort(ctx, "4222")
	if err != nil {
		t.Fatalf("nats port: %v", err)
	}
	url := fmt.Sprintf("nats://%s:%s", host, port.Port())
	return url, func() { _ = ctr.Terminate(ctx) }
}

func (s *ebSuite) recv(name string) (zynaxevents.CloudEvent, error) {
	ch := s.channels[name]
	if ch == nil {
		return zynaxevents.CloudEvent{}, fmt.Errorf("no channel for %q", name)
	}
	select {
	case evt, ok := <-ch:
		if !ok {
			return zynaxevents.CloudEvent{}, fmt.Errorf("channel %q closed", name)
		}
		return evt, nil
	case <-time.After(5 * time.Second):
		return zynaxevents.CloudEvent{}, fmt.Errorf("timeout waiting for %q", name)
	}
}

func (s *ebSuite) publishEvent() error {
	if _, err := s.bus.Publish(context.Background(), s.event); err != nil {
		return fmt.Errorf("publish: %w", err)
	}
	return nil
}

// ── step implementations ──────────────────────────────────────────────────────

// Scenario 1 — Published event reaches all subscribers
func (s *ebSuite) consumersSubscribeToTopic(a, b, topic string) error {
	s.topic = topic
	for _, name := range []string{a, b} {
		ctx, cancel := context.WithCancel(context.Background())
		s.cancelFn[name] = cancel
		ch, err := s.bus.Subscribe(ctx, zynaxevents.SubscribeRequest{
			SubscriberID: name, TypePattern: topic,
		})
		if err != nil {
			cancel()
			return fmt.Errorf("subscribe %s: %w", name, err)
		}
		s.channels[name] = ch
	}
	return nil
}

func (s *ebSuite) eventPublishedToThatTopic() error {
	s.event = zynaxevents.CloudEvent{
		ID: "bdd-evt-1", Source: "bdd", SpecVersion: "1.0", Type: s.topic, Data: []byte(`{}`),
	}
	return s.publishEvent()
}

func (s *ebSuite) bothReceiveEvent(a, b string) error {
	for _, name := range []string{a, b} {
		evt, err := s.recv(name)
		if err != nil {
			return err
		}
		if evt.Type != s.topic {
			return fmt.Errorf("%s: got type %q want %q", name, evt.Type, s.topic)
		}
	}
	return nil
}

// Scenario 2 — Subscriber on different topic does not receive event
func (s *ebSuite) consumerSubscribesTo(c, topic string) error {
	ctx, cancel := context.WithCancel(context.Background())
	s.cancelFn[c] = cancel
	ch, err := s.bus.Subscribe(ctx, zynaxevents.SubscribeRequest{
		SubscriberID: c, TypePattern: topic,
	})
	if err != nil {
		cancel()
		return fmt.Errorf("subscribe %s: %w", c, err)
	}
	s.channels[c] = ch
	return nil
}

func (s *ebSuite) eventPublishedTo(topic string) error {
	s.topic = topic
	s.event = zynaxevents.CloudEvent{
		ID: "bdd-evt-2", Source: "bdd", SpecVersion: "1.0", Type: topic, Data: []byte(`{}`),
	}
	return s.publishEvent()
}

func (s *ebSuite) consumerDoesNotReceiveEvent(c string) error {
	ch := s.channels[c]
	if ch == nil {
		return fmt.Errorf("no channel for %q", c)
	}
	select {
	case evt := <-ch:
		return fmt.Errorf("%s unexpectedly received event type %q", c, evt.Type)
	case <-time.After(500 * time.Millisecond):
		return nil
	}
}

// Scenario 3 — Failed delivery is retried with backoff
func (s *ebSuite) subscriberThatFailsOnFirstAttempt() error {
	zynaxevents.RetryBackoff = []time.Duration{
		50 * time.Millisecond, 100 * time.Millisecond,
		150 * time.Millisecond, 200 * time.Millisecond, 250 * time.Millisecond,
	}
	s.topic = "zynax.v1.bdd.retry.event"
	s.event = zynaxevents.CloudEvent{
		ID: "bdd-retry-1", Source: "bdd", SpecVersion: "1.0", Type: s.topic, Data: []byte(`{}`),
	}
	// Ensure stream exists by subscribing and immediately unsubscribing.
	ctx, cancel := context.WithCancel(context.Background())
	ch, err := s.bus.Subscribe(ctx, zynaxevents.SubscribeRequest{
		SubscriberID: "retry-setup", TypePattern: s.topic,
	})
	if err != nil {
		cancel()
		return fmt.Errorf("retry setup subscribe: %w", err)
	}
	cancel()
	for {
		if _, ok := <-ch; !ok {
			break
		}
	}
	return nil
}

func (s *ebSuite) anEventIsPublished() error {
	return s.publishEvent()
}

func (s *ebSuite) eventIsRedeliveredAtLeastOnce() error {
	streamName := zynaxevents.StreamName(s.topic)
	subject := zynaxevents.SubjectFilter(s.topic)
	dur := zynaxevents.DurableConsumerName("bdd-redeliver")
	backoff := []time.Duration{50 * time.Millisecond, 100 * time.Millisecond}

	sub, err := s.js.SubscribeSync(
		subject, nats.Durable(dur),
		nats.DeliverAll(), nats.AckExplicit(),
		nats.MaxDeliver(3), nats.BackOff(backoff),
		nats.Bind(streamName, dur),
	)
	if err != nil {
		sub, err = s.js.SubscribeSync(
			subject, nats.Durable(dur),
			nats.DeliverAll(), nats.AckExplicit(),
			nats.MaxDeliver(3), nats.BackOff(backoff),
		)
		if err != nil {
			return fmt.Errorf("subscribe bdd-redeliver: %w", err)
		}
	}
	defer func() { _ = sub.Unsubscribe() }()

	msg1, err := sub.NextMsg(3 * time.Second)
	if err != nil {
		return fmt.Errorf("first delivery: %w", err)
	}
	_ = msg1.Nak()

	msg2, err := sub.NextMsg(3 * time.Second)
	if err != nil {
		return fmt.Errorf("redelivery not received: %w", err)
	}
	_ = msg2.Ack()
	return nil
}

// Scenario 4 — Event is DLQ'd after exhausting retries
func (s *ebSuite) subscriberThatAlwaysFails() error {
	zynaxevents.RetryBackoff = []time.Duration{
		50 * time.Millisecond, 100 * time.Millisecond,
		150 * time.Millisecond, 200 * time.Millisecond, 250 * time.Millisecond,
	}
	s.topic = "zynax.v1.bdd.dlq.event"
	s.event = zynaxevents.CloudEvent{
		ID: "bdd-dlq-1", Source: "bdd", SpecVersion: "1.0", Type: s.topic, Data: []byte(`{}`),
	}
	// Watch for the JetStream max-deliveries advisory the exhaustion emits.
	advCh := make(chan *nats.Msg, 4)
	advSub, err := s.nc.ChanSubscribe("$JS.EVENT.ADVISORY.CONSUMER.MAX_DELIVERIES.>", advCh)
	if err != nil {
		return fmt.Errorf("advisory subscribe: %w", err)
	}
	s.advSub = advSub
	s.advCh = advCh
	ctx, cancel := context.WithCancel(context.Background())
	ch, err := s.bus.Subscribe(ctx, zynaxevents.SubscribeRequest{
		SubscriberID: "dlq-setup", TypePattern: s.topic,
	})
	if err != nil {
		cancel()
		return fmt.Errorf("dlq setup subscribe: %w", err)
	}
	cancel()
	for {
		if _, ok := <-ch; !ok {
			break
		}
	}
	return nil
}

func (s *ebSuite) deliveryAttemptsExhausted(_ int) error {
	streamName := zynaxevents.StreamName(s.topic)
	subject := zynaxevents.SubjectFilter(s.topic)
	dur := zynaxevents.DurableConsumerName("bdd-always-fail")
	backoff := zynaxevents.RetryBackoff

	sub, err := s.js.SubscribeSync(
		subject, nats.Durable(dur),
		nats.DeliverAll(), nats.AckExplicit(),
		nats.MaxDeliver(5), nats.BackOff(backoff),
		nats.Bind(streamName, dur),
	)
	if err != nil {
		sub, err = s.js.SubscribeSync(
			subject, nats.Durable(dur),
			nats.DeliverAll(), nats.AckExplicit(),
			nats.MaxDeliver(5), nats.BackOff(backoff),
		)
		if err != nil {
			return fmt.Errorf("subscribe always-fail: %w", err)
		}
	}
	// The "an event is published" step already published the event — the
	// facade's step published a second copy here, splitting the NAK budget
	// across two messages so neither ever reached MaxDeliver (part of why
	// the facade suite was red). NAK the single message to exhaustion.
	for attempt := 0; attempt < 5; attempt++ {
		msg, err := sub.NextMsg(3 * time.Second)
		if err != nil {
			break // MaxDeliver exhausted — expected
		}
		delay := backoff[min(attempt, len(backoff)-1)]
		_ = msg.NakWithDelay(delay)
	}

	// The advisory fires when the server would redeliver past MaxDeliver —
	// which needs the final nak delay to elapse while the consumer still
	// exists. Wait for it BEFORE unsubscribing (deleting the durable consumer
	// beforehand suppresses the advisory — the other reason the facade's
	// scenario could never pass).
	select {
	case msg := <-s.advCh:
		s.advMsg = msg
	case <-time.After(8 * time.Second):
	}
	_ = sub.Unsubscribe()
	return nil
}

func (s *ebSuite) maxDeliveriesAdvisoryEmitted() error {
	if s.advMsg == nil || len(s.advMsg.Data) == 0 {
		return fmt.Errorf("no max-deliveries advisory observed during exhaustion")
	}
	return nil
}

func (s *ebSuite) dlqStreamProvisioned() error {
	dlqName := "DLQ_" + zynaxevents.StreamName(s.topic)
	info, err := s.js.StreamInfo(dlqName)
	if err != nil {
		return fmt.Errorf("DLQ stream %s not provisioned: %w", dlqName, err)
	}
	if info.Config.Retention != nats.WorkQueuePolicy {
		return fmt.Errorf("DLQ stream %s retention = %v, want WorkQueuePolicy", dlqName, info.Config.Retention)
	}
	return nil
}

// Scenario 5 — Durable consumer catches up after being offline
func (s *ebSuite) consumerWasOfflineWhenEventPublished(d string) error {
	s.topic = "zynax.v1.bdd.durable.event"
	ctx, cancel := context.WithCancel(context.Background())
	ch, err := s.bus.Subscribe(ctx, zynaxevents.SubscribeRequest{
		SubscriberID: d, TypePattern: s.topic,
	})
	if err != nil {
		cancel()
		return fmt.Errorf("subscribe %s: %w", d, err)
	}
	// Go offline: cancel context, drain channel.
	cancel()
	for {
		if _, ok := <-ch; !ok {
			break
		}
	}

	// Publish while offline.
	s.event = zynaxevents.CloudEvent{
		ID: "bdd-durable-1", Source: "bdd", SpecVersion: "1.0",
		Type: s.topic, Data: []byte(`{"catch_up":true}`),
	}
	if _, err = s.bus.Publish(context.Background(), s.event); err != nil {
		return fmt.Errorf("publish while offline: %w", err)
	}
	return nil
}

func (s *ebSuite) consumerReconnects(d string) error {
	ctx, cancel := context.WithCancel(context.Background())
	s.cancelFn[d] = cancel
	ch, err := s.bus.Subscribe(ctx, zynaxevents.SubscribeRequest{
		SubscriberID: d, TypePattern: s.topic,
	})
	if err != nil {
		cancel()
		return fmt.Errorf("resubscribe %s: %w", d, err)
	}
	s.channels[d] = ch
	return nil
}

func (s *ebSuite) consumerReceivesMissedEvent(d string) error {
	evt, err := s.recv(d)
	if err != nil {
		return err
	}
	if evt.Type != s.topic {
		return fmt.Errorf("%s: got type %q want %q", d, evt.Type, s.topic)
	}
	return nil
}

// Scenario 6 — Two consumer groups receive same event independently.
// (The facade's step took a third "topic" arg its regex never captured — a
// pre-existing arity bug that kept the suite red; fixed in the port.)
func (s *ebSuite) groupsSubscribeToSameTopic(g1, g2 string) error {
	s.topic = "zynax.v1.bdd.groups.event"
	for _, g := range []string{g1, g2} {
		ctx, cancel := context.WithCancel(context.Background())
		s.cancelFn[g] = cancel
		ch, err := s.bus.Subscribe(ctx, zynaxevents.SubscribeRequest{
			SubscriberID: g, TypePattern: s.topic,
		})
		if err != nil {
			cancel()
			return fmt.Errorf("subscribe group %s: %w", g, err)
		}
		s.channels[g] = ch
	}
	return nil
}

func (s *ebSuite) oneEventIsPublished() error {
	s.event = zynaxevents.CloudEvent{
		ID: "bdd-grp-1", Source: "bdd", SpecVersion: "1.0", Type: s.topic, Data: []byte(`{}`),
	}
	return s.publishEvent()
}

// (Another inherited arity bug: the facade registered this 2-arg func on a
// capture-less regex. The group names are fixed by the scenario text.)
func (s *ebSuite) bothGroupsReceiveIndependentCopy() error {
	return s.bothReceiveEvent("indexer", "notifier")
}

// Scenario 7 — #1149: events under one entity prefix share a single stream
func (s *ebSuite) eventOfTypeIsPublished(eventType string) error {
	ack, err := s.bus.Publish(context.Background(), zynaxevents.CloudEvent{
		ID: "bdd-1149-" + eventType, Source: "bdd", SpecVersion: "1.0",
		Type: eventType, Data: []byte(`{}`),
	})
	if err != nil {
		return fmt.Errorf("publish %s: %w", eventType, err)
	}
	// Publish acks are "STREAM:sequence" — record the stream half.
	for i := range ack {
		if ack[i] == ':' {
			s.ackStreams = append(s.ackStreams, ack[:i])
			return nil
		}
	}
	return fmt.Errorf("malformed publish ack %q", ack)
}

func (s *ebSuite) bothEventsLandOnStream(stream string) error {
	if len(s.ackStreams) < 2 {
		return fmt.Errorf("expected 2 recorded publish acks, got %d", len(s.ackStreams))
	}
	for _, got := range s.ackStreams {
		if got != stream {
			return fmt.Errorf("event landed on stream %q, want %q", got, stream)
		}
	}
	return nil
}

func (s *ebSuite) noSubjectsOverlapError() error {
	// Publish errors would have surfaced in the publish steps; nothing to do.
	return nil
}

// Scenarios 8/9 — workflow-scoped terminal-close vs wildcard stays open
func (s *ebSuite) subscriberScopedToWorkflow(workflowID, pattern string) error {
	ctx, cancel := context.WithCancel(context.Background())
	s.cancelFn["wf-scoped"] = cancel
	ch, err := s.bus.Subscribe(ctx, zynaxevents.SubscribeRequest{
		SubscriberID: "bdd-wf-scoped", TypePattern: pattern, WorkflowID: workflowID,
	})
	if err != nil {
		cancel()
		return fmt.Errorf("scoped subscribe: %w", err)
	}
	s.wfCh = ch
	return nil
}

func (s *ebSuite) subscriberWithPatternNoScope(pattern string) error {
	ctx, cancel := context.WithCancel(context.Background())
	s.cancelFn["wf-wild"] = cancel
	ch, err := s.bus.Subscribe(ctx, zynaxevents.SubscribeRequest{
		SubscriberID: "bdd-wf-wild", TypePattern: pattern,
	})
	if err != nil {
		cancel()
		return fmt.Errorf("wildcard subscribe: %w", err)
	}
	s.wfCh = ch
	return nil
}

func (s *ebSuite) terminalEventForWorkflowPublished(eventType, workflowID string) error {
	if _, err := s.bus.Publish(context.Background(), zynaxevents.CloudEvent{
		ID: "bdd-term-" + workflowID, Source: "bdd", SpecVersion: "1.0",
		Type: eventType, WorkflowID: workflowID, Data: []byte(`{}`),
	}); err != nil {
		return fmt.Errorf("publish terminal event: %w", err)
	}
	return nil
}

func (s *ebSuite) subscriberReceivesTerminalEvent() error {
	select {
	case evt, ok := <-s.wfCh:
		if !ok {
			return fmt.Errorf("channel closed before delivering the terminal event")
		}
		if !zynaxevents.IsTerminalEventType(evt.Type) {
			return fmt.Errorf("received non-terminal event %q", evt.Type)
		}
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("timeout waiting for the terminal event")
	}
}

func (s *ebSuite) eventChannelIsClosed() error {
	select {
	case _, ok := <-s.wfCh:
		if ok {
			return fmt.Errorf("received an extra event; channel should be closed")
		}
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("channel not closed after the terminal event")
	}
}

func (s *ebSuite) eventChannelRemainsOpen() error {
	select {
	case _, ok := <-s.wfCh:
		if !ok {
			return fmt.Errorf("channel closed — a wildcard subscription must stay open")
		}
		return fmt.Errorf("unexpected extra event on the wildcard channel")
	case <-time.After(500 * time.Millisecond):
		return nil // still open, nothing more queued
	}
}

// ── godog wiring ──────────────────────────────────────────────────────────────

func (s *ebSuite) initScenario(sc *godog.ScenarioContext) {
	sc.Step(`^consumers "([^"]*)" and "([^"]*)" subscribe to topic "([^"]*)"$`, s.consumersSubscribeToTopic)
	sc.Step(`^an event is published to that topic$`, s.eventPublishedToThatTopic)
	sc.Step(`^both "([^"]*)" and "([^"]*)" receive the event$`, s.bothReceiveEvent)
	sc.Step(`^consumer "([^"]*)" subscribes to "([^"]*)"$`, s.consumerSubscribesTo)
	sc.Step(`^an event is published to "([^"]*)"$`, s.eventPublishedTo)
	sc.Step(`^consumer "([^"]*)" does NOT receive the event$`, s.consumerDoesNotReceiveEvent)
	sc.Step(`^a subscriber that fails on first attempt$`, s.subscriberThatFailsOnFirstAttempt)
	sc.Step(`^an event is published$`, s.anEventIsPublished)
	sc.Step(`^the event is redelivered at least once$`, s.eventIsRedeliveredAtLeastOnce)
	sc.Step(`^a subscriber that always fails$`, s.subscriberThatAlwaysFails)
	sc.Step(`^(\d+) delivery attempts are exhausted$`, s.deliveryAttemptsExhausted)
	sc.Step(`^a max-deliveries advisory is emitted for the consumer$`, s.maxDeliveriesAdvisoryEmitted)
	sc.Step(`^the DLQ stream for the topic exists with WorkQueuePolicy retention$`, s.dlqStreamProvisioned)
	sc.Step(`^consumer "([^"]*)" was offline when an event was published$`, s.consumerWasOfflineWhenEventPublished)
	sc.Step(`^consumer "([^"]*)" reconnects$`, s.consumerReconnects)
	sc.Step(`^consumer "([^"]*)" receives the missed event$`, s.consumerReceivesMissedEvent)
	sc.Step(`^groups "([^"]*)" and "([^"]*)" both subscribe to the same topic$`, s.groupsSubscribeToSameTopic)
	sc.Step(`^one event is published$`, s.oneEventIsPublished)
	sc.Step(`^both groups receive their own independent copy$`, s.bothGroupsReceiveIndependentCopy)
	sc.Step(`^an event of type "([^"]*)" is published$`, s.eventOfTypeIsPublished)
	sc.Step(`^both events land on stream "([^"]*)"$`, s.bothEventsLandOnStream)
	sc.Step(`^no "subjects overlap with an existing stream" error occurs$`, s.noSubjectsOverlapError)
	sc.Step(`^a subscriber scoped to workflow "([^"]*)" with pattern "([^"]*)"$`, s.subscriberScopedToWorkflow)
	sc.Step(`^a subscriber with pattern "([^"]*)" and no workflow scope$`, s.subscriberWithPatternNoScope)
	sc.Step(`^a "([^"]*)" event for workflow "([^"]*)" is published$`, s.terminalEventForWorkflowPublished)
	sc.Step(`^the subscriber receives the terminal event$`, s.subscriberReceivesTerminalEvent)
	sc.Step(`^the event channel is closed$`, s.eventChannelIsClosed)
	sc.Step(`^the event channel remains open$`, s.eventChannelRemainsOpen)
}

// ── test runner ───────────────────────────────────────────────────────────────

func TestFeatures(t *testing.T) {
	natsURL, cleanup := startNATS(t)
	defer cleanup()

	bus, err := zynaxevents.New(natsURL)
	if err != nil {
		t.Fatalf("connect bus: %v", err)
	}
	defer bus.Close()

	nc, err := nats.Connect(natsURL)
	if err != nil {
		t.Fatalf("raw nats: %v", err)
	}
	defer nc.Close()
	js, err := nc.JetStream()
	if err != nil {
		t.Fatalf("jetstream: %v", err)
	}

	suite := godog.TestSuite{
		Name: "zynaxevents",
		ScenarioInitializer: func(sc *godog.ScenarioContext) {
			s := &ebSuite{
				natsURL:  natsURL,
				bus:      bus,
				channels: make(map[string]<-chan zynaxevents.CloudEvent),
				cancelFn: make(map[string]context.CancelFunc),
				nc:       nc,
				js:       js,
			}
			s.initScenario(sc)
			sc.After(func(ctx context.Context, _ *godog.Scenario, _ error) (context.Context, error) {
				for _, cancel := range s.cancelFn {
					cancel()
				}
				if s.advSub != nil {
					_ = s.advSub.Unsubscribe()
				}
				return ctx, nil
			})
		},
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"features"},
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("godog suite failed")
	}
}
