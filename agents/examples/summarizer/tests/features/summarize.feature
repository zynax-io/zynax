# SPDX-License-Identifier: Apache-2.0
Feature: Summarize capability condenses documents
  As an agent author copying the reference example
  I want the summarizer to return a short extractive summary
  So that I can see a richer SDK request/response pattern with validation

  Scenario: Summarize returns the first sentences of the documents
    Given a SummarizerAgent
    When summarize is called with documents:
      | document                                  |
      | Cats are great. They purr a lot.          |
      | Dogs are loyal. They fetch sticks.        |
    Then the final event is COMPLETED
    And the summary contains "Cats are great"
    And the summary contains "Dogs are loyal"
    And the document_count is 2

  Scenario: Summarize fails when documents are missing
    Given a SummarizerAgent
    When summarize is called with no documents
    Then the final event is FAILED
    And the error code is "EMPTY_INPUT"
