# REASONS Canvas — EPIC D: Documentation (quick-start · authoring · observability)

> Tier 1 (public-safe). `docs:` work is SPDD-exempt; committed for traceability.

**Issue:** #1173 · **Milestone:** M7 (v0.6.0)
**Author:** M7 program plan · **Date:** 2026-06-15 · **Status:** Draft

---

## R — Requirements
- **Problem:** there is no quick-start, no authoring guide, and no observability guide. A new
  developer cannot get from clone to a traced real-workflow run using docs alone.
- **Done when:** the quick-start takes a new developer clone → `docker compose up` (incl. Uptrace) →
  `apply` a real workflow → watch the trace/logs in the Uptrace UI, using only the docs.

## E — Entities
```
docs/quickstart · docs/developer-guide
docs/authoring (workflow + expert)
docs/context · docs/git-mcp
docs/observability (OpenTelemetry + Uptrace)
docs/examples-index · best-practices · faq · migration (v0.5→v0.6)
```

## A — Approach
**We will:** write the quick-start + developer guide, workflow/expert authoring guides, context +
Git MCP guides, observability (OTEL + Uptrace, incl. login UI for logs) guide, and an examples index +
FAQ + v0.5→v0.6 migration notes.
**We will NOT:** write the full example-catalog docs (K8s/Helm/GitOps/security catalog) — **M-dx**.
**Governing ADRs:** all M7 ADRs (028–033) are the source material.

## S — Structure (first S)
```
docs/quickstart.md · docs/developer-guide.md
docs/authoring/{workflows.md,experts.md}
docs/context/context-system.md · docs/git-mcp/git-mcp.md
docs/observability/{opentelemetry.md,uptrace.md,sampling.md,troubleshooting.md}
docs/examples/index.md · docs/best-practices.md · docs/faq.md · docs/migration-v0.6.md
```

## O — Operations (stories — `spdd-story` form)

**GitHub issues:** D.1 #1217 · D.2 #1218 · D.3 #1219 · D.4 #1220 · D.5 #1221 (epic #1173)
**D.1 — Quick Start + Developer Guide** · M · `docs`
- As a `new developer`, I want a clone→traced-run quick-start so I'm productive in minutes.
- AC: [ ] quick-start: compose up (incl. Uptrace) → apply real workflow → see trace+logs in UI; [ ] developer guide covering make targets. Deps: T.3, O.7.

**D.2 — Workflow + Expert authoring guides** · M · `docs`
- As an `author`, I want authoring guides so I can write workflows and experts correctly.
- AC: [ ] workflow authoring (data-flow, templates, versioning); [ ] expert authoring (runtime + Claude). Deps: T.1, X.2.

**D.3 — Context System + Git MCP guides** · S · `docs`
- As an `author`, I want context + Git MCP guides so handoffs and Git usage are safe and clear.
- AC: [ ] context model + handoff contract; [ ] Git MCP least-privilege setup. Deps: C.4, G.4.

**D.4 — Observability guides (OTEL + Uptrace)** · M · `docs`
- As an `operator`, I want OTEL + Uptrace guides so I can run, view (login UI), and tune telemetry.
- AC: [ ] OTEL setup; [ ] Uptrace local + Helm with login UI for logs/traces/APM; [ ] sampling + retention + troubleshooting. Deps: O.7, O.8, O.9.

**D.5 — Examples index + best practices + FAQ + migration** · S · `docs`
- As a `developer`, I want an index + FAQ + migration notes so I can find examples and upgrade cleanly.
- AC: [ ] examples index; [ ] best practices; [ ] FAQ; [ ] v0.5→v0.6 migration notes. Deps: T.3.

**Order:** {D.1, D.4} after their features land → {D.2, D.3, D.5}.

## N — Norms
- Docs are PR-size-exempt; keep examples copy-pasteable and verified against the running stack.
- `Signed-off-by:` + `Assisted-by:`; no literal personal emails in docs (gitleaks PII gate).

## S — Safeguards (second S)
### Context Security
- [ ] No Tier 2 content (placeholder hosts/tokens only); [ ] no PII / no literal emails; [ ] N/A non-feat

### Feature Safeguards
- Never document a command/flow that isn't verified against the running stack — docs must be true.
- Never include real credentials or internal hostnames in examples — placeholders only.
