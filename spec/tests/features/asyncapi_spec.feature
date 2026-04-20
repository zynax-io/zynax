# SPDX-License-Identifier: Apache-2.0
# Zynax — AsyncAPI Specification BDD Contract
#
# This file is the SPECIFICATION. It is written BEFORE the implementation.
# It describes what the AsyncAPI document must contain and validates it
# is machine-processable by AsyncAPI tooling.
# See spec/AGENTS.md for spec governance rules.
#
# Business context: Zynax is event-driven — state transitions, task
# completions, and agent lifecycle changes are communicated asynchronously
# via NATS JetStream. Consumers (adapters, observability tools, external
# systems) need a machine-readable spec of every event type — its schema,
# channel, and when it is emitted. AsyncAPI is the CNCF-aligned standard
# for async API documentation. (ADR-001, ADR-014)

Feature: AsyncAPI specification for all Zynax async events
  As an adapter developer
  I want a machine-readable document describing every event type
  So that I can generate typed consumers without reading source code

  Background:
    Given the file "spec/asyncapi/zynax-events.yaml" exists

  # ─── Document validity ────────────────────────────────────────────────────

  Scenario: AsyncAPI document is valid
    When it is validated with the AsyncAPI validator
    Then validation passes with zero errors

  Scenario: AsyncAPI document specifies version 2.x
    When the document is parsed
    Then the asyncapi field is "2.6.0"

  Scenario: AsyncAPI document declares the NATS server
    When the document is parsed
    Then it declares a server with protocol "nats"

  Scenario: AsyncAPI document has an info block with title and version
    When the document is parsed
    Then the info.title is "Zynax Event Bus"
    And the info.version is present and non-empty

  # ─── Workflow lifecycle events ────────────────────────────────────────────

  Scenario: Workflow started event channel is documented
    When the AsyncAPI spec channels are inspected
    Then it contains a channel for subject "zynax.workflow.started"
    And the channel specifies a publish operation
    And the message payload references the CloudEvent schema

  Scenario: Workflow completed event channel is documented
    When the AsyncAPI spec channels are inspected
    Then it contains a channel for subject "zynax.workflow.completed"
    And the channel specifies a publish operation
    And the message payload references the CloudEvent schema

  Scenario: Workflow failed event channel is documented
    When the AsyncAPI spec channels are inspected
    Then it contains a channel for subject "zynax.workflow.failed"
    And the channel specifies a publish operation
    And the message payload references the CloudEvent schema

  Scenario: Workflow cancelled event channel is documented
    When the AsyncAPI spec channels are inspected
    Then it contains a channel for subject "zynax.workflow.cancelled"
    And the channel specifies a publish operation
    And the message payload references the CloudEvent schema

  # ─── Task lifecycle events ────────────────────────────────────────────────

  Scenario: Task dispatched event channel is documented
    When the AsyncAPI spec channels are inspected
    Then it contains a channel for subject "zynax.task.dispatched"
    And the channel specifies a publish operation
    And the message payload references the CloudEvent schema

  Scenario: Task completed event channel is documented
    When the AsyncAPI spec channels are inspected
    Then it contains a channel for subject "zynax.task.completed"
    And the channel specifies a publish operation
    And the message payload references the CloudEvent schema

  Scenario: Task failed event channel is documented
    When the AsyncAPI spec channels are inspected
    Then it contains a channel for subject "zynax.task.failed"
    And the channel specifies a publish operation
    And the message payload references the CloudEvent schema

  Scenario: Task retrying event channel is documented
    When the AsyncAPI spec channels are inspected
    Then it contains a channel for subject "zynax.task.retrying"
    And the channel specifies a publish operation
    And the message payload references the CloudEvent schema

  # ─── Agent lifecycle events ───────────────────────────────────────────────

  Scenario: Agent registered event channel is documented
    When the AsyncAPI spec channels are inspected
    Then it contains a channel for subject "zynax.agent.registered"
    And the channel specifies a publish operation
    And the message payload references the CloudEvent schema

  Scenario: Agent deregistered event channel is documented
    When the AsyncAPI spec channels are inspected
    Then it contains a channel for subject "zynax.agent.deregistered"
    And the channel specifies a publish operation
    And the message payload references the CloudEvent schema

  Scenario: Agent capability invoked event channel is documented
    When the AsyncAPI spec channels are inspected
    Then it contains a channel for subject "zynax.agent.capability.invoked"
    And the channel specifies a publish operation
    And the message payload references the CloudEvent schema

  # ─── Schema references ────────────────────────────────────────────────────

  Scenario: Every event payload references the CloudEvent envelope schema
    When all channel message schemas in the AsyncAPI spec are inspected
    Then every message payload conforms to the CloudEvents envelope schema

  Scenario: CloudEvent schema reference resolves to spec/schemas/cloudevent.schema.json
    When the schema components are inspected
    Then the "CloudEvent" component references "../../schemas/cloudevent.schema.json"

  # ─── Channel metadata ─────────────────────────────────────────────────────

  Scenario: Every channel has a description
    When all channels in the AsyncAPI spec are inspected
    Then every channel has a non-empty description

  Scenario: Every channel message has a name and summary
    When all channel messages in the AsyncAPI spec are inspected
    Then every message has a non-empty name
    And every message has a non-empty summary

  # ─── Makefile target ──────────────────────────────────────────────────────

  Scenario: make validate-spec runs the AsyncAPI validator via Docker
    Given a Makefile exists at the repository root
    When the "validate-spec" target is inspected
    Then it invokes the AsyncAPI validator using Docker
    And it targets the file "spec/asyncapi/zynax-events.yaml"
