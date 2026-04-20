# SPDX-License-Identifier: Apache-2.0
# Zynax — CloudEvents Envelope BDD Contract Specification
#
# This file is the SPECIFICATION. It is written BEFORE the implementation.
# It describes the contract the CloudEvent envelope must honour.
# See protos/AGENTS.md §7 for contract test rules.
#
# Business context: All async events in Zynax must be interoperable with the
# cloud-native ecosystem. CloudEvents is the CNCF standard for describing event
# data in a common way. Every event published to the event bus uses a
# CloudEvents-compatible envelope so external consumers (monitoring, audit,
# third-party tooling) can process events without Zynax-specific clients.
# (ADR-001, CloudEvents v1.0 spec)

Feature: CloudEvents-compatible event envelope
  As a platform operator
  I want all Zynax events to conform to the CloudEvents specification
  So that any CloudEvents-compatible consumer can process them without custom code

  # ─── Schema validation ────────────────────────────────────────────────────

  Scenario: A valid CloudEvent passes schema validation
    Given a JSON object with required fields: specversion "1.0", id "evt-001", source "/zynax/wf-42", type "zynax.workflow.completed", datacontenttype "application/json"
    When the envelope is validated against the Zynax CloudEvent JSON Schema
    Then validation passes with zero errors

  Scenario: A CloudEvent missing the id field is rejected
    Given a valid CloudEvent JSON with the "id" field removed
    When the envelope is validated against the Zynax CloudEvent JSON Schema
    Then validation fails
    And the error names the missing field "id"

  Scenario: A CloudEvent missing the source field is rejected
    Given a valid CloudEvent JSON with the "source" field removed
    When the envelope is validated against the Zynax CloudEvent JSON Schema
    Then validation fails
    And the error names the missing field "source"

  Scenario: A CloudEvent missing the specversion field is rejected
    Given a valid CloudEvent JSON with the "specversion" field removed
    When the envelope is validated against the Zynax CloudEvent JSON Schema
    Then validation fails
    And the error names the missing field "specversion"

  Scenario: A CloudEvent missing the type field is rejected
    Given a valid CloudEvent JSON with the "type" field removed
    When the envelope is validated against the Zynax CloudEvent JSON Schema
    Then validation fails
    And the error names the missing field "type"

  Scenario: specversion must be exactly "1.0"
    Given a valid CloudEvent JSON with specversion set to "0.3"
    When the envelope is validated against the Zynax CloudEvent JSON Schema
    Then validation fails
    And the error references the "specversion" field

  Scenario: A CloudEvent with all optional fields passes validation
    Given a valid CloudEvent JSON base
    And the envelope includes optional field "datacontenttype" with value "application/json"
    And the envelope includes optional field "subject" with value "wf-42/step-3"
    When the envelope is validated against the Zynax CloudEvent JSON Schema
    Then validation passes with zero errors

  # ─── Zynax extension attributes ───────────────────────────────────────────

  Scenario: Zynax extension attributes are accepted
    Given a valid CloudEvent JSON base
    And the envelope includes Zynax extension "workflow_id" with value "wf-42"
    And the envelope includes Zynax extension "run_id" with value "run-abc"
    And the envelope includes Zynax extension "namespace" with value "team-alpha"
    When the envelope is validated against the Zynax CloudEvent JSON Schema
    Then validation passes and extension attributes are preserved

  Scenario: Zynax capability_name extension is accepted
    Given a valid CloudEvent JSON base
    And the envelope includes Zynax extension "capability_name" with value "code-review"
    When the envelope is validated against the Zynax CloudEvent JSON Schema
    Then validation passes with zero errors

  Scenario: A CloudEvent without Zynax extension attributes is still valid
    Given a valid CloudEvent JSON base with no Zynax extension fields
    When the envelope is validated against the Zynax CloudEvent JSON Schema
    Then validation passes with zero errors

  # ─── Proto message structure ──────────────────────────────────────────────

  Scenario: CloudEvent proto has all required fields
    Given the CloudEvent proto message definition
    Then it contains field "id" of type string
    And it contains field "source" of type string
    And it contains field "specversion" of type string
    And it contains field "type" of type string
    And it contains field "datacontenttype" of type string
    And it contains field "time" of type google.protobuf.Timestamp
    And it contains field "data" of type bytes

  Scenario: CloudEvent proto has Zynax extension attribute fields
    Given the CloudEvent proto message definition
    Then it contains field "workflow_id" of type string
    And it contains field "run_id" of type string
    And it contains field "namespace" of type string
    And it contains field "capability_name" of type string

  # ─── Proto JSON round-trip ────────────────────────────────────────────────

  Scenario: Proto representation round-trips through JSON
    Given a CloudEvent proto message with id "evt-001" source "/zynax/wf-42" type "zynax.workflow.completed"
    When it is serialised to JSON using proto-json encoding
    And deserialised back to a CloudEvent proto message
    Then the result is equal to the original message

  Scenario: Proto round-trip preserves Zynax extension attributes
    Given a CloudEvent proto message with workflow_id "wf-42" run_id "run-abc" namespace "team-alpha"
    When it is serialised to JSON and deserialised back to proto
    Then workflow_id is "wf-42"
    And run_id is "run-abc"
    And namespace is "team-alpha"

  Scenario: Proto round-trip preserves binary data payload
    Given a CloudEvent proto message with data bytes [0x01, 0x02, 0x03, 0xFF]
    When it is serialised to JSON and deserialised back to proto
    Then the data bytes equal [0x01, 0x02, 0x03, 0xFF]

  # ─── Input validation ─────────────────────────────────────────────────────

  Scenario: CloudEvent with empty id is structurally invalid
    Given a CloudEvent JSON with id set to ""
    When the envelope is validated against the Zynax CloudEvent JSON Schema
    Then validation fails
    And the error references the "id" field

  Scenario: CloudEvent with empty source is structurally invalid
    Given a CloudEvent JSON with source set to ""
    When the envelope is validated against the Zynax CloudEvent JSON Schema
    Then validation fails
    And the error references the "source" field
