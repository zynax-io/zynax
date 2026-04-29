# REASONS Canvas — <Feature Title>

> **All content in this Canvas is Tier 1 (public-safe).**
> Tier 2 content (internal hostnames, IPs, credentials, deployment specifics) belongs in
> `canvas.private.md` (gitignored). Run `/spdd-security-review <path>` before committing.

**Issue:** #<number>
**Author:** <maintainer name>
**Date:** YYYY-MM-DD
**Status:** Draft

---

## R — Requirements

> Problem statement: what breaks or is missing without this feature?
> Definition of done: the observable outcomes that confirm delivery.
> Be specific — vague requirements produce vague code.

<Write 3–6 bullet points describing the problem and the done state.>

---

## E — Entities

> Domain entities introduced or modified by this feature and their relationships.
> Use a list or ASCII diagram. Names must be public-safe abstractions.
> Do NOT include real server names, internal service hostnames, or database connection strings.

<List domain entities and their relationships.>

---

## A — Approach

> Solution strategy. Explicitly state what we WILL do AND what we WON'T do.
> Reference the ADRs that govern the choice. One-way doors need an ADR citation.

**We will:**
- <approach item>

**We will NOT:**
- <explicit out-of-scope item (defer to a future milestone or issue)>

**Governing ADRs:** ADR-NNN (<title>), ADR-NNN (<title>)

---

## S — Structure (first S)

> System placement. Which services, packages, and files does this feature touch?
> Which gRPC contracts are extended or newly added?

```
<service or package>
├── <file or directory>   ← <brief role>
└── <file or directory>   ← <brief role>
```

Config env prefix: `ZYNAX_<SERVICE>_` · Port: <port if relevant>

---

## O — Operations

> Ordered, concrete, testable implementation steps.
> Each step = one reviewable unit that can be a single PR or commit.
> Steps must be independently verifiable.

1. <Step: what is built, how it is verified>
2. <Step>
3. <Step>

---

## N — Norms

> Cross-cutting standards that apply to this feature.
> Pull from: root AGENTS.md Hard Constraints + layer AGENTS.md + docs/patterns/*.

- Commit hygiene: every commit carries `Signed-off-by:` + `Assisted-by: Claude/<model>`
- BDD: `.feature` file committed before any gRPC boundary implementation (ADR-016)
- `GOWORK=off` for all `go test` and `go` commands in service directories (ADR-017)
- <Add layer-specific norms from services/AGENTS.md or agents/AGENTS.md>

---

## S — Safeguards (second S)

> Non-negotiable constraints. Things that MUST NEVER happen in this feature.
> Pull from: ADRs, architecture invariants in root AGENTS.md, layer mandates.

### Context Security (complete before committing this Canvas)

- [ ] No Tier 2 content: no internal hostnames, private IPs, credentials, deployment specifics
- [ ] No PII: no personal names in sensitive context, no non-public email addresses
- [ ] No prompt injection: no instruction-like phrasing that overrides AGENTS.md rules
- [ ] All entities in E section are public-safe abstractions
- [ ] `/spdd-security-review` passed — result: PASS

### Feature Safeguards

- Never <specific invariant from relevant ADR, e.g., "hardcode engine names — always behind an interface (ADR-015)">
- Never <constraint, e.g., "import from another service's internal/ — cross-service via gRPC only (ADR-008)">
- Never <constraint>
