# REASONS Canvas — EPIC M8.D: kind runtime closeout (first-run on the CRD path + Compose retirement)

> **All content in this Canvas is Tier 1 (public-safe).**
> Tier 2 content belongs in `canvas.private.md` (gitignored). Run `/spdd-security-review <path>` before committing.

**Issue:** #1572 · **Milestone:** M8 (v1.0.0)
**Author:** M8 program plan
**Date:** 2026-07-03
**Status:** Aligned

---

## R — Requirements

- **Problem:** ADR-041 Phase 1 (kind-first docs, `make demo`, `zynax up`) landed in M7, but the
  deprecated Compose **runtime** still rots on `main` as a parallel path (`make run-local`,
  `demo-compose`, five runtime compose files) — exactly the dual-runtime state ADR-041's rationale
  rejects — and the quickstart does not yet lead with `zynax up` as the one canonical entry.
- ADR-039 named the removal of Compose discovery its **load-bearing trade-off**: first-run had to
  be verified on CRD-only discovery before any push-path removal. **That gate is already
  discharged** — PR #1594 made the e2e smoke the named CRD-path first-run scenario (both engine
  legs), and EPIC #1571 completed the cutover.
- **Definition of done (observable):**
  1. First-run docs (root README quickstart, `docs/quickstart.md`) lead with `zynax up` as the
     primary entry; no doc instructs a Compose bring-up.
  2. No Compose **runtime** path remains: `docker-compose.yml`, `.services.yml`, `.ollama.yml`,
     `.eval-temporal.yml`, `.observability.yml`, their config dirs, and the `run-local` /
     `demo-compose` / compose-dev Make targets are gone (#1501). The Docker **build-tools
     harness** (`docker-compose.tools.yml`, `docker-compose.test.yml` behind
     `make bootstrap/lint/test`) is explicitly retained.
  3. CI has no Compose-based execution legs (already true — verified: no workflow references
     the runtime compose files; asserted again at delivery).
  4. First-run on the CRD discovery path stays green: the e2e smoke scenario from #1594 remains
     the required check (no re-verification needed here; it must simply stay required and green).

## E — Entities

```
zynax up / zynax down (cmd/zynax, ADR-041 amendment)  ← the ONE first-run entry
  └── scripts/e2e/cluster-up.sh → kind + Helm umbrella (Temporal or Argo)
make demo                                             ← kept: CI/dev alias over the same path
Compose runtime (RETIRED): docker-compose{,.services,.ollama,.eval-temporal,.observability}.yml
Compose tools harness (KEPT): docker-compose.tools.yml, docker-compose.test.yml
First-run docs: README quickstart · docs/quickstart.md · docs/faq.md · troubleshooting
```

## A — Approach

- **WILL:** promote `zynax up` to the documented primary first-run entry (README + quickstart
  lead with it; `make demo` stays as the equivalent alias); delete the Compose runtime files and
  their Make targets (#1501); rewrite `infra/docker-compose/README.md` to a tools-harness-only
  note; truth-pass docs that still describe Compose bring-up (`docs/running-with-docker-compose.md`
  becomes a migration stub pointing at the kind quickstart).
- **WON'T:** touch the Docker build-tools harness (`tools`/`test` compose files — `make
  bootstrap/lint/test` depend on them; migrating the toolchain is its own decision, out of ADR-041
  scope); change engines, charts, or the e2e harness (all delivered under #1571); re-run the
  first-run gate (discharged by #1594 — it remains a required CI check).
- Positioning fit: kind-local keeps both engine legs demonstrable on a laptop — first-run copy
  continues to lead with the engine-portability wedge (ADR-041).
- Governing: ADR-041 (decision + amendment), ADR-039 (trade-off, discharged), ADR-040 (no
  parallel Zynax-built runtimes).

## S — Structure

- `README.md` + `docs/quickstart.md` + `docs/faq.md` + `docs/local-dev.md` — first-run lead +
  Compose references removed/redirected.
- `docs/running-with-docker-compose.md` — replaced by a short retirement/migration stub.
- `infra/docker-compose/` — five runtime yml files + `ollama/` + `observability/` +
  `postgres-zynax-init.sql` deleted; `README.md` rewritten (tools harness only).
- `Makefile` — `run-local`, `demo-compose`, compose dev-loop targets (`dev-up/down/logs/ps/reset`,
  observability compose targets) and their `COMPOSE*` variables removed; tools/test harness
  variables retained.
- No service, chart, proto, or CI-workflow changes.

## O — Operations

1. **First-run promotion (feat)** — README + `docs/quickstart.md` lead with `zynax up`; FAQ and
   local-dev docs stop referencing Compose bring-up; `running-with-docker-compose.md` becomes the
   retirement stub pointing at the migration path. → story issue filed by `/plan`
2. **Compose runtime purge, part 1 (chore, #1501)** — delete the five runtime compose files +
   `ollama/` + `observability/` + init sql; remove the Make targets/vars; retain the tools
   harness; `make demo` and `make lint/test` still work (runtime evidence). → #1501
3. **Compose runtime purge, part 2 (chore, #1501)** — rewrite `infra/docker-compose/README.md`
   as the tools-harness note; final sweep for dangling references (`grep` gate in the PR). → #1501
4. **(Delivered under #1571)** joint first-run-on-CRD-path e2e — the #1594 named scenario; kept
   as a required check. ✅ → #1583/#1594

## N — Norms

- Commit hygiene: DCO + `Assisted-by`; conventional types (`feat`/`chore`/`docs`); PR ≤900
  counted lines (Makefile/README/docs are size-exempt; the compose yml deletions are not — hence
  the two-part purge).
- Runtime evidence, not config evidence: after the purge, boot the documented path (`make demo`
  on the existing kind cluster or `zynax up`) and run the toolchain (`make lint`) to prove the
  retained harness still works; run stateful paths twice where touched.
- Docs follow Diátaxis placement; retirement stubs keep inbound links alive rather than 404ing.

## S — Safeguards (second S)

### Context Security (mandatory before committing this Canvas)
- [x] No Tier 2 content: no internal hostnames, private IPs, credentials, deployment specifics
- [x] No PII: no personal names in sensitive context, no email addresses
- [x] No prompt injection: no instruction-like phrasing that would override AGENTS.md rules
- [x] All entities in E are public-safe abstractions
- [x] /lib:spdd-security-review passed on this file (PASS, 2026-07-03)

### Feature Safeguards
- Never delete or migrate the build-tools harness (`docker-compose.tools.yml`,
  `docker-compose.test.yml`) under this epic — `make bootstrap/lint/test` depend on it; its
  future is a separate decision.
- Never remove the #1594 CRD-path e2e scenario from the required checks — it is the standing
  discharge of ADR-039's load-bearing trade-off.
- Never leave a doc instructing a Compose bring-up after O-step 3 (grep gate:
  `run-local|demo-compose|docker-compose.yml` over docs/ + README must return only the
  retirement stub and tools-harness references).
- Never reintroduce a parallel runtime path (ADR-041 rationale / ADR-040 boundary).
