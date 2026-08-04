# SPDX-License-Identifier: Apache-2.0
# Zynax — Direct JetStream events client BDD Contract Specification (M8.H, ADR-046)
#
# This file is the SPECIFICATION for the shared events client
# (libs/zynaxevents). It pins the DLQ and durable-consumer semantics the
# client preserves VERBATIM from the event-bus facade it replaces — the six
# facade scenarios carry over unchanged, plus the #1149 disjoint-stream rule,
# the workflow-scoped terminal-close contracts and the Unsubscribe teardown
# contract (durable deleted, DLQ_* skipped). The golden byte-compat
# fixtures live in libs/zynaxevents/testdata/golden/ and are asserted against
# BOTH implementations until the facade is removed (M9).
#
# Step definitions: tests/steps_test.go (integration-tagged, testcontainers
# NATS). Run with: GOWORK=off go test -tags integration -timeout 300s ./tests/...

Feature: Direct JetStream events client — delivery, DLQ, durable consumers, terminal-close
  As a Zynax service publishing and consuming platform events
  I want the shared client to reproduce the facade's delivery semantics exactly
  So that retiring the facade changes the topology, never the behaviour

  Scenario: Published event reaches all subscribers
    Given consumers "a" and "b" subscribe to topic "zynax.v1.task-broker.task.completed"
    When an event is published to that topic
    Then both "a" and "b" receive the event

  Scenario: Subscriber on different topic does not receive event
    Given consumer "c" subscribes to "zynax.v1.task-broker.task.assigned"
    When an event is published to "zynax.v1.task-broker.task.completed"
    Then consumer "c" does NOT receive the event

  Scenario: Failed delivery is retried with backoff
    Given a subscriber that fails on first attempt
    When an event is published
    Then the event is redelivered at least once

  # End-to-end DLQ forwarding (#1653). The facade never moved an exhausted
  # message, so this used to pin only the advisory + the provisioned stream.
  # The opt-in DLQForwarder closes the loop: it consumes the max-deliveries
  # advisory, fetches the exhausted message from its source stream by sequence
  # and republishes it on the reserved exact zynax.dlq.<prefix>.dead subject —
  # byte-for-byte, without deleting the source, and idempotently (a repeated
  # advisory is deduplicated on a deterministic <dlq-stream>:<sequence> id).
  Scenario: Retry exhaustion forwards the exhausted message into the DLQ stream
    Given a subscriber that always fails
    And the DLQ forwarder is running
    When an event is published
    And 5 delivery attempts are exhausted
    Then a max-deliveries advisory is emitted for the consumer
    And the DLQ stream for the topic exists with WorkQueuePolicy retention
    And the exhausted event lands on the DLQ stream with its bytes intact
    And the source message is still present in its own stream
    And replaying the advisory does not duplicate the message in the DLQ stream

  Scenario: Durable consumer catches up after being offline
    Given consumer "d" was offline when an event was published
    When consumer "d" reconnects
    Then consumer "d" receives the missed event

  Scenario: Two consumer groups receive same event independently
    Given groups "indexer" and "notifier" both subscribe to the same topic
    When one event is published
    Then both groups receive their own independent copy

  Scenario: Events under one entity prefix share a single stream
    When an event of type "zynax.v1.engine-adapter.workflow.completed" is published
    And an event of type "zynax.v1.engine-adapter.workflow.state.entered" is published
    Then both events land on stream "ZYNAX_V1_ENGINE_ADAPTER_WORKFLOW"
    And no "subjects overlap with an existing stream" error occurs

  Scenario: A workflow-scoped subscription closes on the terminal lifecycle event
    Given a subscriber scoped to workflow "wf-terminal-1" with pattern "zynax.v1.bdd.wfterm.**"
    When a "zynax.v1.bdd.wfterm.workflow.completed" event for workflow "wf-terminal-1" is published
    Then the subscriber receives the terminal event
    And the event channel is closed

  Scenario: A wildcard subscription does not close on one run's terminal event
    Given a subscriber with pattern "zynax.v1.bdd.wfwild.**" and no workflow scope
    When a "zynax.v1.bdd.wfwild.workflow.completed" event for workflow "wf-terminal-2" is published
    Then the subscriber receives the terminal event
    And the event channel remains open

  # Unsubscribe (#1657). Unsubscribe is the explicit teardown of a durable
  # consumer, independent of the subscription's context: it walks every stream,
  # deletes the durable named after the subscriber, and reports
  # ErrSubscriberNotFound when no stream owned one. The subscription below is
  # deliberately still live when Unsubscribe is called — nats.go deletes a
  # library-created durable on the subscription's own Unsubscribe/Drain, so
  # cancelling first would leave nothing for the client call to delete and the
  # scenario would assert nothing.
  Scenario: Unsubscribing deletes the subscriber's durable consumer
    Given subscriber "bdd-unsub-live" is subscribed to topic "zynax.v1.bdd.unsub.event"
    And the durable consumer for "bdd-unsub-live" exists on stream "ZYNAX_V1_BDD_UNSUB"
    When subscriber "bdd-unsub-live" unsubscribes
    Then the unsubscribe call succeeds
    And the durable consumer for "bdd-unsub-live" is gone from stream "ZYNAX_V1_BDD_UNSUB"

  Scenario: Unsubscribing an unknown subscriber reports it as not found
    When subscriber "bdd-unsub-ghost" unsubscribes
    Then the unsubscribe call fails with ErrSubscriberNotFound

  # DLQ_* streams are skipped on purpose: dead-letter consumers belong to the
  # DLQ machinery, never to the subscriber. A durable that exists ONLY on a DLQ
  # stream is therefore both "not found" and left standing.
  Scenario: Unsubscribe never deletes a consumer from a DLQ stream
    Given a durable consumer for "bdd-unsub-dlq" exists only on stream "DLQ_ZYNAX_V1_BDD_UNSUBDLQ"
    When subscriber "bdd-unsub-dlq" unsubscribes
    Then the unsubscribe call fails with ErrSubscriberNotFound
    And the durable consumer for "bdd-unsub-dlq" still exists on stream "DLQ_ZYNAX_V1_BDD_UNSUBDLQ"
