# SPDX-License-Identifier: Apache-2.0
# Zynax — http-adapter BDD Contract Specification
#
# This file is the SPECIFICATION. It is written BEFORE the implementation.
# It describes the http-adapter-specific behaviour on top of the generic
# AgentService contract (agent_service.feature covers the universal invariants).
#
# Business context: The http-adapter turns any REST API into a Zynax capability
# via config-only route mapping in AgentDef YAML. No code changes to the target
# service; no SDK import (ADR-013).
#
# Canvas: docs/spdd/380-http-adapter/canvas.md
# Parent epic: #377 (M5 Adapter Library)

Feature: http-adapter — config-driven REST capability proxy
  As a platform operator
  I want a config-driven HTTP adapter
  So that any REST API becomes a Zynax capability by declaring routes in AgentDef YAML
  without any code changes to the target service

  Background:
    Given an http-adapter configured with the following route:
      | capability_name | method | url                           |
      | call_api        | POST   | http://upstream-test/v1/data  |
    And the adapter is registered with AgentRegistryService

  # ─── Happy path ─────────────────────────────────────────────────────────────

  Scenario: Upstream 2xx response produces TASK_EVENT_TYPE_COMPLETED with response body
    Given the upstream returns HTTP 200 with body: {"result": "ok"}
    And a valid ExecuteCapabilityRequest for capability "call_api"
    And the input payload is valid JSON: {"key": "value"}
    When ExecuteCapability is called
    Then the final TaskEvent has event_type COMPLETED
    And the COMPLETED event payload contains the upstream response body
    And the stream closes cleanly after the COMPLETED event

  Scenario: Static request headers declared in RouteConfig are forwarded to upstream
    Given the RouteConfig for "call_api" declares header "X-Api-Key" with value from env var ref
    And the upstream returns HTTP 200
    When ExecuteCapability is called with capability "call_api"
    Then the upstream receives the "X-Api-Key" header
    And the final TaskEvent has event_type COMPLETED

  Scenario: GetCapabilitySchema returns schemas declared in the AgentDef YAML
    When GetCapabilitySchema is called with capability_name "call_api"
    Then the gRPC status is OK
    And the response input_schema_json matches the schema declared in RouteConfig
    And the response output_schema_json matches the schema declared in RouteConfig
    And the response description is non-empty

  # ─── Progress ticker ─────────────────────────────────────────────────────────

  Scenario: Slow upstream triggers PROGRESS event before terminal event
    Given the upstream delays the response by 3 seconds
    And a valid ExecuteCapabilityRequest for capability "call_api" with timeout_seconds 10
    When ExecuteCapability is called
    Then the stream emits at least one TaskEvent with event_type PROGRESS before the terminal event
    And every PROGRESS event has task_id echoed and timestamp populated
    And the final TaskEvent has event_type COMPLETED or FAILED

  Scenario: Fast upstream completes without emitting a PROGRESS event
    Given the upstream returns HTTP 200 immediately
    And a valid ExecuteCapabilityRequest for capability "call_api" with timeout_seconds 10
    When ExecuteCapability is called
    Then the stream emits exactly one TaskEvent
    And that TaskEvent has event_type COMPLETED

  # ─── Timeout handling ────────────────────────────────────────────────────────

  Scenario: Upstream timeout breach emits FAILED with TIMEOUT code
    Given the upstream delays the response by 5 seconds
    And an ExecuteCapabilityRequest for capability "call_api" with timeout_seconds 1
    When ExecuteCapability is called
    Then the stream receives a TaskEvent of type FAILED within 2 seconds
    And the CapabilityError code is "TIMEOUT"
    And the CapabilityError message does not contain credential values or raw response bodies

  # ─── Upstream error mapping ──────────────────────────────────────────────────

  Scenario: Upstream 4xx response produces TASK_EVENT_TYPE_FAILED with UPSTREAM_ERROR
    Given the upstream returns HTTP 400 with body: {"error": "bad request"}
    When ExecuteCapability is called with capability "call_api"
    Then the final TaskEvent has event_type FAILED
    And the CapabilityError code is "UPSTREAM_ERROR"
    And the CapabilityError message is non-empty and sanitised
    And the CapabilityError message does not contain the raw upstream response body

  Scenario: Upstream 5xx response produces TASK_EVENT_TYPE_FAILED with UPSTREAM_ERROR
    Given the upstream returns HTTP 500 with body: {"error": "internal server error"}
    When ExecuteCapability is called with capability "call_api"
    Then the final TaskEvent has event_type FAILED
    And the CapabilityError code is "UPSTREAM_ERROR"
    And the CapabilityError details contain upstream_status "500"

  Scenario: Upstream connection refused produces TASK_EVENT_TYPE_FAILED with UPSTREAM_ERROR
    Given the upstream is not reachable
    When ExecuteCapability is called with capability "call_api"
    Then the final TaskEvent has event_type FAILED
    And the CapabilityError code is "UPSTREAM_ERROR"
    And the CapabilityError message does not contain internal network details

  # ─── Input validation ────────────────────────────────────────────────────────

  Scenario: input_payload failing JSON Schema validation is rejected before HTTP call
    Given the RouteConfig for "call_api" declares an input_schema requiring field "key"
    And an ExecuteCapabilityRequest with input_payload: {"wrong_field": "value"}
    When ExecuteCapability is called
    Then no HTTP request is made to the upstream
    And the final TaskEvent has event_type FAILED
    And the CapabilityError code is "INVALID_INPUT"

  Scenario: Non-JSON input_payload is rejected before HTTP call
    Given an ExecuteCapabilityRequest with input_payload set to "not valid json"
    When ExecuteCapability is called with capability "call_api"
    Then no HTTP request is made to the upstream
    And the gRPC status is INVALID_ARGUMENT or the CapabilityError code is "INVALID_INPUT"

  # ─── SSRF prevention ─────────────────────────────────────────────────────────

  Scenario: URL fields in input_payload are ignored — adapter always calls static RouteConfig URL
    Given the upstream test server is listening at the configured static URL
    And an ExecuteCapabilityRequest with input_payload containing a "url" field pointing elsewhere
    When ExecuteCapability is called with capability "call_api"
    Then the HTTP request is made to the static URL declared in RouteConfig
    And the HTTP request is NOT made to the URL in input_payload

  Scenario: Upstream URL is never constructed from input_payload fields
    Given an ExecuteCapabilityRequest with input_payload: {"host": "evil.example.com", "path": "/steal"}
    When ExecuteCapability is called with capability "call_api"
    Then the upstream receives the request at the static RouteConfig URL only

  # ─── Capability routing ──────────────────────────────────────────────────────

  Scenario: Unknown capability returns NOT_FOUND without entering the stream
    Given an ExecuteCapabilityRequest for capability "nonexistent_route"
    When ExecuteCapability is called
    Then the gRPC status is NOT_FOUND
    And no TaskEvent is emitted
    And the error message contains "nonexistent_route"

  Scenario: Empty capability_name is rejected before routing
    Given an ExecuteCapabilityRequest with capability_name set to ""
    When ExecuteCapability is called
    Then the gRPC status is INVALID_ARGUMENT
    And the error message mentions "capability_name"

  # ─── Response body safety ────────────────────────────────────────────────────

  Scenario: Oversized upstream response body is handled safely without memory exhaustion
    Given the upstream returns HTTP 200 with a response body larger than 10 MB
    When ExecuteCapability is called with capability "call_api"
    Then the final TaskEvent has event_type FAILED
    And the CapabilityError code is "UPSTREAM_ERROR"
    And the CapabilityError message indicates the response was too large

  # ─── Registration lifecycle ──────────────────────────────────────────────────

  Scenario: Adapter registers with AgentRegistryService on startup
    Given the http-adapter starts with a valid AdapterConfig
    When the adapter initialises
    Then AgentRegistryService.RegisterAgent is called with the configured agent_id
    And the registered AgentDef contains all capabilities declared in the config
    And the registered endpoint matches the adapter's configured bind address

  Scenario: Adapter deregisters from AgentRegistryService on graceful shutdown
    Given the http-adapter is running and registered
    When a SIGTERM signal is received
    Then AgentRegistryService.DeregisterAgent is called with the configured agent_id
    And the gRPC server performs a graceful stop
