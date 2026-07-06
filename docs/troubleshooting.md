<!-- SPDX-License-Identifier: Apache-2.0 -->

# Troubleshooting

Known failure modes, how to diagnose them, and the fix. Every entry here was
hit for real on kind or in CI — if you find a new one, a PR adding it is a
perfect first contribution.

Quick orientation: the platform runs on Kubernetes (kind locally, `zynax up`),
fronted by an Envoy Gateway edge on host port 8080, with workflows compiled to
an engine-agnostic IR and dispatched to Temporal or Argo (ADR-015).

---

## 1. `http://localhost:8080` unreachable after `zynax up`

**Symptom:** `zynax apply` (or `curl http://localhost:8080/healthz`) hangs or
gets connection refused right after cluster bring-up.

**Why:** the first-run contract relies on a chain — kind `extraPortMapping`
(host 8080 → nodePort 30080) → the Envoy Gateway edge proxy Service pinned to
nodePort 30080. If the edge proxy is still rolling out, or a stale vendored
chart dropped the nodePort pin (#1488), the chain breaks silently.

**Diagnose:**

```bash
kubectl -n envoy-gateway-system get svc | grep envoy     # edge proxy Service + nodePort
kubectl -n zynax get gateway                             # Gateway CR accepted?
kubectl -n zynax get svc zynax-api-gateway -o jsonpath='{.spec.ports[0].nodePort}'
```

**Fix:** wait for the edge proxy rollout (`cluster-up.sh` asserts it and the
host path with retries); if the nodePort is wrong, re-run
`scripts/e2e/cluster-up.sh` — it rebuilds chart dependencies from source
specifically to prevent stale-chart drift.

---

## 2. API calls return 401 Unauthorized

**Symptom:** every `/api/v1/*` request is rejected with 401; `/healthz` works.

**Why:** bearer auth is enforced **at the Envoy Gateway edge** (ADR-044,
M8.F), not in api-gateway code. The `SecurityPolicy` checks the
`zynax-edge-apikey` Secret, which stores the **bare key** (no `Bearer `
prefix) — Envoy strips the scheme from the `Authorization` header.

**Diagnose:**

```bash
kubectl -n zynax get securitypolicy                       # Accepted?
kubectl -n zynax get secret zynax-edge-apikey -o jsonpath='{.data.zynax-cli}' | base64 -d
```

**Fix:** pass exactly that key: `zynax --api-key <key> …`. If you rotated the
Secret, restart nothing — the edge reads it dynamically; just use the new key.
A 401 with a correct key usually means the Secret value was stored with a
`Bearer ` prefix — store the bare key.

---

## 3. engine-adapter crash-loops with `Namespace default is not found`

**Symptom:** `zynax-engine-adapter` restarts repeatedly on the full profile;
logs show the Temporal namespace error.

**Why:** the Temporal Helm chart does **not** auto-register a namespace, and
engine-adapter connects to `default` at startup. `cluster-up.sh` registers it
via the admintools pod after the frontend rolls out; if that step raced or the
cluster was brought up by hand, the namespace is missing.

**Diagnose / fix:**

```bash
kubectl -n zynax exec deployment/zynax-temporal-admintools -- \
  temporal operator namespace describe default   # missing?
kubectl -n zynax exec deployment/zynax-temporal-admintools -- \
  temporal operator namespace create default
```

The lite profile (ADR-041) is immune: single-binary dev Temporal
auto-registers `default`.

---

## 4. Workflow CR rejected at admission: `engine '<x>' is not in this namespace's allow-list`

**Symptom:** `kubectl apply` of a `Workflow` CR fails with a
`ValidatingAdmissionPolicy` denial naming the engine and an allow-list.

**Why:** the namespace engine allow-list is enforced by a CEL admission policy
on `spec.engine` (ADR-045, M8.G) when `admissionPolicy.enabled` is set. Unset
`spec.engine` always admits (platform default engine); an empty allow-list
means no restriction.

**Fix:** either omit `spec.engine`, pick an allowed engine, or update the
namespace's params ConfigMap (`…-engine-allowlist-params`, key
`allowedEngines`, comma-separated). Full guide:
[patterns/engine-allowlist-admission.md](patterns/engine-allowlist-admission.md).
Note the policy requires Kubernetes ≥ 1.30 — on older clusters the objects
fail to install (`admissionregistration.k8s.io/v1` VAP not served).

---

## 5. `zynax logs` shows engine history but no capability events

**Symptom:** the `/logs` stream (SSE) carries workflow state transitions but
never the per-capability CloudEvents.

**Why:** the capability-event merge is **best-effort** over NATS JetStream. If
the broker is unreachable (e.g. the ADR-041 lite profile ships no NATS, or the
NATS pod is still starting), the gateway falls back to engine-history-only —
by design, not an error.

**Diagnose:**

```bash
kubectl -n zynax get pods | grep nats
kubectl -n zynax exec deployment/zynax-nats-box -- nats stream ls
```

**Fix:** on the full profile, wait for NATS; on lite, this is expected — use
`zynax status`/`zynax result` for outcomes.

---

## 6. NATS error 10065: `subjects overlap with an existing stream`

**Symptom:** an event publish/subscribe fails with err 10065 and whole event
families stop being delivered.

**Why:** JetStream rejects a stream whose subject filter overlaps another
stream. Zynax derives every stream at a fixed depth-4 entity prefix
(`zynax.<version>.<service>.<entity>`) precisely so filters are identical or
pairwise disjoint by construction (#1149). Hitting 10065 means something
published outside the taxonomy (or a pre-#1149 stream survives).

**Diagnose / fix:**

```bash
kubectl -n zynax exec deployment/zynax-nats-box -- nats stream ls
# Look for streams NOT derived from a 4-segment prefix (or legacy leftovers).
kubectl -n zynax exec deployment/zynax-nats-box -- nats stream rm <legacy-stream>
```

Event types must follow `zynax.<version>.<service>.<entity>.<event_type>`;
the `zynax.dlq.*` prefix is reserved for dead-letter subjects.

---

## 7. `go build` / `go test` fail with workspace module errors in a service directory

**Symptom:** commands inside `services/*/`, `cmd/zynax/`, or `protos/tests/`
fail resolving modules that have nothing to do with your change.

**Why:** the repository root `go.work` lists every module for IDE
convenience, and several intentionally do not build together (ADR-017).

**Fix:** always prefix Go commands in those directories with `GOWORK=off`:

```bash
cd services/workflow-compiler
GOWORK=off go test ./... -race -timeout 60s
```

---

## 8. e2e echo-worker never becomes Ready (Python adapter images)

**Symptom:** on kind, `deployment/echo-worker` stays unready; its logs show a
protobuf `runtime version is lower than the generated code` error.

**Why:** the Python adapter pins its protobuf **runtime** in `uv.lock`, while
the generated stubs are produced by the ci-runner's protoc **gencode**
version. If the gencode advances past the locked runtime, imports fail at
startup.

**Fix:** bump the protobuf runtime in the adapter's `uv.lock` to at least the
ci-runner's gencode version and rebuild the image (`agents/adapters/…`).

---

## 9. A `Workflow` manifest's `on:` transitions silently vanish

**Symptom:** a hand-written workflow YAML applies cleanly but transitions
never fire; the compiled IR has no `on` edges.

**Why:** bare `on` is a YAML 1.1 boolean — many YAML loaders parse the key
`on:` as `true:`. The CLI's `--crd` path emits JSON specifically to dodge
kubectl's coercion.

**Fix:** always quote the key in YAML manifests: `"on":`. The CRD schema and
all shipped examples already do this.

---

## Still stuck?

- Check the [FAQ](faq.md) and [runbooks](runbooks/).
- Search existing issues; if it's new, open one with logs from
  `kubectl -n zynax logs deployment/<service>` — that report is a valued
  contribution in itself.
