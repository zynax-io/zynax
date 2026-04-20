# SPDX-License-Identifier: Apache-2.0
# Zynax — Capability Schema BDD Contract
#
# This file is the SPECIFICATION. It is written BEFORE the implementation.
# It describes what the capability schema must enforce and how it integrates
# with task dispatch validation.
# See spec/AGENTS.md for spec governance rules.
#
# Business context: Every capability in Zynax has declared input and output
# contracts. Without a machine-readable schema the platform cannot validate
# task inputs at dispatch time, tooling cannot generate type-safe clients,
# and workflow authors have no contract to write against. The capability
# schema is the "OpenAPI spec" for individual agent capabilities. (ADR-013)

Feature: Capability schema for input/output declarations
  As a workflow author
  I want to declare capability inputs and outputs with a schema
  So that the platform can validate task data before dispatching

  Background:
    Given the JSON Schema at "spec/schemas/capability.schema.json" is loaded

  # ─── Valid declarations ───────────────────────────────────────────────────

  Scenario: A valid capability declaration passes schema validation
    Given a capability declaration with name "summarize"
    And the capability declares input_schema with fields: text (string), max_words (integer)
    And the capability declares output_schema with fields: summary (string), word_count (integer)
    When the declaration is validated against the capability schema
    Then validation passes with zero errors

  Scenario: A minimal capability with only required fields passes validation
    Given a capability declaration with name "ping" and description "health check"
    And no input_schema or output_schema is provided
    When the declaration is validated against the capability schema
    Then validation passes with zero errors

  Scenario: A capability with timeout_seconds passes validation
    Given a capability declaration with name "slow-job"
    And timeout_seconds is set to 600
    When the declaration is validated against the capability schema
    Then validation passes with zero errors

  Scenario: A capability with max_retries passes validation
    Given a capability declaration with name "flaky-op"
    And max_retries is set to 3
    When the declaration is validated against the capability schema
    Then validation passes with zero errors

  Scenario: Optional fields with defaults are accepted when omitted
    Given a capability declaration with name "code-review"
    And neither timeout_seconds nor max_retries is specified
    When the declaration is validated against the capability schema
    Then validation passes with zero errors
    And the default timeout_seconds is 300
    And the default max_retries is 0

  # ─── Required field validation ────────────────────────────────────────────

  Scenario: A capability missing the name field is rejected
    Given a capability declaration with no name field
    When the declaration is validated against the capability schema
    Then validation fails
    And the error names the missing field "name"

  Scenario: A capability with an empty name is rejected
    Given a capability declaration with name set to ""
    When the declaration is validated against the capability schema
    Then validation fails
    And the error references the "name" field

  # ─── input_schema and output_schema structure ─────────────────────────────

  Scenario: input_schema must be a valid JSON Schema object
    Given a capability declaration with name "check"
    And input_schema is set to a valid JSON Schema object with type "object"
    When the declaration is validated against the capability schema
    Then validation passes with zero errors

  Scenario: output_schema must be a valid JSON Schema object
    Given a capability declaration with name "check"
    And output_schema is set to a valid JSON Schema object with type "object"
    When the declaration is validated against the capability schema
    Then validation passes with zero errors

  Scenario: A capability with invalid input_schema type reference is rejected
    Given a capability declaration with name "bad-cap"
    And input_schema declares a field "count" with type "doesNotExist"
    When the declaration is validated against the capability schema
    Then validation fails
    And the error names the invalid type on field "count"

  Scenario: input_schema with nested object properties passes validation
    Given a capability declaration with name "complex-cap"
    And input_schema declares a nested object field "config" with sub-fields
    When the declaration is validated against the capability schema
    Then validation passes with zero errors

  # ─── Field type constraints ───────────────────────────────────────────────

  Scenario: timeout_seconds must be a positive integer
    Given a capability declaration with name "timed"
    And timeout_seconds is set to -1
    When the declaration is validated against the capability schema
    Then validation fails
    And the error references the "timeout_seconds" field

  Scenario: timeout_seconds must not be zero
    Given a capability declaration with name "timed"
    And timeout_seconds is set to 0
    When the declaration is validated against the capability schema
    Then validation fails
    And the error references the "timeout_seconds" field

  Scenario: max_retries must be a non-negative integer
    Given a capability declaration with name "retryable"
    And max_retries is set to -1
    When the declaration is validated against the capability schema
    Then validation fails
    And the error references the "max_retries" field

  Scenario: max_retries of zero is valid
    Given a capability declaration with name "no-retry"
    And max_retries is set to 0
    When the declaration is validated against the capability schema
    Then validation passes with zero errors

  # ─── AgentDef YAML integration ────────────────────────────────────────────

  Scenario: A valid AgentDef YAML with capability declarations passes validation
    Given a YAML AgentDef at "spec/workflows/examples/agent-def-example.yaml"
    When it is validated against the capability schema for each capability
    Then all capability declarations pass with zero errors

  Scenario: make validate-spec validates all AgentDef YAMLs in spec/
    Given a Makefile exists at the repository root
    When the "validate-spec" target is inspected
    Then it validates YAML files in spec/ against capability.schema.json

  # ─── Schema self-consistency ──────────────────────────────────────────────

  Scenario: The capability schema itself is a valid JSON Schema document
    When "spec/schemas/capability.schema.json" is validated as a JSON Schema
    Then it is a valid draft 2020-12 JSON Schema document

  Scenario: The capability schema has a title and description
    When "spec/schemas/capability.schema.json" is parsed
    Then the title field is present and non-empty
    And the description field is present and non-empty
