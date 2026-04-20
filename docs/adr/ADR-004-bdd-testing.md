# ADR-004: BDD as Primary Testing Methodology

**Status:** Accepted  **Date:** 2025-04-01

## Decision
Use Behavior-Driven Development (BDD) with Gherkin feature files as the
primary testing methodology. Tool: `pytest-bdd`.

## Rationale
- Feature files serve as living documentation readable by non-engineers.
- Forces specification-before-implementation discipline.
- Scenarios describe observable behavior — implementation can be refactored
  without rewriting tests.
- BDD naturally produces high-value tests (not implementation-detail tests).
- `pytest-bdd` integrates with the existing pytest ecosystem.

## Consequences
- Every domain feature requires a `.feature` file written before implementation.
- Step definitions live alongside tests in `tests/unit/` or `tests/integration/`.
- Contributors must learn Gherkin (5-minute learning curve — documented in CONTRIBUTING.md).
- CI enforces: no `.feature` file = PR blocked (via custom check).
