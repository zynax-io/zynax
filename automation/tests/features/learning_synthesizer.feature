# SPDX-License-Identifier: Apache-2.0
# automation/tests/features/learning_synthesizer.feature
#
# BDD contract for the learning-synthesizer AgentDef (EPIC #881 — O7,
# #1102; ADR-016: contract before implementation). Steps are bound by
# automation/tests/test_learning_synthesizer.py, which evaluates the
# declarative manifest (automation/workflows/learning-synthesizer.yaml)
# against sample session results — no running platform required (the live
# e2e is O8, #1103).

Feature: Learning-synthesizer AgentDef — human-gated expert updates
  As the self-hosted automation plane
  I want a synthesizer agent that clusters accumulated session learnings
  into proposed updates for the O2 expert AgentDefs
  So that the expert knowledge base improves over time while every apply
  stays human-gated — no manifest is ever auto-edited

  Background:
    Given the AgentDef manifest "automation/workflows/learning-synthesizer.yaml"

  Scenario: Manifest validates against the AgentDef schema
    Then the manifest validates against "spec/schemas/agent-def.schema.json"

  Scenario: The agent exposes exactly the synthesize_learnings capability
    Then the capabilities are exactly "synthesize_learnings"
    And no capability name suggests applying, editing or committing a change

  Scenario: The capability I/O contract matches the canvas
    Then the input requires "session_results, applied_patterns, context_slice"
    And the output requires "proposed_manifest_updates, apply_log_entry, summary"

  Scenario: Context slice is the learning record plus the expert manifests
    Then the context slice files are exactly
      | docs/ai-learnings/*.md             |
      | automation/workflows/experts/*.yaml |
    And the context budget is 4000 tokens, matching the manifest label

  Scenario: The recurrence rule is declared in the contract
    Then the proposal recurrence field declares a minimum of 2

  Scenario: Sample session results emit proposed_manifest_updates
    Given sample session results where two patterns recur in two sessions,
      one pattern is seen in a single session, and one pattern was already
      applied per the apply log
    When the synthesize_learnings contract is evaluated over the samples
    Then proposed_manifest_updates[] contains exactly the two recurring,
      not-yet-applied patterns
    And the single-session pattern is not proposed
    And the already-applied pattern is not re-proposed
    And the result validates against the capability output schema

  Scenario: Proposals target existing O2 expert AgentDefs
    When the synthesize_learnings contract is evaluated over the samples
    Then every target_manifest is an existing expert AgentDef under
      "automation/workflows/experts/"

  Scenario: Apply is human-gated — no manifest is auto-edited
    Then the only expressible proposal status is "pending-human-review"
    And every emitted proposal carries status "pending-human-review"
    And no capability matches a prohibited auto action
      | merge | push | bump-dependency | close-issue | delete-branch | force-push |
    And the expert AgentDef manifests on disk are unchanged after synthesis
    And the manifest is labelled human-gated
