// SPDX-License-Identifier: Apache-2.0

//go:build integration

// Package event_bus_bdd_test provides BDD contract tests for event-bus service
// driven by godog against a real NATS JetStream server (testcontainers-go).
// Run with: GOWORK=off go test -tags integration -v -timeout 300s ./tests/...
package event_bus_bdd_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/cucumber/godog"
	nats "github.com/nats-io/nats.go"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/zynax-io/zynax/services/event-bus/internal/domain"
	"github.com/zynax-io/zynax/services/event-bus/internal/infrastructure"
)

// ── test harness ─────────────────────────────────────────────────────────────

type ebSuite struct {
	natsURL  string
	bus      *infrastructure.NATSEventBus
	channels map[string]<-chan domain.CloudEvent
	cancelFn map[string]context.CancelFunc
	topic    string
	event    domain.CloudEvent
	nc       *nats.Conn
	js       nats.JetStreamContext
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

func (s *ebSuite) recv(name string) (domain.CloudEvent, error) {
	ch := s.channels[name]
	if ch == nil {
		return domain.CloudEvent{}, fmt.Errorf("no channel for %q", name)
	}
	select {
	case evt, ok := <-ch:
		if !ok {
			return domain.CloudEvent{}, fmt.Errorf("channel %q closed", name)
		}
		return evt, nil
	case <-time.After(5 * time.Second):
		return domain.CloudEvent{}, fmt.Errorf("timeout waiting for %q", name)
	}
}

func (s *ebSuite) publishEvent() error {
	_, err := s.bus.Publish(context.Background(), s.event)
	return err
}

// ── step implementations ──────────────────────────────────────────────────────

// Scenario 1 — Published event reaches all subscribers
func (s *ebSuite) consumersSubscribeToTopic(a, b, topic string) error {
	s.topic = topic
	for _, name := range []string{a, b} {
		ctx, cancel := context.WithCancel(context.Background())
		s.cancelFn[name] = cancel
		ch, err := s.bus.Subscribe(ctx, domain.SubscribeRequest{
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
	s.event = domain.CloudEvent{
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
	ch, err := s.bus.Subscribe(ctx, domain.SubscribeRequest{
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
	s.event = domain.CloudEvent{
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
	infrastructure.RetryBackoff = []time.Duration{
		50 * time.Millisecond, 100 * time.Millisecond,
		150 * time.Millisecond, 200 * time.Millisecond, 250 * time.Millisecond,
	}
	s.topic = "zynax.v1.bdd.retry.event"
	s.event = domain.CloudEvent{
		ID: "bdd-retry-1", Source: "bdd", SpecVersion: "1.0", Type: s.topic, Data: []byte(`{}`),
	}
	// Ensure stream exists by subscribing and immediately unsubscribing.
	ctx, cancel := context.WithCancel(context.Background())
	ch, err := s.bus.Subscribe(ctx, domain.SubscribeRequest{
		SubscriberID: "retry-setup", TypePattern: s.topic,
	})
	if err != nil {
		cancel()
		return fmt.Errorf("retry setup subscribe: %w", err)
	}
	cancel()
	for range ch {
	}
	return nil
}

func (s *ebSuite) anEventIsPublished() error {
	return s.publishEvent()
}

func (s *ebSuite) eventIsRedeliveredAtLeastOnce() error {
	streamName := infrastructure.StreamName(s.topic)
	subject := infrastructure.SubjectFilter(s.topic)
	dur := infrastructure.DurableConsumerName("bdd-redeliver")
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
	infrastructure.RetryBackoff = []time.Duration{
		50 * time.Millisecond, 100 * time.Millisecond,
		150 * time.Millisecond, 200 * time.Millisecond, 250 * time.Millisecond,
	}
	s.topic = "zynax.v1.bdd.dlq.event"
	s.event = domain.CloudEvent{
		ID: "bdd-dlq-1", Source: "bdd", SpecVersion: "1.0", Type: s.topic, Data: []byte(`{}`),
	}
	ctx, cancel := context.WithCancel(context.Background())
	ch, err := s.bus.Subscribe(ctx, domain.SubscribeRequest{
		SubscriberID: "dlq-setup", TypePattern: s.topic,
	})
	if err != nil {
		cancel()
		return fmt.Errorf("dlq setup subscribe: %w", err)
	}
	cancel()
	for range ch {
	}
	return nil
}

func (s *ebSuite) deliveryAttemptsExhausted(_ int) error {
	streamName := infrastructure.StreamName(s.topic)
	subject := infrastructure.SubjectFilter(s.topic)
	dur := infrastructure.DurableConsumerName("bdd-always-fail")
	backoff := infrastructure.RetryBackoff

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
	defer func() { _ = sub.Unsubscribe() }()

	if _, err := s.bus.Publish(context.Background(), s.event); err != nil {
		return fmt.Errorf("publish: %w", err)
	}

	for attempt := 0; attempt < 5; attempt++ {
		msg, err := sub.NextMsg(3 * time.Second)
		if err != nil {
			break // MaxDeliver exhausted — expected
		}
		delay := backoff[min(attempt, len(backoff)-1)]
		_ = msg.NakWithDelay(delay)
	}
	return nil
}

func (s *ebSuite) eventAppearsOnDLQTopic() error {
	dlqSubj := "zynax.dlq.zynax.v1.bdd.dlq.>"
	sub, err := s.nc.SubscribeSync(dlqSubj)
	if err != nil {
		return fmt.Errorf("dlq subscribe: %w", err)
	}
	defer func() { _ = sub.Unsubscribe() }()
	msg, err := sub.NextMsg(5 * time.Second)
	if err != nil {
		return fmt.Errorf("DLQ message not received: %w", err)
	}
	if len(msg.Data) == 0 {
		return fmt.Errorf("DLQ message is empty")
	}
	return nil
}

// Scenario 5 — Durable consumer catches up after being offline
func (s *ebSuite) consumerWasOfflineWhenEventPublished(d string) error {
	s.topic = "zynax.v1.bdd.durable.event"
	ctx, cancel := context.WithCancel(context.Background())
	ch, err := s.bus.Subscribe(ctx, domain.SubscribeRequest{
		SubscriberID: d, TypePattern: s.topic,
	})
	if err != nil {
		cancel()
		return fmt.Errorf("subscribe %s: %w", d, err)
	}
	// Go offline: cancel context, drain channel.
	cancel()
	for range ch {
	}

	// Publish while offline.
	s.event = domain.CloudEvent{
		ID: "bdd-durable-1", Source: "bdd", SpecVersion: "1.0",
		Type: s.topic, Data: []byte(`{"catch_up":true}`),
	}
	_, err = s.bus.Publish(context.Background(), s.event)
	return err
}

func (s *ebSuite) consumerReconnects(d string) error {
	ctx, cancel := context.WithCancel(context.Background())
	s.cancelFn[d] = cancel
	ch, err := s.bus.Subscribe(ctx, domain.SubscribeRequest{
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

// Scenario 6 — Two consumer groups receive same event independently
func (s *ebSuite) groupsSubscribeToSameTopic(g1, g2, topic string) error {
	s.topic = topic
	for _, g := range []string{g1, g2} {
		ctx, cancel := context.WithCancel(context.Background())
		s.cancelFn[g] = cancel
		ch, err := s.bus.Subscribe(ctx, domain.SubscribeRequest{
			SubscriberID: g, TypePattern: topic,
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
	s.event = domain.CloudEvent{
		ID: "bdd-grp-1", Source: "bdd", SpecVersion: "1.0", Type: s.topic, Data: []byte(`{}`),
	}
	return s.publishEvent()
}

func (s *ebSuite) bothGroupsReceiveIndependentCopy(g1, g2 string) error {
	return s.bothReceiveEvent(g1, g2)
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
	sc.Step(`^the event appears on the DLQ topic$`, s.eventAppearsOnDLQTopic)
	sc.Step(`^consumer "([^"]*)" was offline when an event was published$`, s.consumerWasOfflineWhenEventPublished)
	sc.Step(`^consumer "([^"]*)" reconnects$`, s.consumerReconnects)
	sc.Step(`^consumer "([^"]*)" receives the missed event$`, s.consumerReceivesMissedEvent)
	sc.Step(`^groups "([^"]*)" and "([^"]*)" both subscribe to the same topic$`, s.groupsSubscribeToSameTopic)
	sc.Step(`^one event is published$`, s.oneEventIsPublished)
	sc.Step(`^both groups receive their own independent copy$`, s.bothGroupsReceiveIndependentCopy)
}

// ── test runner ───────────────────────────────────────────────────────────────

func TestFeatures(t *testing.T) {
	natsURL, cleanup := startNATS(t)
	defer cleanup()

	bus, err := infrastructure.NewNATSEventBus(natsURL)
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
		Name: "event-bus",
		ScenarioInitializer: func(sc *godog.ScenarioContext) {
			s := &ebSuite{
				natsURL:  natsURL,
				bus:      bus,
				channels: make(map[string]<-chan domain.CloudEvent),
				cancelFn: make(map[string]context.CancelFunc),
				nc:       nc,
				js:       js,
			}
			s.initScenario(sc)
			sc.After(func(ctx context.Context, _ *godog.Scenario, _ error) (context.Context, error) {
				for _, cancel := range s.cancelFn {
					cancel()
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
