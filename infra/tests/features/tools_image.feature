# SPDX-License-Identifier: Apache-2.0
# Zynax — Developer Tools Image BDD Contract
#
# This file is the SPECIFICATION. It is written BEFORE the implementation.
# It describes what the zynax-tools Docker image must contain and how it
# must behave for all make targets to work correctly.
# See infra/AGENTS.md for infrastructure governance rules.
#
# Business context: All CI and local development tooling runs inside a
# single Docker image to guarantee reproducible output across machines and
# CI runners. The image is the contract between the Makefile targets and
# the tools they depend on. (ADR-003)

Feature: Developer tools image — Alpine-based with all CI tools
  As a contributor
  I want a single `make build-tools` to produce a minimal image with every tool
  So that all make targets work identically on any machine and in CI

  Background:
    Given the image "zynax-tools:local" has been built with "make build-tools"

  # ─── Base image ───────────────────────────────────────────────────────────

  Scenario: The final image is based on Alpine Linux
    When the image OS is inspected
    Then the base distribution is Alpine Linux
    And the image does not contain apt or dpkg

  Scenario: The image is smaller than 2 GB
    When the image size is inspected
    Then the compressed image size is less than 2 GB

  # ─── buf ──────────────────────────────────────────────────────────────────

  Scenario: buf 1.47.2 is installed and on PATH
    When "buf --version" is run inside the image
    Then the output contains "1.47.2"

  Scenario: buf lint passes on the proto workspace
    When "buf lint" is run from protos/ inside the image
    Then exit code is 0

  Scenario: buf generate runs without error
    When "buf generate --template buf.gen.yaml" is run from protos/ inside the image
    Then exit code is 0
    And Go stubs appear under protos/generated/go/
    And Python stubs appear under protos/generated/python/

  # ─── Go tools ────────────────────────────────────────────────────────────

  Scenario: Go 1.22 is available
    When "go version" is run inside the image
    Then the output contains "go1.22"

  Scenario: golangci-lint is installed
    When "golangci-lint --version" is run inside the image
    Then exit code is 0
    And the output contains "golangci-lint"

  Scenario: govulncheck is installed
    When "govulncheck -version" is run inside the image
    Then exit code is 0

  Scenario: protoc-gen-go is installed and on PATH
    When "which protoc-gen-go" is run inside the image
    Then exit code is 0

  Scenario: protoc-gen-go-grpc is installed and on PATH
    When "which protoc-gen-go-grpc" is run inside the image
    Then exit code is 0

  Scenario: godog is installed and on PATH
    When "godog --version" is run inside the image
    Then exit code is 0
    And the output contains "godog"

  # ─── Python tools ─────────────────────────────────────────────────────────

  Scenario: Python 3.12 is available
    When "python --version" is run inside the image
    Then the output contains "Python 3.12"

  Scenario: uv is installed and pinned
    When "uv --version" is run inside the image
    Then exit code is 0
    And the output contains "uv"

  Scenario: ruff is installed
    When "uv run ruff --version" is run inside the image
    Then exit code is 0

  Scenario: mypy is installed
    When "uv run mypy --version" is run inside the image
    Then exit code is 0

  Scenario: pytest is installed
    When "uv run pytest --version" is run inside the image
    Then exit code is 0

  Scenario: pytest-bdd is importable
    When "python -c 'import pytest_bdd'" is run inside the image
    Then exit code is 0

  Scenario: pytest-cov is importable
    When "python -c 'import pytest_cov'" is run inside the image
    Then exit code is 0

  Scenario: bandit is installed
    When "uv run bandit --version" is run inside the image
    Then exit code is 0

  Scenario: pip-audit is installed
    When "uv run pip-audit --version" is run inside the image
    Then exit code is 0

  # ─── make targets ────────────────────────────────────────────────────────

  Scenario: make lint-protos passes inside the image
    When "make lint-protos" is run with the image
    Then exit code is 0

  Scenario: make generate-protos produces stubs inside the image
    When "make generate-protos" is run with the image
    Then exit code is 0
    And protos/generated/ contains at least one .go file
    And protos/generated/ contains at least one .py file
