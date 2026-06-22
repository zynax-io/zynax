# SPDX-License-Identifier: Apache-2.0
# Contract spec for the adk-adapter (ADR-038) — a Go-native AI-framework adapter
# that embeds Google ADK (Go) agents as Zynax capabilities. Written before the
# bridge implementation (ADR-016, BDD-first). The S2 skeleton (#1478) delivers
# config/registry/health wiring and routes ExecuteCapability to a "not wired"
# terminal event; the ADK Runner -> TaskEvent bridge lands in S3 (#1479).

Feature: adk-adapter — Google ADK (Go) capability adapter
  As a platform operator
  I want an adk-adapter that registers ADK-backed capabilities and serves AgentService
  So that workflow steps can dispatch multi-step, tool-using reasoning by name
  without importing the ADK framework into the control plane

  Background:
    Given an adk-adapter configured with model provider "ollama"
    And the adapter declares a capability "triage" with an instruction and JSON schemas

  # ─── Lifecycle: register + health (S2 skeleton, #1478) ───────────────────

  Scenario: Adapter registers its capabilities and reports SERVING on startup
    When the adapter starts
    Then it registers an AgentDef with AgentRegistryService
    And the gRPC health status is SERVING
    And on graceful shutdown it deregisters and reports NOT_SERVING

  # ─── GetCapabilitySchema (S2 skeleton, #1478) ────────────────────────────

  Scenario: GetCapabilitySchema returns the declared schemas for a known capability
    When GetCapabilitySchema is called with capability_name "triage"
    Then the gRPC status is OK
    And the response carries the declared input and output JSON schemas

  Scenario: GetCapabilitySchema returns NOT_FOUND for an unknown capability
    When GetCapabilitySchema is called with capability_name "missing"
    Then the gRPC status is NOT_FOUND

  # ─── ExecuteCapability validation (S2 skeleton, #1478) ───────────────────

  Scenario: ExecuteCapability rejects a request with an empty task_id
    Given a valid ExecuteCapabilityRequest for capability "triage"
    But the task_id is empty
    When ExecuteCapability is called
    Then the gRPC status is INVALID_ARGUMENT

  Scenario: ExecuteCapability emits FAILED INVALID_INPUT for an unknown capability
    Given a valid ExecuteCapabilityRequest for capability "missing"
    When ExecuteCapability is called
    Then the final TaskEvent has event_type FAILED
    And the CapabilityError code is "INVALID_INPUT"

  # ─── ExecuteCapability dispatch via the ADK Runner bridge (S3 target, #1479) ──

  Scenario: A known capability streams PROGRESS then a terminal COMPLETED
    Given a valid ExecuteCapabilityRequest for capability "triage"
    When ExecuteCapability is called
    Then the stream emits at least one TaskEvent with event_type PROGRESS
    And the final TaskEvent has event_type COMPLETED
    And the COMPLETED payload validates against the declared output_schema_json
