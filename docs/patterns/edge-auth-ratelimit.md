<!-- SPDX-License-Identifier: Apache-2.0 -->

# Edge auth & rate-limiting: the Gateway API edge

Bearer authentication and rate-limiting are enforced at a **Kubernetes Gateway
API edge (Envoy Gateway)** in front of the api-gateway — not in api-gateway code
(ADR-044, M8.F). The api-gateway keeps only its Zynax core (kind-routing +
compile/submit fan-out); the in-process bearer check and per-pod rate limiter
were deleted. This fixes the HPA `N × RPS` per-pod-limit footgun (one edge-global
limit) and delegates a generic L7 concern to the ecosystem (thin-Zynax, ADR-040).

## How it fits together

```
request → Envoy Gateway (edge)                         request → api-gateway (no auth)
          ├─ HTTPRoute  /api/v1/*   → SecurityPolicy (bearer auth) ─┐
          ├─ HTTPRoute  /healthz,/metrics,… → open (probes)         ├─→ api-gateway:8080 (ClusterIP)
          └─ BackendTrafficPolicy (global rate-limit, profile-gated)┘
```

- The api-gateway Service is **ClusterIP** and its `NetworkPolicy` admits `:8080`
  **only from the Envoy Gateway namespace** — the edge is the sole ingress
  (ADR-044 §4a), so the deleted in-process auth cannot be bypassed by a direct
  in-cluster connection.
- The kind `host:8080` first-run contract is unchanged: the edge takes the
  host-mapped NodePort the api-gateway vacated.

## Enabling the edge

The edge is off by default. On kind:

```bash
EDGE_ENABLED=true zynax up          # cluster-up installs Envoy Gateway (v1.5.0+)
                                    # before the umbrella, then applies the CRs
```

In Helm values, under the api-gateway chart:

```yaml
edge:
  enabled: true
  nodePort: 30080          # fixed NodePort so host:port reaches the edge
  apiKeySecretName: zynax-edge-apikey
```

Requirements and gotchas:

- **Envoy Gateway v1.5.0+** — `apiKeyAuth` is absent in v1.4.x.
- The Envoy Gateway controller must be **Ready before** the Gateway/HTTPRoute/
  SecurityPolicy CRs (ordered prerequisite, ADR-044 §5) — `cluster-up.sh` does
  this; a raw umbrella install would race the CRDs.
- The edge proxy Service is pinned to a fixed NodePort via an `EnvoyProxy`
  resource with **`externalTrafficPolicy: Cluster`** — required so a nodePort hit
  on any node (e.g. kind's `host:8080 → control-plane:30080`) routes cross-node to
  the proxy pod. The Envoy Gateway default (`Local`) would drop it when the proxy
  lands on a different node.

## Bearer auth — the Secret shape (load-bearing)

The `SecurityPolicy` uses `apiKeyAuth`, extracting the credential from the
`Authorization` header. **Envoy strips the `Bearer ` scheme**, so the referenced
Secret stores the **bare key** (no prefix), keyed by client name:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: zynax-edge-apikey
type: Opaque
stringData:
  zynax-cli: <the-bare-key>        # NOT "Bearer <key>"
```

The CLI is unchanged — it still sends `Authorization: Bearer <key>`; the edge
extracts `<key>` and matches it. (Storing `Bearer <key>` was tested and rejected:
it 401s.)

```console
$ zynax --api-key "$(kubectl -n zynax get secret zynax-edge-apikey \
    -o jsonpath='{.data.zynax-cli}' | base64 -d)" apply workflow.yaml   # → 200 via the edge
$ curl -s -o /dev/null -w '%{http_code}' http://localhost:8080/api/v1/workflows/x  # → 401 at the edge
```

## Migrating from the in-process key

| Before (in-process) | After (edge) |
|---------------------|--------------|
| Secret `zynax-gw-api-key`, key `api-key`, value = the key | Secret `zynax-edge-apikey`, key `zynax-cli`, value = the **bare** key |
| `ZYNAX_GW_API_KEY` env on api-gateway | (removed — api-gateway no longer authenticates) |
| api-gateway 401s unauthenticated requests | the **edge** 401s them; api-gateway is edge-only |

## Rate-limiting

A `BackendTrafficPolicy` provides a **global** limit across all api-gateway
replicas (not per-pod). It needs Envoy's rate-limit service + a Redis store, so it
is **profile-gated** (opt-in) — off in the minimal quickstart (ADR-041), on in the
HPA/CI/prod profile where scale-out makes a coordinated limit meaningful.

## See also

- [docs/adr/ADR-044-gateway-api-edge-auth-ratelimit.md](../adr/ADR-044-gateway-api-edge-auth-ratelimit.md) — the decision.
- [docs/adr/ADR-020-zero-trust-auth.md](../adr/ADR-020-zero-trust-auth.md) — mTLS internal mesh (unchanged; the `:8080` edge stays outside it, locked to the Gateway).
