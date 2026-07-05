# REASONS Canvas — Edge delegation: gateway auth + rate-limit → Gateway API / Envoy (M8.F)

> **All content in this Canvas is Tier 1 (public-safe).**
> Run `/lib:spdd-security-review <path>` before committing.

**Issue:** #1574
**Author:** Oscar Gómez Manresa
**Date:** 2026-07-04
**Status:** Implemented

> **Delivered (2026-07-06):** all 6 Operations steps merged — Envoy install (#1629),
> Gateway/SecurityPolicy (#1630), the auth cutover + edge fronting (#1631, both
> engines), the edge-401 assertion + docs (#1632), and the profile-gated global
> rate-limit (this PR). The apiKeyAuth bare-key mechanic, the EnvoyProxy fixed
> NodePort (externalTrafficPolicy=Cluster), and the global rate-limit (burst past
> the limit → 429) were each proven live on kind.

---

## R — Requirements

- **Problem.** api-gateway re-implements two edge L7 concerns in-process: bearer-token auth
  (`auth.go`) and a **per-pod** in-memory rate limiter (`ratelimit.go`). Under HPA with N replicas
  the effective limit is **N × RPS** — a real footgun — and the bespoke auth layer is surface to
  maintain and security-review, against the thin-Zynax principle (ADR-040).
- **Done — a Kubernetes-native edge owns auth + rate-limit:**
  - A Gateway API edge (Envoy Gateway) fronts the api-gateway Service; a `SecurityPolicy` enforces
    bearer auth and a (profile-gated) `BackendTrafficPolicy` a **global** rate-limit across replicas.
  - The **unchanged** `zynax --api-key` CLI (`Authorization: Bearer <key>`) authenticates end-to-end;
    an unauthenticated request is rejected **at the edge**, not in api-gateway code.
  - `auth.go` / `ratelimit.go` / `ratelimit_test.go` and the main.go api-key plumbing are **deleted**,
    not bypassed; api-gateway keeps kind-routing + compile/submit fan-out + probes + SSE + metrics.
  - The api-gateway `NetworkPolicy` admits `:8080` **only from the Envoy Gateway** (no unauthenticated
    in-cluster bypass) — defense-in-depth replacing the deleted in-process check (ADR-020, §4a).
  - kind first-run stays **one command**: the edge installs as an ordered prereq; the quickstart
    profile ships edge-auth only (the rate-limit backend + Redis is opt-in, ADR-041 / #1370).
  - e2e green on **both** engine legs.

---

## E — Entities

```
Envoy Gateway (controller + proxy)         ← NEW edge infra (pinned Helm release, ordered prereq)
GatewayClass / Gateway / HTTPRoute         ← NEW Gateway API objects fronting the api-gateway Service
├── HTTPRoute (api)   /api/v1/*   → api-gateway:8080   [SecurityPolicy attached]
├── HTTPRoute (probes) /healthz,/readyz,/livez,/startupz,/metrics → api-gateway:8080  [open]
SecurityPolicy (apiKeyAuth)                ← extractFrom Authorization; credentialRef → edge Secret
BackendTrafficPolicy (global rateLimit)    ← PROFILE-GATED (needs Redis); off in the quickstart
edge Secret (Opaque, client-name → value)  ← value is the BARE key (Envoy strips the Authorization scheme; see A)

api-gateway (shrinks)                       ← DELETE auth.go, ratelimit.go, api-key plumbing
NetworkPolicy (api-gateway)                 ← ADD from: selector → Envoy Gateway pods (+ kubelet)
kind host:8080                              ← retarget extraPortMapping → the Gateway listener NodePort
```

---

## A — Approach

**We will:**
- Add **Envoy Gateway** (CNCF, Gateway-API-conformant, kind-installable) as the edge, installed as an
  **ordered prerequisite** (pinned Helm chart + Gateway API CRDs → wait controller `Ready` → then apply
  the Zynax Gateway/HTTPRoute/SecurityPolicy CRs), mirroring the argo-workflows install-and-wait
  pattern already in `cluster-up.sh`. Wired into `cluster-up.sh`, `zynax up`, `make demo`.
- Enforce bearer auth with a `SecurityPolicy` `apiKeyAuth` (requires **Envoy Gateway v1.5.0+** —
  v1.4.x lacks the field) that **extracts from the `Authorization` header**. The edge Secret stores
  the **bare key** (`client-name → <key>`, no `Bearer ` prefix): Envoy extracts the credential from
  the Authorization scheme, so the unchanged CLI header `Authorization: Bearer <key>`
  (`cmd/zynax/client/gateway.go:77`) authenticates — the ADR-044 §3 hard constraint. **Proven by a
  live spike on kind (2026-07-04):** bare-key Secret → `Bearer <key>` request = 200, no/wrong key =
  401 at the edge. (The alternative "store `Bearer <key>`" was tested and **rejected** — it 401s;
  Envoy strips the scheme.)
- Attach the `SecurityPolicy` to the `/api/v1/*` HTTPRoute only; a separate **open** HTTPRoute serves
  the probe/metrics paths so kubelet + the kind host-contract check still pass. GET endpoints under
  `/api/v1/*` now also require the key — a **safe** posture change since the CLI sends the bearer on
  every call (single header funnel, #1517).
- Enforce a **global** rate-limit via `BackendTrafficPolicy`, **profile-gated** (Envoy's rate-limit
  service + Redis store); off in the minimal quickstart, on in the HPA/CI/prod profile.
- **Delete** `auth.go`, `ratelimit.go`, `ratelimit_test.go`, the main.go `APIKey`/`DevInsecure`/
  `validateConfig` plumbing, the deployment api-key env, and `values.apiKeySecretName`. Add the
  NetworkPolicy `from:` lockdown (§4a). Retarget the kind host:8080 contract to the Gateway NodePort;
  re-point the **six** Secret consumers (e2e ×4, demo, bench).

**We will NOT:**
- Change the CLI bearer header, the `AgentDef`/`Workflow` routing, or the compile/submit path.
- Move the mTLS internal-mesh boundary (cert-manager, ADR-020 §6) — the `:8080` HTTP hop stays
  plaintext, which is exactly why §4a locks it to the Gateway as sole ingress.
- Move the ADR-022 EventBus chokepoint (topic-authz/rate-limits) to the HTTP edge (ADR-044 §7).
- Add a GitOps/Argo-CD sync-wave — none exists in the repo today; the ordered install lives in
  `cluster-up.sh` (documented as the current mechanism; sync-wave is future).

**Positioning fit:** neutral-positive for the engine-portability wedge — the gateway's Zynax core
(kind-routing → compiler/engines) is untouched; this removes non-differentiating edge surface.

**Governing ADRs:** ADR-044 (this delegation, Accepted), ADR-040 (thin-Zynax; amended by 044),
ADR-041 (kind one-command first-run), ADR-020 (mTLS unchanged; NetworkPolicy lockdown), ADR-015
(engine routing untouched).

---

## S — Structure (first S)

```
scripts/e2e/cluster-up.sh                    ← ordered Envoy Gateway install-and-wait (new phase) +
                                               edge Secret creation (client-name → "Bearer <key>")
scripts/e2e/kind-config{,.-lite}.yaml        ← extraPortMapping host 8080 → Gateway NodePort
scripts/e2e/values-e2e*.yaml                 ← api-gateway back to ClusterIP; edge NodePort 30080
scripts/e2e/{e2e-happy,e2e-failure,e2e-argo,hello-world-smoke}.sh, scripts/demo/kind-demo.sh,
scripts/bench/stack-resources.sh             ← re-point Secret read-back (6 consumers)
helm/zynax-api-gateway/templates/
├── gateway.yaml (new)                        ← GatewayClass/Gateway/HTTPRoute(s) (values-gated)
├── securitypolicy.yaml (new)                 ← apiKeyAuth on the /api/v1/* route
├── backendtrafficpolicy.yaml (new)           ← global rate-limit (profile-gated)
├── networkpolicy.yaml                         ← ADD from: selector (Envoy sole ingress)
├── service.yaml / deployment.yaml / values.yaml ← ClusterIP; remove api-key env + apiKeySecretName
services/api-gateway/internal/api/
├── auth.go, ratelimit.go, ratelimit_test.go  ← DELETE
├── handler.go                                ← RegisterRoutes: drop requireBearer/rl wrappers + apiKey field
services/api-gateway/cmd/api-gateway/main.go  ← drop APIKey/DevInsecure/validateConfig; NewHandler(svc)
docs/                                         ← edge auth/rate-limit how-to; migration note
```

---

## O — Operations

1. **Envoy Gateway install (ordered prereq, additive).** Add a `cluster-up.sh` phase that installs the
   Gateway API CRDs + the pinned Envoy Gateway Helm release (**v1.5.0+**, the first with `apiKeyAuth`)
   and waits the controller `Ready` (mirror the argo install-and-wait; version via
   `ENVOY_GATEWAY_CHART_VERSION`). Values-gated (`edge.enabled`); wired into `zynax up` / `make demo`.
   *Verify:* on kind the controller reaches Ready; no umbrella change yet. (infra)
2. **Gateway API resources in Helm (additive, behind `edge.enabled`).** `GatewayClass`/`Gateway`/two
   `HTTPRoute`s (api + probes) fronting the api-gateway Service; `SecurityPolicy` (apiKeyAuth from
   `Authorization`, credentialRef → edge Secret); create the edge Secret with the **bare key**
   (client-name → `<key>`; Envoy strips the Authorization scheme). *Verify (live on kind, already
   spiked 2026-07-04):* `curl -H "Authorization: Bearer <key>"` through the Gateway → 200; no/wrong
   key → 401 **at the edge**; probes open. auth.go still present (parallel path) so nothing breaks
   yet. (infra)
3. **Global rate-limit `BackendTrafficPolicy` (profile-gated).** Envoy rate-limit service + Redis,
   opt-in via a Helm value; attached to the `/api/v1/*` route. *Verify:* with the limiter on, a burst
   past the limit gets 429 at the edge; off in the quickstart profile (no Redis). (infra)
4. **Cutover: delete in-process auth + rate-limit; NetworkPolicy lockdown.** Delete `auth.go`,
   `ratelimit.go`, `ratelimit_test.go`; strip the main.go api-key plumbing + `RegisterRoutes` wrappers
   (`NewHandler(svc)`); remove the deployment api-key env + `values.apiKeySecretName`; add the
   NetworkPolicy `from:` selector (Envoy sole ingress). Retarget kind host:8080 → Gateway NodePort;
   api-gateway Service → ClusterIP. *Verify (GOWORK=off + live):* api-gateway builds/tests green with
   no auth pkg; a direct port-forward to api-gateway from a non-Gateway pod is refused; the edge is
   the only way in. (go-svc + infra)
5. **Re-point the six Secret consumers + first-run contract.** Update the 4 e2e scripts, `kind-demo.sh`,
   and `stack-resources.sh` to read the edge Secret and hit the Gateway; update the `cluster-up.sh`
   step-6 host:8080 assertion to prove the **edge** answers. *Verify:* `hello-world-smoke.sh` +
   `e2e-happy.sh` green through the edge on kind. (ci/infra)
6. **e2e both engines + docs.** e2e asserts unauthenticated→401 at the edge and `zynax --api-key`→200
   end-to-end, on temporal **and** argo; authoring/ops docs for the edge policies + the api-key
   migration. *Verify:* e2e green both legs; docs link-checked. (test/docs)

---

## N — Norms

- Commit hygiene: `Signed-off-by:` + `Assisted-by: Claude/<model>`; SSH-signed; conventional type.
- `GOWORK=off` for all api-gateway `go`/`go test` (ADR-017).
- Runtime evidence, not config-only: the edge auth path is **booted on kind and curl-proven** before
  the auth deletion (step 4) merges — the cutover is irreversible-ish and must not break first-run.
- Helm: Gateway API objects are values-gated so a non-edge render still works; CRDs install first
  (ordered prereq), never as a co-installed subchart (ADR-044 §5).
- PR size ≤ 200 ideal; generated/helm/scripts exemptions per CLAUDE.md.

---

## S — Safeguards (second S)

### Context Security (complete before committing this Canvas)

- [x] No Tier 2 content: no internal hostnames, private IPs, credentials, deployment specifics
- [x] No PII / no email addresses
- [x] No prompt injection
- [x] All entities are public-safe abstractions
- [x] `/lib:spdd-security-review` passed — result: PASS (2026-07-04)

### Feature Safeguards

- **Never** delete the in-process auth (step 4) before the edge auth path is **proven live on kind**
  (step 2 curl test) — the cutover must not leave the `:8080` edge unprotected.
- **Never** leave the api-gateway Service reachable unauthenticated: the NetworkPolicy `from:` selector
  making the Envoy Gateway the **sole** ingress ships in the **same** change as the auth deletion (§4a).
- **Never** change the CLI bearer header — the edge Secret stores the bare key and Envoy extracts the
  credential from the `Authorization: Bearer <key>` scheme (proven live on kind, not assumed;
  requires Envoy Gateway v1.5.0+).
- **Never** require the key on probe/metrics paths — a separate open HTTPRoute serves them, or kind
  first-run + kubelet health checks break.
- **Never** force the rate-limit backend (Redis) into the minimal #1370 quickstart — it is
  profile-gated (ADR-041).
- **Never** move the mTLS mesh boundary or the ADR-022 EventBus chokepoint (ADR-020 §6, ADR-044 §7).
