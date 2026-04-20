# SPDX-License-Identifier: Apache-2.0
Feature: API Gateway

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
