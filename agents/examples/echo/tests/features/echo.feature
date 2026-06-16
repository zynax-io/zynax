# SPDX-License-Identifier: Apache-2.0
Feature: Echo capability returns the input payload
  As an agent author copying the reference example
  I want the echo agent to return exactly what it was given
  So that I can verify the SDK request/response round-trip

  Scenario: Echo streams progress then completes with the input payload
    Given an EchoAgent
    When echo is called with payload {"hello": "world"}
    Then the stream emits a PROGRESS event
    And the final event is COMPLETED
    And the completed payload echoes {"hello": "world"}

  Scenario: Echo handles an empty payload
    Given an EchoAgent
    When echo is called with an empty payload
    Then the final event is COMPLETED
    And the completed payload echoes {}
