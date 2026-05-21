<!-- SPDX-License-Identifier: Apache-2.0 -->

# ADR-020: mTLS with cert-manager for Inter-Service gRPC Authentication

**Status:** Accepted  **Date:** 2026-05-21  
**Related:** ADR-001 (gRPC-first), ADR-009 (Go services), ADR-016 (Layered Testing)  
**Issues:** [#240](https://github.com/zynax-io/zynax/issues/240) (ADR authoring) · [#488](https://github.com/zynax-io/zynax/issues/488) (implementation)  
**Milestone:** M6 — K8s Production-Ready

---

## Context

Today, all inter-service gRPC communication in Zynax is **unauthenticated**. Any process
co-located in the same Docker network (or Kubernetes namespace) can call any service's
gRPC endpoint without presenting credentials. This is acceptable for local development
but is a critical gap for production K8s deployment (M6).

The 2026-05-20 principal architect review rated this as gap **H3** (High) in
`docs/reviews/04-architecture-gaps.md`. The gap was elevated to a blocking concern
for M6 because:

1. Zynax will run in a shared K8s cluster where other pods could call platform services.
2. api-gateway bearer-token auth protects the external boundary but not the internal mesh.
3. CNCF Sandbox submission (M8) requires documented inter-service authentication.

Three options were considered:
- **Option A — mTLS with cert-manager** (selected)
- **Option B — SPIFFE/SPIRE workload identity**
- **Option C — Shared API key / service-account tokens**

---

## Decision

**Adopt mutual TLS (mTLS) for all inter-service gRPC calls in Kubernetes, managed by cert-manager.**

### What mTLS provides

- Each service presents a certificate during the TLS handshake; the peer verifies it.
- Certificates are issued by cert-manager's `ClusterIssuer` (self-signed CA for M6;
  Let's Encrypt or Vault for M7+).
- Per-service identity is encoded in certificate SANs (e.g. `workflow-compiler.zynax.svc`).
- No additional API call or token rotation needed — TLS handles authentication.

### Scope

| Environment | Authentication |
|-------------|----------------|
| Kubernetes (M6+) | mTLS enforced on all inter-service gRPC |
| Docker Compose (local dev) | Insecure gRPC (no TLS) — dev convenience |
| CI/test environment | Insecure gRPC with bufconn in-process transport |

### Implementation approach

```go
// Each service loads TLS credentials from cert-manager-issued files:
creds, err := credentials.NewClientTLSFromFile(cfg.TLSCertPath, "")
conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(creds))

// Server-side:
creds, err := credentials.NewServerTLSFromFile(cfg.TLSCertPath, cfg.TLSKeyPath)
server := grpc.NewServer(grpc.Creds(creds))
```

Configuration via environment variables:
- `ZYNAX_TLS_CERT_PATH` — path to service certificate (PEM)
- `ZYNAX_TLS_KEY_PATH` — path to private key (PEM)
- `ZYNAX_TLS_CA_PATH` — path to CA certificate for peer verification
- `ZYNAX_TLS_ENABLED` — `true` in K8s, unset/`false` in Docker Compose

### cert-manager setup

```yaml
# ClusterIssuer (self-signed for M6)
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: zynax-internal-ca
spec:
  selfSigned: {}

# Certificate per service (example: workflow-compiler)
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: workflow-compiler-tls
  namespace: zynax
spec:
  secretName: workflow-compiler-tls
  issuerRef:
    name: zynax-internal-ca
    kind: ClusterIssuer
  dnsNames:
    - workflow-compiler.<namespace>.svc  # short-form in-cluster DNS
```

---

## Consequences

### Positive
- Per-service identity without shared secrets or token rotation
- cert-manager handles certificate renewal automatically
- Standard K8s pattern — any platform engineer can operate it
- Compatible with service mesh (Istio/Linkerd) if adopted in M8

### Negative / Trade-offs
- cert-manager is a new cluster dependency (Helm chart required)
- Local Docker Compose stays insecure — developers must be aware of this gap
- Integration tests must stub TLS credentials or use bufconn (no network TLS)

### Deferred
- SPIFFE/SPIRE: deferred until CNCF Sandbox (M8). mTLS with cert-manager satisfies
  the M6 security requirement. SPIFFE adds workload federation across clusters,
  which is only relevant post-Sandbox.
- Let's Encrypt / Vault CA: upgrade from self-signed in M7 when external clients
  need to verify service certificates.

---

## Rejected Alternatives

### Option B — SPIFFE/SPIRE workload identity

SPIFFE provides platform-independent workload identity (SVIDs), which are useful when
services run across multiple clusters or clouds. However:
- Adds SPIRE server + node agents as K8s dependencies
- Adds ~2–3 weeks of implementation + operational complexity
- Not necessary for M6 (single-cluster deployment)

Deferred to M8 CNCF Sandbox evaluation.

### Option C — Shared API key / service-account tokens

Shared secrets do not provide per-service identity. Key rotation is manual.
A compromised service exposes all others. Not acceptable for a platform targeting
production workloads. Suitable only as a stepping stone (already done for external
auth via ZYNAX_GW_API_KEY).

---

*See also: `docs/reviews/04-architecture-gaps.md §H3` · `docs/reviews/DECISIONS-NEEDED.md §D2`*
