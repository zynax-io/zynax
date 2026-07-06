# SPDX-License-Identifier: Apache-2.0
# Zynax — Direct JetStream events client BDD Contract Specification (M8.H, ADR-046)
#
# This file is the SPECIFICATION, committed BEFORE the libs/zynaxevents
# implementation (ADR-016: contracts before code; ADR-046 acceptance gate).
# It pins the DLQ and durable-consumer semantics the shared client must
# preserve VERBATIM from the event-bus facade it replaces. The golden
# byte-compat fixtures live in libs/zynaxevents/testdata/golden/ and are
# asserted against BOTH implementations until the facade is removed (M9).
#
# Step definitions land with the subscribe side of the client (#1646),
# integration-tagged against a real JetStream (kind NATS / testcontainer).

Feature: Direct JetStream events client — DLQ, durable consumers, terminal-close
  As a Zynax service publishing and consuming platform events
  I want the shared client to reproduce the facade's delivery semantics exactly
  So that retiring the facade changes the topology, never the behaviour

  Background:
    Given a JetStream server is running
    And the shared events client is connected

  # ─── Stream derivation (#1149 disjoint-filter rule) ────────────────────────

  Scenario: Events under one entity prefix share a single stream
    When an event of type "zynax.v1.engine-adapter.workflow.completed" is published
    And an event of type "zynax.v1.engine-adapter.workflow.state.entered" is published
    Then both events land on stream "ZYNAX_V1_ENGINE_ADAPTER_WORKFLOW"
    And no "subjects overlap with an existing stream" error occurs

  # ─── DLQ: retry exhaustion routes to the dead-letter stream ────────────────

  Scenario: A message that exhausts its delivery retries lands on the DLQ
    Given a durable subscriber whose handler never acknowledges
    And the subscriber consumes events of type "zynax.v1.task-broker.task.dispatched"
    When an event of that type is published
    And all 5 delivery attempts are exhausted per the retry backoff schedule
    Then the event is delivered to DLQ stream "DLQ_ZYNAX_V1_TASK_BROKER_TASK"
    And the DLQ deliver subject is "zynax.dlq.zynax.v1.task-broker.task.dead"
    And the DLQ stream retention is WorkQueuePolicy

  # ─── Durable consumers ──────────────────────────────────────────────────────

  Scenario: A durable consumer resumes from its acknowledged position
    Given a durable subscriber "resume-sub" consuming "zynax.v1.engine-adapter.workflow.**"
    And it has acknowledged an event
    When the subscription is closed and reopened with the same subscriber id
    Then the acknowledged event is not redelivered
    And the durable consumer name is the sanitized subscriber id

  # ─── Workflow-scoped terminal-close ─────────────────────────────────────────

  Scenario: A workflow-scoped subscription closes on the terminal lifecycle event
    Given a subscriber scoped to workflow "wf-terminal-1" with pattern "zynax.v1.engine-adapter.workflow.**"
    When a "zynax.v1.engine-adapter.workflow.completed" event for workflow "wf-terminal-1" is published
    Then the subscriber receives the terminal event
    And the event channel is closed

  Scenario: A wildcard subscription does not close on one run's terminal event
    Given a subscriber with pattern "zynax.v1.engine-adapter.workflow.**" and no workflow scope
    When a "zynax.v1.engine-adapter.workflow.completed" event for workflow "wf-terminal-2" is published
    Then the subscriber receives the terminal event
    And the event channel remains open
