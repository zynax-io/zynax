# SPDX-License-Identifier: Apache-2.0
Feature: API Gateway

  # ── Existing cross-cutting scenarios ────────────────────────────────────

  Scenario: POST /api/v1/agents returns 201 with agent_id
    Given a valid agent registration request body
    When POST /api/v1/agents is called with a valid API key
    Then the HTTP status is 201
    And the response contains a non-empty agent_id

  Scenario: Missing auth token returns 401
    When any API endpoint is called without Authorization header
    Then the HTTP status is 401
    And the response code is "UNAUTHENTICATED"

  Scenario: Insufficient permissions returns 403
    Given a token with permissions ["tasks:read"]
    When POST /api/v1/agents is called (requires agents:write)
    Then the HTTP status is 403

  Scenario: Rate limit exceeded returns 429
    Given 101 requests in 1 minute from the same client
    When the 102nd request is made
    Then the HTTP status is 429
    And Retry-After header is present

  Scenario: gRPC NOT_FOUND maps to 404
    When GET /api/v1/agents/does-not-exist is called
    Then the HTTP status is 404

  Scenario: Internal gRPC errors return 500 without leaking details
    Given the upstream service returns a gRPC INTERNAL error
    When the corresponding REST endpoint is called
    Then the HTTP status is 500
    And the response message is exactly "internal error"

  # ── POST /api/v1/apply — Workflow kind (M4 step 1, issue #315) ──────────

  Scenario: POST /api/v1/apply with valid Workflow YAML returns 202 with run_id
    Given a WorkflowCompilerService that compiles the manifest successfully
    And an EngineAdapterService that accepts the workflow submission
    When POST /api/v1/apply is called with a valid kind: Workflow YAML body
    Then the HTTP status is 202
    And the response contains a non-empty run_id

  Scenario: POST /api/v1/apply dry-run returns 200 with warnings and no run_id
    Given a WorkflowCompilerService that compiles the manifest with a warning
    When POST /api/v1/apply is called with kind: Workflow YAML and dry_run=true
    Then the HTTP status is 200
    And the response contains dry_run: true
    And the response contains a warnings list
    And the response does not contain a run_id

  Scenario: POST /api/v1/apply with invalid YAML returns 422 with error message
    Given a WorkflowCompilerService that returns a compilation error
    When POST /api/v1/apply is called with kind: Workflow YAML
    Then the HTTP status is 422
    And the response contains a non-empty errors list

  Scenario: POST /api/v1/apply with unknown kind returns 400
    When POST /api/v1/apply is called with kind: SomethingUnknown in the body
    Then the HTTP status is 400
    And the response code is "UNSUPPORTED_KIND"

  Scenario: POST /api/v1/apply with missing kind field returns 400
    When POST /api/v1/apply is called with a YAML body that has no kind field
    Then the HTTP status is 400
    And the response code is "UNSUPPORTED_KIND"

  Scenario: POST /api/v1/apply when engine adapter is unavailable returns 503
    Given a WorkflowCompilerService that compiles the manifest successfully
    And an EngineAdapterService that returns UNAVAILABLE
    When POST /api/v1/apply is called with a valid kind: Workflow YAML body
    Then the HTTP status is 503
    And the response code is "ENGINE_UNAVAILABLE"

  Scenario: POST /api/v1/apply with body larger than 1 MB returns 413
    When POST /api/v1/apply is called with a request body exceeding 1 MB
    Then the HTTP status is 413

  # ── GET /api/v1/workflows/{id} (M4 step 1, issue #315) ──────────────────

  Scenario: GET /api/v1/workflows/{id} returns workflow status and current state
    Given a submitted workflow with run_id "smoke-run-001"
    When GET /api/v1/workflows/smoke-run-001 is called
    Then the HTTP status is 200
    And the response contains a status field
    And the response contains a current_state field

  Scenario: GET /api/v1/workflows/{id} for unknown run_id returns 404
    Given the engine adapter does not know about run_id "ghost-run"
    When GET /api/v1/workflows/ghost-run is called
    Then the HTTP status is 404

  # ── POST /api/v1/apply — AgentDef kind (M4 step 2, issue #316) ──────────

  Scenario: POST /api/v1/apply with valid AgentDef YAML returns 201 with agent_id
    Given an AgentRegistryService that accepts the registration
    When POST /api/v1/apply is called with a valid kind: AgentDef YAML body
    Then the HTTP status is 201
    And the response contains a non-empty agent_id

  Scenario: POST /api/v1/apply with duplicate AgentDef returns 409
    Given an AgentRegistryService that returns ALREADY_EXISTS
    When POST /api/v1/apply is called with a valid kind: AgentDef YAML body
    Then the HTTP status is 409
    And the response code is "ALREADY_EXISTS"

  # ── POST /api/v1/apply — Idempotent Apply (#485) ────────────────────────

  Scenario: POST /api/v1/apply with same manifest while workflow is running returns existing run_id
    Given a WorkflowCompilerService that compiles the manifest successfully
    And an EngineAdapterService that reports a running workflow for the derived manifest hash
    When POST /api/v1/apply is called with a valid kind: Workflow YAML body
    Then the HTTP status is 202
    And the response contains a non-empty run_id
    And the response has status "existing"

  Scenario: POST /api/v1/apply after the workflow completes starts a new workflow run
    Given a WorkflowCompilerService that compiles the manifest successfully
    And an EngineAdapterService that reports a completed workflow for the derived manifest hash
    And an EngineAdapterService that accepts a new workflow submission for re-run
    When POST /api/v1/apply is called with a valid kind: Workflow YAML body
    Then the HTTP status is 202
    And the response contains a non-empty run_id
    And the response has status "new"

  # ── GET /api/v1/workflows/{id}/logs (M4 step 4, issue #318) ─────────────

  Scenario: GET /api/v1/workflows/{id}/logs streams SSE events and closes on terminal
    Given a submitted workflow with run_id "log-run-001"
    And the engine adapter streams a state-entered event and then a completed event
    When GET /api/v1/workflows/log-run-001/logs is called
    Then the HTTP status is 200
    And the Content-Type is "text/event-stream"
    And the response body contains 2 SSE data lines

  Scenario: GET /api/v1/workflows/{id}/logs for unknown run_id returns 404
    Given the engine adapter does not know about run_id "ghost-log-run"
    When GET /api/v1/workflows/ghost-log-run/logs is called
    Then the HTTP status is 404
