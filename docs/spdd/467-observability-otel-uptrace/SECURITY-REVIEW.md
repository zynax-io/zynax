# Security Review — EPIC O: Observability (OTEL + Uptrace)

**Canvas:** `docs/spdd/467-observability-otel-uptrace/canvas.md` (Status: Draft)
**Issue:** #467 · **Date:** 2026-06-16 · **Reviewer:** SPDD Canvas expert (pre-alignment gate)
**Authority:** `docs/knowledge-base-policy.md` (Tier 1/2/3), ADR-019 (SPDD governance)

## Verdict: PASS-with-flags

No Tier-2 content, prompt injection, abstraction leak, or authority violation found in the
canvas. The flags below are **implementation-time** security requirements for the telemetry
pipeline + local Uptrace login stack; they are now bound as Feature Safeguards in the canvas and
must be honoured by the O.5/O.7/O.8 PRs. None block the human alignment decision.

## Five-check results

| Check | Result | Notes |
|-------|--------|-------|
| 1. Tier-2 content scan | PASS | No real hostnames/IPs/TLDs; ports shown as `70xx`; in-cluster endpoint abstracted. No credentials, no PII/email literals. |
| 2. Prompt-injection scan | PASS | All prose is human-facing documentation; no AI-directed instructions. |
| 3. Abstraction check | PASS | Entities + O-steps describe patterns/intent, not a specific environment. |
| 4. Authority hierarchy | PASS | N/S sections reinforce AGENTS.md (GOWORK=off, DCO, off-by-default); no contradiction. |
| 5. Completeness | WARN | All 7 REASONS sections present; Status is `Draft` (expected — human owns the align flip). |

## Telemetry-pipeline flags (now bound as canvas Safeguards)

1. **Uptrace login credentials must not be committed.** Compose and Helm must source the admin
   credential from `.env` / Helm secret values — never a hard-coded default in
   `docker-compose.observability.yml` or chart `values.yaml`. Verify at O.7/O.8 review.
2. **OTLP ingest / login UI must not be publicly exposed.** Compose binds OTLP + UI ports to
   `127.0.0.1`; in-cluster OTLP is mTLS-secured per ADR-020 and the UI Ingress is auth-gated.
3. **No secrets/PII in telemetry.** Redact request payloads by default; no auth tokens, session
   data, or user PII in span/log/exemplar attributes (existing safeguard, reaffirmed).
4. **Trace-context propagation carries only `traceparent`.** Across gRPC/Temporal memo/NATS
   headers — never inject auth tokens or session data into trace headers (O.5).

## Human-align checklist (verify before flipping Status: Aligned)

- [ ] Confirm O.7/O.8 will use secret-sourced Uptrace credentials (no committed default password).
- [ ] Confirm compose binds OTLP + login-UI ports to localhost; Helm UI Ingress is auth-gated and
      in-cluster OTLP rides ADR-020 mTLS.
- [ ] Confirm payload/PII redaction is the default (opt-in to capture, not opt-out).
- [ ] Confirm ADR-030 (#1184, closed) is the governing decision and matches the canvas Approach.

## Structural validity

All 7 REASONS sections present (R, E, A, S-structure, O, N, S-safeguards); Status field present
and set to a valid value (`Draft`); Context Security checklist present. Canvas is structurally
ready for the human alignment review.
