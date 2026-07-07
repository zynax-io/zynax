<!-- SPDX-License-Identifier: Apache-2.0 -->
# ADR-050: Fuzz-testing strategy for untrusted-input surfaces (retroactive)

**Status:** Proposed  **Date:** 2026-07-08
**Related:** ADR-016 (**extends** — layered testing strategy gains a fuzz tier for parse surfaces), ADR-017 (GOWORK=off applies to fuzz runs)
**Proposal issue:** #1695 · **Inventory completion:** #1417, #1659

---

## Context

Zynax's control plane parses untrusted or semi-trusted input at several boundaries: YAML
manifests (workflow-compiler), CEL guard expressions (engine-adapter), protobuf payloads
at every gRPC boundary, and event-subject globs (`libs/zynaxevents.MatchesGlob`). A
malformed input at any of these must produce a clean validation error — never a panic in
a control-plane service.

Fuzz coverage exists but is ungoverned: `FuzzParseManifest` and `FuzzEvalGuard` shipped
as code without a decision record — the 2026-06-19 architecture review flags exactly this
(R8/T2.4: "closed by code, not decision; file the ADR retroactively") and recommends
extending fuzzing to proto unmarshalling. Two open issues (#1417, #1659) ask for more
fuzz targets with no policy defining which surfaces are mandatory, where corpora live, or
what fuzzing costs in CI. ADR-016 governs test *placement* and coverage gates but is
silent on fuzzing.

---

## Decision

We will govern fuzzing with a **declared inventory of untrusted-input surfaces** that
MUST carry Go native fuzz targets:

1. **Initial inventory:** YAML manifest parsing (workflow-compiler), CEL guard evaluation
   (engine-adapter), protobuf unmarshalling at gRPC boundaries, event-subject glob
   matching (`libs/zynaxevents`). The inventory lives with the testing docs and is the
   routing target for #1417 and #1659.
2. **Corpus convention:** seed corpora are committed under each module's
   `testdata/fuzz/<FuzzTarget>/` (the Go toolchain's native layout); every crash found
   becomes a committed regression seed plus, where practical, a permanent unit test.
3. **CI budget:** PR CI runs fuzz targets in deterministic seed-replay mode (corpus
   verification, effectively free); bounded exploratory runs (`-fuzztime` on the order of
   seconds per target) ride the existing test lane. Long-running exploration is a
   maintainer action, not a PR gate.
4. **Growth rule:** a PR that introduces a new parse/decode surface for untrusted input
   adds the corresponding fuzz target *in the same PR* — enforced in review, mirroring
   the ADR-016 feature-before-implementation discipline.

---

## Rationale

| Option | Assessment |
|--------|------------|
| Status quo — fuzz targets appear organically | ✗ Rejected — coverage is accidental; the review already found the proto-unmarshal hole; no corpus or budget convention to review against. |
| OSS-Fuzz onboarding now | ✗ Deferred — deepest coverage and free compute, but onboarding/triage overhead is disproportionate pre-v1.0.0; an in-repo policy is prerequisite anyway; natural at Incubation maturity. |
| **Declared inventory + native Go fuzz, committed corpora, bounded CI budget, same-PR growth rule** | ✅ Chosen — stdlib tooling already in use; makes existing targets policy-backed; deterministic CI cost; gives #1417/#1659 a Definition-of-Done. |

---

## Consequences

- **Positive:** the two shipped targets become policy-backed; #1417/#1659 become
  inventory-completion work with clear acceptance; new parsers can't ship unfuzzed
  silently; CNCF security-questionnaire answer exists.
- **Negative / trade-off:** bounded PR-CI fuzz finds shallow bugs only (mitigated by
  crash-seed promotion and optional long runs); adding a parse surface now carries a
  same-PR fuzz obligation.
- **Neutral / follow-up required:** document the inventory in the testing docs; complete
  it via #1417 (CEL/YAML/proto) and #1659 (MatchesGlob); wire seed-replay verification
  into the test lane if not already implicit; conformance-suite manifests (#1692) become
  natural seed-corpus donors; revisit OSS-Fuzz post-CNCF-acceptance.
