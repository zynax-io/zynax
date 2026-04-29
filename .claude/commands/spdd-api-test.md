# /spdd-api-test

Generate functional test scripts for a feature based on its REASONS Canvas definition of done.

## Instructions

Read the Canvas at $ARGUMENTS. Focus on the **R — Requirements** section (definition of done) and the **O — Operations** section (what was implemented).

Generate test coverage for three scenario types:

1. **Normal path**: happy path that validates the primary requirement
2. **Boundary conditions**: edge cases at the limits of the specification (empty inputs, maximum sizes, timeout boundaries)
3. **Error scenarios**: invalid inputs, missing dependencies, service unavailability

For each gRPC service touched, generate:
- A `grpcurl` or `buf curl` command demonstrating the call
- The expected response shape (field names and types, not values)
- The expected error code for the error scenario

For BDD-testable boundaries, generate Gherkin scenarios in `.feature` file format that could be added to `protos/tests/<service>/features/<service>.feature`.

## Output Format

### Normal Path
```bash
# grpcurl command + expected response skeleton
```

### Boundary Conditions
```bash
# grpcurl commands for each boundary
```

### Error Scenarios
```bash
# grpcurl commands + expected gRPC status codes
```

### New BDD Scenarios (if applicable)
```gherkin
Scenario: <title>
  Given ...
  When ...
  Then ...
```

## Input

$ARGUMENTS — path to canvas.md (e.g., docs/spdd/205-spdd/canvas.md)
