# SPDX-License-Identifier: Apache-2.0
# Zynax — Reusable Template + Versioning BDD Contract
#
# This file is the SPECIFICATION. It is written BEFORE / alongside the spec
# change. It describes the contract that reusable templates and the manifest
# `version:` field must honour and how they integrate with `make validate-spec`.
# See spec/AGENTS.md for spec governance rules and EPIC #1171 (canvas T.1).
#
# Business context: workflow authors should start from a known-good baseline
# rather than authoring manifests from scratch. Zynax ships three reusable,
# parameterized templates — workflow, task, expert — under spec/templates/.
# Manifests carry an optional, backward-compatible `metadata.version` (SemVer)
# so an author can evolve a workflow or capability contract without breaking
# already-registered instances. (#1206, EPIC #1171 step T.1)

Feature: Reusable, versioned templates for workflows, tasks, and experts
  As a workflow author
  I want reusable, versioned templates
  So that I do not author manifests from scratch and can evolve them safely

  Background:
    Given the JSON Schemas under "spec/schemas/" are loaded
    And the reusable templates under "spec/templates/" exist

  # ─── Template existence + shape ────────────────────────────────────────────

  Scenario: The workflow template exists and validates as a Workflow
    Given the template at "spec/templates/workflow/workflow.template.yaml"
    When it is validated against "spec/schemas/workflow.schema.json"
    Then validation passes with zero errors
    And the manifest kind is "Workflow"
    And at least one state has type "terminal"

  Scenario: The task template exists and validates as an AgentDef
    Given the template at "spec/templates/task/task.template.yaml"
    When it is validated against "spec/schemas/agent-def.schema.json"
    Then validation passes with zero errors
    And the manifest kind is "AgentDef"
    And it declares at least one capability

  Scenario: The expert template exists and validates as an AgentDef
    Given the template at "spec/templates/expert/expert.template.yaml"
    When it is validated against "spec/schemas/agent-def.schema.json"
    Then validation passes with zero errors
    And the manifest kind is "AgentDef"
    And it declares a "review" capability with a context_slice input

  # ─── The version: field ─────────────────────────────────────────────────────

  Scenario: A workflow manifest with a SemVer version passes validation
    Given a Workflow manifest with metadata.version "1.2.3"
    When it is validated against the workflow schema
    Then validation passes with zero errors

  Scenario: An AgentDef manifest with a SemVer version passes validation
    Given an AgentDef manifest with metadata.version "2.0.0-rc.1"
    When it is validated against the agent-def schema
    Then validation passes with zero errors

  Scenario: The version field is optional and backward-compatible
    Given a Workflow manifest with no metadata.version field
    When it is validated against the workflow schema
    Then validation passes with zero errors

  Scenario: A non-SemVer version string is rejected
    Given a Workflow manifest with metadata.version "v1"
    When it is validated against the workflow schema
    Then validation fails
    And the error references the "version" field

  Scenario: An empty version string is rejected
    Given an AgentDef manifest with metadata.version set to ""
    When it is validated against the agent-def schema
    Then validation fails
    And the error references the "version" field

  # ─── make validate-spec integration ─────────────────────────────────────────

  Scenario: make validate-spec validates the template manifests
    Given a Makefile exists at the repository root
    When the "validate-spec" target is inspected
    Then it validates manifests under "spec/templates/workflow/"
    And it validates manifests under "spec/templates/task/"
    And it validates manifests under "spec/templates/expert/"

  # ─── Schema self-consistency ─────────────────────────────────────────────────

  Scenario: The workflow schema declares an optional version property
    When "spec/schemas/workflow.schema.json" is parsed
    Then the metadata properties include a "version" field
    And "version" is not listed in the metadata required array

  Scenario: The agent-def schema declares an optional version property
    When "spec/schemas/agent-def.schema.json" is parsed
    Then the metadata properties include a "version" field
    And "version" is not listed in the metadata required array
