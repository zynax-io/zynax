# SPDX-License-Identifier: Apache-2.0
Feature: Event Bus

  Scenario: Published event reaches all subscribers
    Given consumers "a" and "b" subscribe to topic "keel.v1.task-broker.task.completed"
    When an event is published to that topic
    Then both "a" and "b" receive the event

  Scenario: Subscriber on different topic does not receive event
    Given consumer "c" subscribes to "keel.v1.task-broker.task.assigned"
    When an event is published to "keel.v1.task-broker.task.completed"
    Then consumer "c" does NOT receive the event

  Scenario: Failed delivery is retried with backoff
    Given a subscriber that fails on first attempt
    When an event is published
    Then the event is redelivered at least once

  Scenario: Event is DLQ'd after exhausting retries
    Given a subscriber that always fails
    When an event is published
    And 5 delivery attempts are exhausted
    Then the event appears on the DLQ topic

  Scenario: Durable consumer catches up after being offline
    Given consumer "d" was offline when an event was published
    When consumer "d" reconnects
    Then consumer "d" receives the missed event

  Scenario: Two consumer groups receive same event independently
    Given groups "indexer" and "notifier" both subscribe to the same topic
    When one event is published
    Then both groups receive their own independent copy
