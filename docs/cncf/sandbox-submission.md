<!-- SPDX-License-Identifier: Apache-2.0 -->

# CNCF Sandbox submission — prepared application (M8.B)

> **Status: PREPARED, NOT FILED.** Filing is an external action only the
> maintainer can take (the application is submitted under the maintainer's
> identity and carries ongoing obligations). Everything below is ready to
> copy into the official form. Nothing in this document claims the
> application has been submitted.

## How to file (maintainer actions)

1. **Sandbox application** — CNCF Sandbox applications are filed as issues on
   [github.com/cncf/sandbox](https://github.com/cncf/sandbox) using the
   project-application form. Copy the answers below. Applications are
   reviewed by the TOC in batches (check the repo's README for the current
   review schedule).
2. **CNCF Landscape entry** — a PR to
   [github.com/cncf/landscape](https://github.com/cncf/landscape) adding the
   entry in [§Landscape entry](#landscape-entry) below (category:
   Orchestration & Management → Scheduling & Orchestration). A landscape
   entry does not require Sandbox acceptance and can be filed first.
3. After filing, record the application issue link on epic #471 and in this
   file's status line.

## Application answers (copy into the cncf/sandbox form)

**Project name:** Zynax

**Project repository:** https://github.com/zynax-io/zynax

**License:** Apache-2.0 (SPDX headers repo-wide; no proprietary components)

**One-line description:** Write your agent workflow once — run it on Temporal
or Argo without a rewrite: an engine-portable declarative YAML layer for
agentic automation.

**Longer description:**
Zynax is the engine-portability layer for AI agent workflows. Users author a
declarative YAML state machine once; Zynax compiles it to an engine-agnostic
intermediate representation and dispatches it to interchangeable workflow
engines (Temporal and Argo Workflows today) behind a single interface —
portability is proved by a dual-engine e2e matrix on every change, not
claimed. The platform is deliberately thin and Kubernetes-native: agents and
workflows are CRDs reconciled by controllers, edge auth and rate-limiting are
delegated to the Gateway API (Envoy Gateway), the namespace engine allow-list
is a CEL `ValidatingAdmissionPolicy`, eventing is CloudEvents over NATS
JetStream, and internal transport is mTLS with cert-manager identities.
Custom code is reserved for the genuinely differentiating core: manifest
compilation, engine-fit routing, and capability scheduling.

**Why CNCF / alignment:**
- Consumes and composes CNCF/ecosystem projects rather than wrapping them:
  Kubernetes CRDs + controller-runtime, Envoy Gateway (Gateway API),
  cert-manager, NATS JetStream, CloudEvents, Argo Workflows, Helm,
  Prometheus, OpenTelemetry.
- The project's governing architectural principle (ADR-040, "thin-Zynax") is
  to delegate every generic concern to a Kubernetes-native primitive — the
  M8 milestone retired bespoke services in favour of CRDs, admission policy,
  Gateway API, and direct JetStream.
- Vendor-neutral governance (GOVERNANCE.md), DCO on every commit, OpenSSF
  Scorecard badge, signed commits, SBOM/supply-chain hardening in CI.

**Maturity / current state (honest):**
- Single-maintainer project (MAINTAINERS.md) actively building community:
  curated `good first issue` programme, troubleshooting guide, contributor
  docs. Sandbox is sought precisely to grow a contributor base;
  ≥2-maintainer diversity is understood to be an Incubation requirement.
- v0.7.x: local-Kubernetes (kind) first-run in one command with zero
  secrets; dual-engine e2e in CI; Postgres-backed control-plane state; mTLS
  mesh; Helm charts; Python SDK on PyPI.
- No known production adopters yet — adopter discovery is a Sandbox-phase
  goal. (Do not overstate this in the form.)

**Comparison with similar projects:**
- **Kagent / K8s-native agent frameworks:** couple agent workflows to one
  runtime; Zynax's wedge is engine portability of the workflow definition.
- **Temporal / Argo Workflows themselves:** engines, not portability layers —
  Zynax targets the authoring/compilation layer above them and treats them
  as interchangeable backends.
- **LangGraph and Python agent frameworks:** in-process orchestration
  libraries; Zynax is a control plane with a declarative contract, engine
  execution, and Kubernetes-native operations.

**TOC sponsor:** none yet — to be sought during review (standard for
Sandbox).

## Landscape entry

```yaml
- item:
    name: Zynax
    homepage_url: https://github.com/zynax-io/zynax
    repo_url: https://github.com/zynax-io/zynax
    logo: zynax.svg   # SVG logo required by landscape rules — export before filing
    crunchbase: ""    # omit if none
    description: Engine-portable declarative YAML layer for AI agent workflows — write once, run on Temporal or Argo.
```

> The landscape requires an SVG logo committed to their `hosted_logos/`;
> export the project logo as SVG before opening that PR.

## Submission checklist

- [x] Apache-2.0 license, SPDX headers
- [x] GOVERNANCE.md + MAINTAINERS.md (single-maintainer mode stated honestly)
- [x] CODE_OF_CONDUCT.md, CONTRIBUTING.md, SECURITY.md
- [x] DCO enforced in CI; signed commits; conventional commits
- [x] Public roadmap (ROADMAP.md) with Fork A positioning
- [x] Troubleshooting guide + good-first-issue programme
- [x] Dual-engine e2e (the portability proof) green in CI
- [ ] SVG logo exported for the landscape PR
- [ ] Sandbox application issue filed (maintainer)
- [ ] Landscape PR filed (maintainer)
