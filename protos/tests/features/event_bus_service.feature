# SPDX-License-Identifier: Apache-2.0
# Zynax — EventBusService BDD Contract Specification
#
# This file is the SPECIFICATION. It is written BEFORE the implementation.
# It describes the contract the event bus service must honour.
# See protos/AGENTS.md §7 for contract test rules.
#
# Business context: Zynax workflows are event-driven state machines (ADR-014).
# State transitions are triggered by events — "review.approved", "push",
# "task.completed". These events flow through a central bus. All services
# (workflow engine, task broker, adapters) must be able to publish and subscribe
# to events via a stable contract. (ADR-001, ADR-014)

Feature: EventBusService contract — async event publish/subscribe
  As a workflow engine
  I want to publish and subscribe to domain events
  So that state transitions trigger automatically without polling

  Background:
    Given an EventBusService is running on a test gRPC server

  # ─── Publish ──────────────────────────────────────────────────────────────

  Scenario: Publish an event returns an event_id
    Given a valid CloudEvent with type "zynax.workflow.review.approved" scoped to "wf-42"
    When Publish is called with the event
    Then the gRPC status is OK
    And the response contains a non-empty event_id

  Scenario: Published event is delivered to a matching subscriber
    Given subscriber "sub-A" is listening to type pattern "zynax.workflow.*"
    When Publish is called with a CloudEvent of type "zynax.workflow.review.approved"
    Then subscriber "sub-A" receives the event
    And the received event type is "zynax.workflow.review.approved"

  Scenario: Published event is delivered to all matching subscribers
    Given subscriber "sub-A" is listening to type pattern "zynax.workflow.*"
    And subscriber "sub-B" is listening to type pattern "zynax.workflow.*"
    When Publish is called with a CloudEvent of type "zynax.workflow.completed"
    Then subscriber "sub-A" receives the event
    And subscriber "sub-B" receives the event

  Scenario: Published event is not delivered to a non-matching subscriber
    Given subscriber "sub-A" is listening to type pattern "zynax.task.*"
    When Publish is called with a CloudEvent of type "zynax.workflow.completed"
    Then subscriber "sub-A" does not receive the event

  Scenario: Publish with missing CloudEvent returns INVALID_ARGUMENT
    Given a PublishRequest with no CloudEvent envelope
    When Publish is called
    Then the gRPC status is INVALID_ARGUMENT
    And the error message mentions "event"

  Scenario: Publish with empty workflow_id in event is accepted
    Given a valid CloudEvent with an empty workflow_id field
    When Publish is called with the event
    Then the gRPC status is OK

  # ─── Subscribe ────────────────────────────────────────────────────────────

  Scenario: Subscribe streams CloudEvents matching the type pattern
    Given subscriber "sub-A" subscribes with type pattern "zynax.workflow.*"
    When a CloudEvent of type "zynax.workflow.review.approved" is published
    Then the Subscribe stream delivers the CloudEvent to "sub-A"
    And the delivered event has a non-empty id

  Scenario: Subscribe filters events by workflow_id scope
    Given subscriber "sub-A" subscribes with workflow_id scope "wf-1"
    And subscriber "sub-B" subscribes with workflow_id scope "wf-2"
    When a CloudEvent scoped to workflow_id "wf-1" is published
    Then subscriber "sub-A" receives the event
    And subscriber "sub-B" does not receive the event

  Scenario: Subscribe with no workflow_id scope receives events for all workflows
    Given subscriber "sub-A" subscribes with type pattern "zynax.*" and no workflow_id filter
    When a CloudEvent scoped to "wf-1" is published
    And a CloudEvent scoped to "wf-2" is published
    Then subscriber "sub-A" receives both events

  Scenario: Subscribe with empty subscriber_id returns INVALID_ARGUMENT
    Given a SubscribeRequest with subscriber_id set to ""
    When Subscribe is called
    Then the gRPC status is INVALID_ARGUMENT
    And the error message mentions "subscriber_id"

  Scenario: Subscribe with empty type_pattern returns INVALID_ARGUMENT
    Given a SubscribeRequest with type_pattern set to ""
    When Subscribe is called
    Then the gRPC status is INVALID_ARGUMENT
    And the error message mentions "type_pattern"

  Scenario: Subscribe stream stays open until Unsubscribe is called
    Given subscriber "sub-keep" subscribes with type pattern "zynax.*"
    When a CloudEvent is published
    And Unsubscribe is not called
    Then the Subscribe stream remains open
    And "sub-keep" continues to receive subsequent events

  # ─── Unsubscribe ──────────────────────────────────────────────────────────

  Scenario: Unsubscribe stops event delivery
    Given subscriber "sub-99" is actively subscribed to type pattern "zynax.*"
    When Unsubscribe is called for subscriber_id "sub-99"
    Then the gRPC status is OK
    And a subsequent matching event is published
    And "sub-99" receives no further events

  Scenario: Unsubscribe closes the Subscribe stream cleanly
    Given subscriber "sub-99" has an active Subscribe stream
    When Unsubscribe is called for subscriber_id "sub-99"
    Then the Subscribe stream closes with status OK

  Scenario: Unsubscribe for unknown subscriber_id returns NOT_FOUND
    When Unsubscribe is called for subscriber_id "ghost-sub"
    Then the gRPC status is NOT_FOUND
    And the error message contains "ghost-sub"

  Scenario: Unsubscribe with empty subscriber_id returns INVALID_ARGUMENT
    Given an UnsubscribeRequest with subscriber_id set to ""
    When Unsubscribe is called
    Then the gRPC status is INVALID_ARGUMENT
    And the error message mentions "subscriber_id"

  # ─── Glob pattern matching ────────────────────────────────────────────────

  Scenario: Wildcard pattern matches multiple event type prefixes
    Given subscriber "sub-A" subscribes with type pattern "zynax.*"
    When a CloudEvent of type "zynax.workflow.completed" is published
    And a CloudEvent of type "zynax.task.failed" is published
    Then subscriber "sub-A" receives both events

  Scenario: Exact pattern matches only the exact event type
    Given subscriber "sub-A" subscribes with type pattern "zynax.workflow.completed"
    When a CloudEvent of type "zynax.workflow.completed" is published
    And a CloudEvent of type "zynax.workflow.started" is published
    Then subscriber "sub-A" receives exactly 1 event

  Scenario: Double-wildcard pattern matches nested event types
    Given subscriber "sub-A" subscribes with type pattern "zynax.**"
    When a CloudEvent of type "zynax.workflow.review.approved" is published
    Then subscriber "sub-A" receives the event

  # ─── SubscribeResponse metadata ───────────────────────────────────────────

  Scenario: Subscribe response includes the subscriber_id
    Given a SubscribeRequest with subscriber_id "sub-meta"
    When Subscribe is called
    Then the initial SubscribeResponse contains subscriber_id "sub-meta"

  # ─── Input validation ─────────────────────────────────────────────────────

  Scenario: Publish with CloudEvent missing required id field is rejected
    Given a CloudEvent with id set to ""
    When Publish is called
    Then the gRPC status is INVALID_ARGUMENT
    And the error message mentions "id"

  Scenario: Publish with CloudEvent missing required source field is rejected
    Given a CloudEvent with source set to ""
    When Publish is called
    Then the gRPC status is INVALID_ARGUMENT
    And the error message mentions "source"

  Scenario: Publish with CloudEvent missing required type field is rejected
    Given a CloudEvent with type set to ""
    When Publish is called
    Then the gRPC status is INVALID_ARGUMENT
    And the error message mentions "type"
