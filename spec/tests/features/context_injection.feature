# SPDX-License-Identifier: Apache-2.0
# Zynax — Declarative Context-Injection BDD Contract
#
# This file is the SPECIFICATION. It is written BEFORE the implementation
# (ADR-016). It describes what the context-injection block must enforce and how
# `zynax apply`/`zynax validate` bind a scenario's declared context into a
# Workflow action's input through the EXISTING {{ .ctx.* }} template surface.
# See spec/AGENTS.md for spec governance and
# docs/spdd/1387-context-injection/canvas.md for the A2 approach.
#
# Business context: today a demo grounds itself by hand-pasting a git diff into a
# workflow prompt — unbounded, mixed with instructions, re-pasted every run.
# #1387 makes that injection DECLARATIVE: a scenario declares file-rooted content
# sources and a token budget once; the CLI binds them into {{ .ctx.<key> }}
# references, bounded and isolated. The block is STRICTLY DATA-ONLY (ADR-013/035)
# — it can never carry provider/model/endpoint/URL or any routing-redirecting
# field. It fills the reserved spec.context slot of a Scenario index (#1385).

Feature: Declarative context-injection block for demo scenarios
  As a scenario author
  I want to declare the content my scenario injects, bounded and data-only
  So that `zynax apply` grounds the run in real content with no prompt hand-edit

  Background:
    Given the JSON Schema at "spec/schemas/context-injection.schema.json" is loaded

  # ─── Valid block + substitution ─────────────────────────────────────────────

  Scenario: A valid context block compiles and substitutes into {{ .ctx.* }}
    Given a scenario whose spec.context declares a source key "diff" sourced from "diff.patch"
    And the block declares a max_tokens budget
    And the scenario's Workflow action input references "{{ .ctx.diff }}"
    When the scenario is validated and applied
    Then the file content is bound into the action input in place of {{ .ctx.diff }}
    And no template reference remains unresolved

  Scenario: A reference the block does not supply fails fast
    Given a scenario Workflow that references "{{ .ctx.absent }}"
    And the context block declares no source with key "absent"
    When the scenario is validated
    Then validation fails because the {{ .ctx.absent }} reference is unresolved

  # ─── Data-only safeguard (load-bearing) ─────────────────────────────────────

  Scenario: A context block carrying a routing field is rejected at compile time
    Given a context block whose source also declares a "provider" field
    When the block is parsed
    Then it is rejected because a context block is data-only and may never carry provider/model/endpoint/URL

  Scenario Outline: Each routing-redirecting field is forbidden
    Given a context block that declares a "<field>" field
    When the block is parsed
    Then the block is rejected citing the forbidden "<field>" field

    Examples:
      | field    |
      | provider |
      | model    |
      | endpoint |
      | url      |
      | base_url |
      | api_key  |

  Scenario: An unknown top-level field is rejected
    Given a context block that declares a "language" field
    When the block is parsed
    Then it is rejected because a context block carries sources/max_tokens/overflow only

  # ─── Bounds enforcement ─────────────────────────────────────────────────────

  Scenario: An over-budget block is truncated oldest-first deterministically
    Given a context block whose combined sources exceed max_tokens
    And the overflow policy is "truncate-oldest"
    When the block is resolved
    Then the earliest-declared sources are dropped first until the budget is met

  Scenario: An over-budget block with overflow error fails resolution
    Given a context block whose combined sources exceed max_tokens
    And the overflow policy is "error"
    When the block is resolved
    Then resolution fails citing the exceeded max_tokens budget

  # ─── Isolation & containment ────────────────────────────────────────────────

  Scenario: One scenario's context never reaches another
    Given two scenarios that each declare a context key "diff" from different files
    When both blocks are resolved
    Then each scenario sees only its own content and never the other's

  Scenario: A source file that escapes the scenario directory is rejected
    Given a context block whose source file is "../../etc/passwd"
    When the block is resolved
    Then resolution fails because source files must stay within the scenario directory
