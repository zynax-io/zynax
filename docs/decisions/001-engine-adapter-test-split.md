# 001: EngineAdapter Test Two-File Split

**Date:** 2026-04-21  **Author:** M1 Engineering

## Context

`EngineAdapterService` is the most complex service contract in M1: it has a
`StartWorkflow` / `CancelWorkflow` / `CompleteWorkflow` / `FailWorkflow` family
(workflow lifecycle) and a separate `WatchWorkflow` bidirectional streaming RPC
(signal delivery). Both share the same proto and the same stub server, but they
exercise entirely different state: lifecycle RPCs manipulate `WorkflowRun` records
while `WatchWorkflow` tests a pub/sub signal channel.

The feature file for `EngineAdapterService` has 40+ scenarios. Putting all step
definitions in a single `steps_test.go` would produce a ~600-line file with two
logically separate concerns sharing a struct, making it hard to locate step
definitions and hard to run a subset in isolation.

## Decision

Split the test suite into two files in the same Go package
(`package engine_adapter_service_test`):

- `lifecycle_steps_test.go` — `TestLifecycle` function, `@lifecycle` tag filter,
  step definitions for Start/Cancel/Complete/Fail RPCs
- `signals_steps_test.go` — `TestSignals` function, `@signals` tag filter,
  step definitions for WatchWorkflow delivery

Both files share the same `engineStub` and `runRecord` types defined in
`lifecycle_steps_test.go`. Because they are in the same package, `signals_steps_test.go`
can access those types directly without an import.

Each `TestXxx` function passes a `Tags` option to godog so the two test functions
exercise non-overlapping scenario sets even when run together with `./...`.

## Alternatives considered

- **Single file, all scenarios:** Rejected — would exceed 600 lines, mixing two
  logically independent concerns.
- **Separate packages:** Rejected — would require duplicating the `engineStub`
  type and the `testserver` setup, adding maintenance burden. Shared types are the
  point of the same-package pattern.
- **Sub-directories:** Rejected — go test package granularity is the directory;
  sub-directories would require separate `go.mod` scoping which conflicts with the
  single `protos/tests/go.mod` design.

## Consequences

- Developers adding EngineAdapter lifecycle tests edit `lifecycle_steps_test.go`.
  Developers adding signal delivery tests edit `signals_steps_test.go`.
- Running `go test -run TestLifecycle` or `go test -run TestSignals` executes only
  the relevant subset.
- This pattern is available to any future service suite that grows beyond ~150 lines
  of step definitions. See `protos/tests/AGENTS.md §Two-File Split Pattern`.
