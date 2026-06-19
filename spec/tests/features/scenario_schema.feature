# SPDX-License-Identifier: Apache-2.0
# Zynax — Scenario Index Schema BDD Contract
#
# This file is the SPECIFICATION. It is written BEFORE the implementation
# (ADR-016). It describes what the scenario index schema must enforce and how
# `zynax apply`/`zynax validate` expand a scenario manifest set.
# See spec/AGENTS.md for spec governance rules and
# docs/spdd/1385-scenario-manifest/canvas.md for the A2 approach.
#
# Business context: A scenario wires together a Workflow, the AgentDef(s) that
# supply its capabilities, and a reserved context slot — declaratively, in one
# place — so a user runs their own scenario without imperative edits (ADR-011).
# A Scenario is NOT a server-side composite kind: the index is a client-side
# manifest-set convention (A2, ADR-028 "two kinds, no third schema"). Members
# are applied over the EXISTING /api/v1/apply REST path — no new boundary.

Feature: Scenario index schema for declarative manifest sets
  As a scenario author
  I want to declare a Workflow, its AgentDef(s), and a context slot in one index
  So that `zynax apply` brings the whole scenario up in dependency order

  Background:
    Given the JSON Schema at "spec/schemas/scenario.schema.json" is loaded

  # ─── Valid indexes ─────────────────────────────────────────────────────────

  Scenario: A valid scenario index passes schema validation
    Given a scenario index with kind "Scenario" and apiVersion "zynax.io/v1alpha1"
    And metadata.name is "code-review"
    And spec.members lists an AgentDef member "agent" and a Workflow member "workflow"
    And spec.apply_order is ["agent", "workflow"]
    When the index is validated against the scenario schema
    Then validation passes with zero errors

  Scenario: A scenario index may reserve a context slot
    Given a valid scenario index
    And spec.context declares a data-only key "language" with value "go"
    When the index is validated against the scenario schema
    Then validation passes with zero errors
    And the context slot is treated as a reserved pass-through (semantics owned by #1387)

  # ─── Rejected indexes ──────────────────────────────────────────────────────

  Scenario: An index missing apply_order is rejected
    Given a scenario index whose spec omits apply_order
    When the index is validated against the scenario schema
    Then validation fails with an error citing the missing apply_order

  Scenario: A member with a kind other than Workflow or AgentDef is rejected
    Given a scenario index with a member of kind "Policy"
    When the index is validated against the scenario schema
    Then validation fails because only Workflow and AgentDef members are allowed

  Scenario: A member file path that escapes the scenario directory is rejected
    Given a scenario index whose member file is "../../etc/passwd"
    When the index is validated against the scenario schema
    Then validation fails because member files must be relative to the index directory

  # ─── CLI expansion contract ────────────────────────────────────────────────

  Scenario: zynax validate expands an index and validates each member
    Given a scenario directory containing an index, a Workflow, and an AgentDef
    When "zynax validate" is run against the scenario directory
    Then each member is validated against its own existing schema
    And per-member errors are reported with the member file name

  Scenario: zynax apply submits members over the existing REST path in apply_order
    Given a scenario directory whose apply_order registers the AgentDef before the Workflow
    When "zynax apply" is run against the scenario directory
    Then the AgentDef is submitted first and returns an agent_id
    And the Workflow is submitted next and returns a run_id
    And no new api-gateway endpoint or response shape is introduced

  Scenario: An apply_order id that names no declared member fails fast
    Given a scenario index whose apply_order references an unknown member id
    When "zynax apply" expands the index
    Then expansion fails with a bounded error naming the unknown id
