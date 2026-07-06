# SPDX-License-Identifier: Apache-2.0
# Zynax — Direct JetStream events client BDD Contract Specification (M8.H, ADR-046)
#
# This file is the SPECIFICATION for the shared events client
# (libs/zynaxevents). It pins the DLQ and durable-consumer semantics the
# client preserves VERBATIM from the event-bus facade it replaces — the six
# facade scenarios carry over unchanged, plus the #1149 disjoint-stream rule
# and the workflow-scoped terminal-close contracts. The golden byte-compat
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

  # End-to-end DLQ forwarding (max-deliveries advisory -> DLQ mover) was never
  # built in the facade and is NOT claimed here — the conventions provision the
  # DLQ stream and stop redelivery after MaxDeliver; forwarding is a tracked
  # follow-up. This scenario pins what is actually guaranteed.
  Scenario: Retry exhaustion surfaces max-deliveries and the DLQ stream is provisioned
    Given a subscriber that always fails
    When an event is published
    And 5 delivery attempts are exhausted
    Then a max-deliveries advisory is emitted for the consumer
    And the DLQ stream for the topic exists with WorkQueuePolicy retention

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
