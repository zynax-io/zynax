---
name: Epic
about: A large, multi-PR feature that delivers measurable user/adoption value across several stories
labels: ["type: epic", "status: needs-triage"]
assignees: ''
---

<!--
An Epic is the unit of *value*, not of code. Before filing:
  1. Search existing epics — extend/merge rather than duplicate.
  2. Check ROADMAP.md + state/current-milestone.md — which milestone does this serve?
  3. feat: epics require a REASONS Canvas committed before any implementation (ADR-019).
Keep child stories small, independent, testable, and individually valuable.
-->

## The one question this epic answers

> _One sentence, in the reader's words — the job-to-be-done. A Zynax user (or operator,
> or adopter) should understand the point without any repository knowledge._
>
> e.g. "How does someone go from nothing to experiencing Zynax's value with one command?"

---

## Why it matters (product · business · adoption)

> Orient the work toward impact, not mechanics. Which levers does it move, and how?

- **For the Zynax user / adopter:** <the observable value they gain>
- **Adoption & business angle:** <which of these does this advance, and how — be concrete>
  - Individual users / "people interest"
  - Enterprises (security, scale, operability, trust)
  - CNCF interest / ecosystem credibility
  - Community / contributor pull
- **Cost of not doing it:** <what stays broken, who stays blocked, what adoption is lost>

---

## Target experience / outcome

> What does "great" look like once this lands? A short narrative or flow the user lives.

```
<flow: trigger → … → meaningful visible result → next action>
```

---

## Scope

**In scope**
-

**Out of scope** (→ which other epic/milestone owns it)
-

---

## Acceptance criteria (measurable)

> Observable outcomes that prove the value above is delivered. Avoid "it works".

- [ ]
- [ ]

---

## Child stories

> Small, independent, testable, individually valuable. Each links back here and uses the Feature
> template's **Story (INVEST)** block (as-a/I-want/so-that · size · canvas Operations-step · deps).

- [ ] #… — <story>
- [ ] #… — <story>

---

## Dependencies & sequencing

- Depends on: #…
- Blocks: #…
- Suggested order / critical path:

---

## Architecture, contracts & decisions

- Proto / gRPC boundary touched? → RFC + `buf breaking` + `.feature` (ADR-016)
- New one-way-door decision? → ADR (`docs/adr/`)
- Relevant ADRs / RFCs:

---

## SPDD Canvas (feat: epics)

- REASONS Canvas: `docs/spdd/<this-issue>-<slug>/canvas.md` — Status: Draft → Aligned before code (ADR-019)

---

## Documentation & human validation

- Docs to add/update (Diátaxis: tutorial / how-to / reference / explanation):
- Each user-visible child story ships a **human-validation guide** (see the validation standard).

---

## Traceability

> Nothing orphaned. Keep the chain linked both ways.

`Milestone → this Epic → Stories → Implementation PRs → Documentation → Human validation → Release notes → Future work`

- Milestone:
- Release notes line:
- Future work / follow-on epics:

---

## AI-planning metadata (optional, machine-readable)

```yaml
objective:          # the one-question, restated for agents
required_experts:   []   # e.g. go-services, infra-helm, spdd-canvas
optional_experts:   []
inputs:             []
outputs:            []
dependencies:       []   # issue/epic ids
risk:               # low | medium | high
confidence:         # low | medium | high
context_packs:      []   # docs/paths an agent must load
large_context_refs: []   # load lazily
```
