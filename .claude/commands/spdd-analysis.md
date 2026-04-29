# /spdd-analysis

Scan the codebase for context relevant to a feature, extract domain keywords, surface risks, and produce a structured analysis ready for Canvas generation.

## Instructions

Given the feature or issue in $ARGUMENTS:

1. **Read the relevant AGENTS.md files** for the layers the feature will touch (root, services/, agents/, protos/, etc.)
2. **Scan docs/adr/INDEX.md** — identify all ADRs that constrain or inform this feature
3. **Identify existing domain entities** in the codebase that the feature will extend or interact with
4. **Identify new domain entities** the feature introduces that don't exist yet
5. **Map the system boundaries** — which services and gRPC contracts are involved
6. **Surface risks** in three categories:
   - Breaking changes (proto fields, API contracts, gRPC method signatures)
   - Performance risks (new hot paths, stream cardinality)
   - Security risks (new external inputs, trust boundaries crossed)
7. **Classify all context** by trust level:
   - Tier 1 (public-safe): architecture patterns, domain entity names, ADR references → goes in Canvas
   - Tier 2 (private): internal hostnames, deployment details, credentials → flag for canvas.private.md
   - Tier 3 (ephemeral): current branch state, test output → session-only, never persisted

## Output Format

### Existing Concepts (from codebase)
- <entity/package/service>: <brief description of current role>

### New Concepts (introduced by this feature)
- <entity/type/interface>: <what it will do>

### ADR Constraints
- ADR-NNN (<title>): <how it constrains this feature>

### System Boundaries Touched
- <service>/<package>: <nature of change>

### Risks
| Risk | Category | Severity | Mitigation |
|------|----------|----------|------------|
| | | | |

### Tier 2 Flags (move to canvas.private.md)
- <any sensitive context that surfaced — list or "None">

### Recommended Design Direction
<2–3 sentences on the recommended approach before generating the Canvas>

## Input

$ARGUMENTS — GitHub issue number, issue URL, or feature description
