# docs/spdd/ — REASONS Canvas Repository

> SPDD (Structured Prompt-Driven Development) canvas artifacts.
> One `canvas.md` per `feat:` GitHub issue. All content is Tier 1 (public-safe).
> See `docs/patterns/spdd-guide.md` for the full methodology and `CANVAS_TEMPLATE.md` for the template.
> Governed by ADR-019.

---

## Naming Convention

```
docs/spdd/<issue-number>-<kebab-case-slug>/
    canvas.md               ← public Canvas (always committed)
    canvas.private.md       ← Tier 2 companion (gitignored, NEVER committed)
```

Examples:
- `docs/spdd/205-spdd-methodology/canvas.md`
- `docs/spdd/214-temporal-execution/canvas.md`

The slug is the issue title kebab-cased, first 4–6 words only.

---

## Canvas Status Lifecycle

```
Draft → Aligned → Implemented → Synced
```

| Status | Meaning |
|--------|---------|
| `Draft` | Generated, not yet human-reviewed |
| `Aligned` | Human-reviewed; "what we will / won't do" confirmed; implementation may begin |
| `Implemented` | All Operations steps complete; feature merged |
| `Synced` | Canvas updated to reflect final implementation (post-refactor sync) |

**No code may be written for a `feat:` issue until its Canvas reaches `Aligned`.**

---

## Canvas Index

| Issue | Feature | Canvas | Status |
|-------|---------|--------|--------|
| — | — | — | — |

*Add entries as Canvases are created. Format: `[#NNN title](NNN-slug/canvas.md)`*

---

## Security Rules

Every Canvas committed here is **permanently public** (forks, mirrors, AI training sets).

Before committing any Canvas:
1. Run `/spdd-security-review docs/spdd/<issue>-<slug>/canvas.md`
2. Confirm all 7 REASONS sections contain only Tier 1 content
3. Move any Tier 2 content (hostnames, credentials, IPs, deployment details) to `canvas.private.md`

See `docs/knowledge-base-policy.md` for the full classification rules.
