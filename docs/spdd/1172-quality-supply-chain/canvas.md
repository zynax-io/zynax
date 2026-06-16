# REASONS Canvas — EPIC Q: Quality & Supply-Chain Fixes (audit closeout)

> Tier 1 (public-safe). `chore:`/`ci:`/`docs:` work is SPDD-exempt; committed for traceability.

**Issue:** #1172 + #582 #583 · **Milestone:** M7 (v0.6.0)
**Author:** M7 program plan · **Date:** 2026-06-15 · **Status:** Draft

---

## R — Requirements
- **Problem (from the 2026-06-15 reality check):** `make security-agents`/`lint-go` fail on a
  tools-image `pip` CVE; `make test-coverage` fails on an interface-only package; PyPI Trusted
  Publisher provenance is undocumented.
- **Done when:** `make ci` is green on a clean checkout; the PyPI Trusted Publisher history is
  recorded; the Go module consumption path is documented.

## E — Entities
```
tools image (pip 26.1.1 → 26.1.2, PYSEC-2026-196)
coverage gate (event-bus interface-only domain package)
PyPI Trusted Publisher (OIDC) provenance record
pkg.go.dev module consumption path
ManifestWorkflowID collision-domain ADR
```

## A — Approach
**We will:** bump tools-image `pip`; make the coverage gate honest for zero-statement packages;
document the PyPI Trusted Publisher history; verify pkg.go.dev import; record the ManifestWorkflowID ADR.
**We will NOT:** restructure the coverage gate beyond zero-statement handling (no blanket skips).
**Governing ADRs:** ADR-024 (image SoT), ADR-025 (SLSA provenance), ADR-027 (shift-left pipeline).

## S — Structure (first S)
```
infra/docker/Dockerfile.tools (+ images.yaml)   ← pip bump
Makefile (test-coverage gate)                     ← zero-statement package handling
docs/milestones/M7-planning.md §14                ← PyPI Trusted Publisher history
docs/ (module consumption) · docs/adr/ADR-034     ← #582 / #583
```

## O — Operations (stories — `spdd-story` form)

**GitHub issues:** Q.1 #1212 · Q.2 #1213 · Q.3 #1214 · Q.4 #1215 · Q.5 #1216 (epic #1172)
**Q.1 — Bump tools-image pip → 26.1.2** · XS · `ci`
- As a `maintainer`, I want the pip CVE closed so `make security-agents`/`lint-go` run clean.
- AC: [ ] tools image rebuilt with pip 26.1.2; [ ] `make security-agents` passes; [ ] images.yaml updated via `make sync-images`. Deps: none. **(Wave 0 — unblocks green CI for all EPICs.)**

**Q.2 — Honest coverage gate for interface-only packages** · S · `ci`
- As a `maintainer`, I want the gate to handle zero-statement packages so `make test-coverage` passes truthfully.
- AC: [ ] `event-bus/internal/domain` no longer reported as `0.0% < 90%`; [ ] only zero-statement packages excluded (verified via `go tool cover` count, not a blanket skip). Deps: none. **(Wave 0.)**

**Q.3 — Document PyPI Trusted Publisher history** · S · `docs`
- As a `maintainer`, I want the OIDC publisher config recorded so SDK provenance is auditable.
- AC: [ ] §14 of the M7 plan filled with publisher/workflow/environment/first-publish; [ ] linked from v0.6.0 release notes; [ ] if the PyPI entry is missing, it is created before next publish. Deps: none.

**Q.4 — Document Go module consumption path (#582)** · XS · `docs`
- As a `consumer`, I want a verified pkg.go.dev import path so the module is usable downstream.
- AC: [ ] import availability verified + documented. Deps: none.

**Q.5 — ManifestWorkflowID collision-domain ADR (#583)** · S · `adr-proposal`
- As a `maintainer`, I want the 64-bit collision domain + canonicalization recorded so the id scheme is stable.
- AC: [ ] ADR-034 committed. Deps: none.

**Order:** {Q.1, Q.2} first (Wave 0) → {Q.3, Q.4, Q.5} any time.

## N — Norms
- Image refs only via `images.yaml` + `make sync-images` (ADR-024) — never hand-edit banner regions.
- `Signed-off-by:` + `Assisted-by:`; squash-merge required; no `[skip ci]` token.

## S — Safeguards (second S)
### Context Security
- [ ] No Tier 2 content (no real tokens/registry creds); [ ] no PII / no literal emails; [ ] N/A non-feat

### Feature Safeguards
- Never exclude a package with executable statements from the coverage gate — only true zero-statement packages.
- Never store a long-lived PyPI token — Trusted Publisher (OIDC) only.
- Never hand-edit image-version banner regions — use `make sync-images`.
