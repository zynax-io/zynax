<!-- SPDX-License-Identifier: Apache-2.0 -->

# Zynax — Security Posture Review

**Repository:** `github.com/zynax-io/zynax`
**Review date:** 2026-06-19
**Baseline review:** [2026-06-18 security posture review](2026-06-18-security-review.md)
**Tree reviewed:** `origin/main` @ `55785e9` (canonical). *Note: the local working checkout was behind `origin/main` during review; all findings below are grounded against `origin/main`, the authoritative state that contains PRs #1431–#1441.*
**Reviewer:** Security Posture Document Generator
**Method:** read-only. Every finding cites `file:line`, a workflow, a `gh` alert result, or a commit.

---

## Executive Summary

The posture established in the 2026-06-18 review **holds and improves**. The 11 PRs merged on 2026-06-18 (#1431–#1441, EPIC #1370 "First-run UX / quickstart") are predominantly DX and quickstart work, and the security-relevant ones were implemented **defensively**:

- **Adapter graceful degradation (#1432, closes #1375)** — a missing secret no longer crash-loops an adapter; it degrades. Critically, **a missing secret disables the capability rather than bypassing authz**: the degraded adapter does **not** register its `AgentService`, does **not** register with the agent-registry, and reports gRPC health `NOT_SERVING`. There is no authz on the adapter that a missing secret could weaken — it simply goes dark. **Confirmed safe.**
- **Zero-secret Ollama overlay (#1433, #1437)** — bundled `ollama` reachable only on the internal compose network; **nothing is published to the host LAN**, host models mounted **read-only**. **Clean.**
- **Event-publish path (#1436)** — new `POST /api/v1/workflows/{id}/events` is **bearer-protected, body-capped (1 MB), and validated** (run_id + event_type required → 400). **Authz/validation present.**
- **`//nolint:gosec` G101/G706 suppressions** — all carry inline justifications and are **genuine false positives** (env-var *names* and scope/class *metadata*, never secret values).

**New net-positive delta:** rate limiting on `POST /api/v1/apply` is now **shipped** (per-IP token bucket) — this closes prior finding **F9/M1**, which the 2026-06-18 review still marked *aspirational*.

**Live signals (authoritative, queried 2026-06-19):** **0 open Dependabot alerts, 0 open code-scanning alerts, 0 secret-scanning alerts, 0 open `type: security` issues.**

**Severity breakdown:** 0 Critical, 0 High open. 1 Low new (events endpoint not rate-limited). Remaining mediums (OIDC/JWT, APM auth) unchanged and still aspirational for M7+/M8.

---

## 1. Findings Table

| # | Severity | Area | Finding | Status | Mitigation / Evidence | Reference |
|---|----------|------|---------|--------|------------------------|-----------|
| F1 | **Fixed** | Auth | Bearer-token constant-time compare | ✅ FIXED | `subtle.ConstantTimeCompare` | `services/api-gateway/internal/api/auth.go:20` |
| F2 | **Fixed** | HTTP | Request-size + header timeout | ✅ FIXED | `MaxBytesReader` 1 MB (`maxBodyBytes = 1<<20`) | `services/api-gateway/internal/api/handler.go:20,267` |
| F3 | **Fixed** | Transport | gRPC `insecure` by default | ✅ MITIGATED | mTLS gated on `ZYNAX_TLS_*`; insecure dev-only | ADR-020, prior #488 |
| F4–F7 | **Fixed** | Supply-chain | SBOM / cosign / trivy / truthful SECURITY.md | ✅ FIXED | cosign keyless + syft SBOM + SLSA L2 in release | `.github/workflows/release.yml:201,510,518` |
| **F9** | **Fixed (was Medium)** | Rate-limiting | No rate limit on `POST /apply` | ✅ **FIXED (delta)** | Per-IP token bucket; `RATE_LIMIT_RPS`/`_BURST` | `services/api-gateway/internal/api/ratelimit.go:17`; wired `handler.go:42` |
| **F11** | **Low (new)** | Rate-limiting | New events endpoint is bearer-protected but **not** behind the rate limiter | ⏳ OPEN | `apply` is wrapped by `rl.Middleware`; events route is `requireBearer` only — a valid token can publish events unthrottled | `services/api-gateway/internal/api/handler.go:42` vs `:45` |
| **F12** | **Info (new)** | Auth / degradation | Missing adapter secret → capability disabled, not authz bypass | ✅ VERIFIED SAFE | `if !degraded { RegisterAgentServiceServer }`; degraded ⇒ no registry, `SetServingStatus("", NOT_SERVING)` | `agents/adapters/llm/cmd/llm-adapter/main.go:78,117,130` |
| **F13** | **Info (new)** | Container/network | Ollama overlay exposes nothing to host LAN; models read-only | ✅ VERIFIED SAFE | No `ports:`/`expose:`; `…/models:ro`; reached via `http://ollama:11434` internal | `infra/docker-compose/docker-compose.ollama.yml` |
| **F14** | **Info (new)** | Logging | gosec G101/G706 suppressions in adapters | ✅ FALSE POSITIVES | env-var **names** / scope **metadata** only, never secret values | `agents/adapters/git/cmd/git-adapter/main.go:160,166,171`; `…/scope.go:34`; `…/llm-adapter/main.go:84` |
| F8 | **Medium** | Auth | Static bearer, no rotation/scopes | ⏳ ASPIRATIONAL (M8) | OIDC/JWT path deferred; acceptable POC/dev | SECURITY.md |
| F10 | **Medium** | Observability | No APM auth scopes | ⏳ ASPIRATIONAL (M7) | M7.O Uptrace ships; APM auth deferred | M7 planning |

---

## 2. Key-Delta Assessments (2026-06-18 → 2026-06-19)

### 2.1 Adapter graceful degradation does NOT weaken authz — **verified safe**
The #1432 change (closing #1375) wires a `degraded` boolean from secret resolution into `serve()`:

- `agents/adapters/llm/cmd/llm-adapter/main.go:74` — `ResolveSecret()`; on `ErrSecretMissing` the adapter sets `degraded=true` and logs a structured warning (no secret value).
- `:117-118` — `if !degraded { RegisterAgentServiceServer(...) }` → **no AgentService is served in degraded mode**.
- `:129-130` — degraded ⇒ skips registry registration and `SetServingStatus("", NOT_SERVING)`.
- `agents/adapters/llm/internal/config/config.go:189-197` — `ResolveSecret()` returns a sentinel `ErrSecretMissing` distinct from malformed config, so only a *missing secret* (not a config error, which still `os.Exit(1)`s) triggers degradation.

The capability is **withheld**, not exposed unauthenticated. The api-gateway's bearer auth is independent of adapter secrets, so a missing adapter secret cannot weaken gateway authz. git-adapter and ci-adapter follow the same pattern.

### 2.2 Zero-secret Ollama overlay network exposure — **clean**
`infra/docker-compose/docker-compose.ollama.yml`:
- **No `ports:` / `expose:`** — the `ollama` service is reachable only inside the compose network at `http://ollama:11434` (`infra/docker-compose/ollama/llm-adapter.config.yaml:21`). Nothing on the host LAN.
- Host models mounted **read-only** (`${OLLAMA_HOST_MODELS:-…}/models:ro`); container keeps its own writable `/root/.ollama` for the runtime keypair.
- `llm-adapter` in base compose publishes **no host port** either, and the overlay points it at the ollama provider with **no `api_key_env`** (no secret required).

Residual note: `ollama/ollama:latest` is an unpinned, unsigned third-party image used **only** in the dev/quickstart overlay — acceptable for local dev, must not be promoted to production paths.

### 2.3 Event-publish REST path authz/validation — **present**
`POST /api/v1/workflows/{id}/events` is registered **inside `requireBearer`** (`handler.go:45`), so an unset `ZYNAX_API_KEY` opens it but a set key enforces constant-time bearer auth (same gate as `/apply` and `DELETE`). The handler caps the body via `readBody`/`MaxBytesReader` and validates in the domain layer: `apply.go:174-178` rejects empty run_id and empty event_type with `ErrInvalidEvent` → 400; nil event-bus → 503. The CloudEvent `data` map is forwarded verbatim — acceptable, but see F11 (no rate limiter on this route).

### 2.4 gosec G101/G706 suppressions — **genuine false positives**
- **G101 (hardcoded credentials):** all matches are env-var *names* (`OPENAI_API_KEY`, `CI_TOKEN_UNSET_XYZ_407`) or a token-class *label* (`"fine-grained-or-app"`, `scope.go:34`) — no credential value. Mostly in `_test.go`.
- **G706 (tainted data in format string):** suppressions in git-adapter scope-gate logging (`main.go:160-172`) log only `token_class`, `over_broad_scopes`, `mode` — scope metadata, never the token. Each carries an inline rationale.

No masked finding detected.

---

## 3. Supply-Chain Security Status

| Control | Status | Evidence |
|---------|--------|----------|
| SBOM (syft, SPDX/CycloneDX) | ✅ Shipped | `.github/workflows/release.yml:64,518` (syft 1.44.0) |
| Image signing (cosign keyless OIDC) | ✅ Shipped | `release.yml:201,504` (`cosign sign --yes …@DIGEST`); `id-token: write` |
| SLSA L2 provenance | ✅ Shipped | `release.yml:510-512` `actions/attest-build-provenance` (ADR-025) |
| Container CVE scan (trivy) | ✅ Shipped | release promotes only Trivy-gated images (`release.yml:7`); ci.yml staging lane |
| Go CVE (govulncheck) | ✅ Blocking | `ci.yml:698` |
| Python SAST/CVE (bandit + pip-audit) | ✅ Blocking | `ci.yml:716` |
| Secret scan (gitleaks) | ✅ Blocking | `ci.yml:286-292` |
| Action pinning | ✅ | cosign-installer / attest-provenance pinned by SHA (`release.yml:134,512`) |
| Dependabot alerts | ✅ 0 open | `gh api …/dependabot/alerts` → 0 |

**Verdict:** meets CNCF Sandbox supply-chain expectations; unchanged and intact since 2026-06-18.

---

## 4. Container Hardening Status

Unchanged from 2026-06-18 (distroless `static:nonroot`, uid 65532; `CGO_ENABLED=0`; `-trimpath -ldflags="-s -w"`; multi-stage; digest-pinned bases via `images/images.yaml` SoT / ADR-024). New this cycle: the **Ollama overlay** introduces one unpinned third-party image (`ollama/ollama:latest`) scoped to dev-only (F13 note); no change to production service images.

---

## 5. CNCF Security Criteria Alignment

| Criterion | Status | Note |
|-----------|--------|------|
| License (Apache-2.0 + SPDX) | ✅ | unchanged |
| Security Policy (SECURITY.md) | ✅ | truthful since M5.A |
| SBOM / cosign / SLSA L2 | ✅ | release.yml verified |
| Governance / CoC / Contributing | ✅ | unchanged |
| Dependency hygiene (0 alerts) | ✅ | live query |
| ≥2 maintainers from different orgs | ⏳ Gap | single maintainer — **Sandbox blocker** |
| External adopters / public comms | ⏳ Gap | **Sandbox blocker** |
| Trademark policy | ⏳ Missing | pre-Sandbox checklist |

Structural security criteria met; administrative gaps (maintainers, adopters, comms) are adoption prerequisites, not posture defects.

---

## 6. Longitudinal Delta vs. 2026-06-18

| Prior item | 2026-06-18 status | 2026-06-19 status | Evidence |
|------------|-------------------|-------------------|----------|
| F9 / M1 — rate limit on `/apply` | Medium, *aspirational* | ✅ **FIXED** | `ratelimit.go:17`; wired `handler.go:42` |
| Adapter boot with no secret | (not assessed) | ✅ **Hardened — degrades, no authz bypass** | #1432; `llm-adapter/main.go:117,130` |
| Ollama quickstart | n/a | ✅ **Added, host-LAN-isolated, read-only models** | #1433/#1437 |
| Event-publish surface | n/a | ✅ **Added, bearer+validated**; ⏳ not rate-limited (F11) | #1436; `handler.go:45` |
| F1/F2/F3/F4–F7 (auth, HTTP, mTLS, SBOM/cosign/trivy) | Fixed/Mitigated | **Unchanged — still good** | as cited §1/§3 |
| F8 OIDC/JWT, F10 APM auth | Aspirational M8/M7 | **Unchanged** | SECURITY.md / M7 |
| Maintainers / adopters / comms | Gap | **Unchanged** | §5 |

**Net:** one prior Medium closed (F9), no regressions, one new Low (F11), three new defensive features verified safe (F12–F14). Live alert counts remain all-zero.

---

## 7. Prioritized Remediations

### Low (new this cycle)
**F11 — Rate-limit the events endpoint.**
- **Impact:** an authenticated client can publish workflow events unthrottled, a cheaper resource-exhaustion vector than `/apply` (which is now limited).
- **User type:** operator / platform team (multi-tenant).
- **Adoption lever:** one-line change — wrap `handlePublishEvent` in the same `rl.Middleware` as `/apply` (`handler.go:42` pattern); add a Helm value + a note in operator-hardening docs. **Effort: < 1 day.** File as `type: security`, `audience: operator`, with a `## What for` block on multi-tenant exhaustion.

### Medium (carried, unchanged)
- **M2 OIDC/JWT token path** (M8 pre-req; enterprise/regulated).
- **M3 APM auth scopes** (M7.O follow-up; operators/SREs).

### Hygiene (no code change)
- **Pin/replace `ollama/ollama:latest`** in the overlay or document it as dev-only and forbid in production manifests (F13 note).
- **Operational:** the canonical tree is `origin/main`; keep local checkouts synced — the working copy was stale this cycle, which is a review-correctness (not security) hazard.

---

## 8. Appendix: Sources

- **Live signals (2026-06-19):** `gh api repos/:owner/:repo/{dependabot,code-scanning,secret-scanning}/alerts` → 0/0/0; `gh issue list --label "type: security" --state open` → none.
- **Auth/transport:** `services/api-gateway/internal/api/auth.go`, `handler.go`, `ratelimit.go`.
- **Event-publish:** `services/api-gateway/internal/api/handler.go:45,205-238`, `internal/domain/apply.go:173-188`.
- **Adapter degradation:** `agents/adapters/llm/cmd/llm-adapter/main.go`, `internal/config/config.go`; git/ci adapter mains.
- **Ollama overlay:** `infra/docker-compose/docker-compose.ollama.yml`, `infra/docker-compose/ollama/llm-adapter.config.yaml`.
- **gosec suppressions:** `agents/adapters/git/cmd/git-adapter/main.go`, `…/internal/auth/scope.go`, adapter `main_test.go`.
- **Supply-chain:** `.github/workflows/release.yml`, `ci.yml`.
- **Prior review:** [docs/architecture/2026-06-18-security-review.md](2026-06-18-security-review.md).
- **Merged PRs:** #1431–#1441 (`gh pr view`), EPIC #1370.

---

**Private file:** none required. No unfixed vulnerability with exploit/reproduction detail exists this cycle — all findings are Fixed, Verified-Safe, or aspirational-with-documented-mitigation; F11 is a one-line config gap, not an exploit.
