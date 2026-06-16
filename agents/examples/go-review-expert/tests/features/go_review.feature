# SPDX-License-Identifier: Apache-2.0
Feature: Go review expert flags common Go issues
  As an agent author copying the reference expert example
  I want the go-review-expert to return structured findings
  So that I can see an expert-style agent built on the SDK

  Scenario: A clean diff is approved with no findings
    Given a GoReviewExpert
    When go_review is called with diff:
      """
      func Add(a, b int) int {
          return a + b
      }
      """
    Then the final event is COMPLETED
    And the finding_count is 0
    And the review is approved

  Scenario: A panic is flagged as an error and rejects the review
    Given a GoReviewExpert
    When go_review is called with diff:
      """
      func mustParse(s string) int {
          panic("not implemented")
      }
      """
    Then the final event is COMPLETED
    And there is an "error" finding with message containing "panic"
    And the review is not approved

  Scenario: Missing diff fails the review
    Given a GoReviewExpert
    When go_review is called with an empty diff
    Then the final event is FAILED
    And the error code is "EMPTY_INPUT"
