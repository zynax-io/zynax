# Private Canvas Context — <Feature Title>

> **CLASSIFICATION: Tier 2 — NEVER commit to the public repository.**
> This file is gitignored (`docs/spdd/**/canvas.private.md`).
> Distribute to collaborators via a private channel (private GitHub repo, encrypted file, or secure messaging).
>
> Public Canvas: `docs/spdd/<issue>-<slug>/canvas.md`

**Issue:** #<number>
**Author:** <name>
**Date:** YYYY-MM-DD

---

## Private Deployment Context

> Real hostnames, cluster names, namespace names, internal service addresses.
> Use this section for anything you cannot abstract in the public Canvas.

---

## Private Service Dependencies

> Internal service names, real endpoint addresses, port numbers, credential references.
> Do NOT store actual credentials here — reference the secret store location only.

---

## Security-Sensitive Design Notes

> WAF rules, rate-limit thresholds, security assumptions, vulnerability details.
> Anything an attacker could use if this file were leaked.

---

## Customer / Tenant-Specific Constraints

> Business rules specific to a customer or tenant that cannot be made generic.

---

## Private Incident Context

> Ongoing incidents, internal postmortems, or debugging context that informed this feature.

---

## Out-of-Band Distribution

Share this file using one of:
- **Option A (recommended):** Private GitHub repo `zynax-io/zynax-private-context` — same `docs/spdd/` structure, matching issue numbers
- **Option B:** Encrypt with `age` or `gpg` using the team's public key; decrypt on recipient machine
- **Option C:** Paste relevant sections into the session at start time (local only, never committed)

---

## Injection Instructions

When starting a Claude Code session with private context:
```bash
# Copy relevant sections from this file into the session prompt at start
cat docs/spdd/<issue>/canvas.private.md
# Then paste the relevant sections when starting your session
```

**Never use any /spdd-* command to commit this content.**
