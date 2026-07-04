#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# cluster-up.sh — bring up a kind cluster and deploy the full Zynax stack.
#
# EPIC G (#770) step 1 / #809. Creates a reproducible, ephemeral Kubernetes
# cluster via kind and installs the `zynax-umbrella` Helm chart so that the
# e2e assertion scripts (#810–#813) run against a real cluster.
#
# Idempotent: re-running against an existing cluster reuses it and performs a
# `helm upgrade --install`, so the script is safe to invoke repeatedly.
#
# Usage:
#   scripts/e2e/cluster-up.sh
#
# Environment overrides:
#   CLUSTER_NAME          kind cluster name           (default: zynax-e2e)
#   NAMESPACE             release namespace           (default: zynax)
#   RELEASE_NAME          Helm release name           (default: zynax)
#   CERT_MANAGER_VERSION  cert-manager release tag    (default: v1.14.5)
#   KIND_NODE_IMAGE       kind node image (digest-pinnable)
#   WAIT_TIMEOUT          per-resource rollout wait   (default: 600s)
#   E2E_IMAGE_TAG         service image tag override — set by e2e-smoke.yml to
#                         pr-<head-sha> so the cluster runs the exact staging
#                         images built pre-merge (#1118 / ADR-027). Unset =
#                         values-e2e.yaml default (:main lane).
#   E2E_IMAGE_PREFIX      registry prefix for E2E_IMAGE_TAG
#                         (default: ghcr.io/zynax-io/zynax/staging)
#   E2E_ENGINE            workflow engine for the deployed stack: temporal|argo
#                         (default: temporal). argo additionally installs the
#                         Argo Workflows control plane and deploys the umbrella
#                         with values-e2e-argo.yaml (#1071, ADR-015).
#   ARGO_WORKFLOWS_CHART_VERSION
#                         argo-helm argo-workflows chart version pin
#                         (default: 0.47.5 → Argo Workflows v3.7.11)
#   EDGE_ENABLED          install the Envoy Gateway edge (bearer auth + rate-limit
#                         delegation, M8.F/ADR-044) before the umbrella: true|false
#                         (default: false). Inherited by `zynax up` from the env.
#   ENVOY_GATEWAY_CHART_VERSION
#                         gateway-helm chart version pin — v1.5.0+ required for
#                         apiKeyAuth (default: v1.5.0)
#   PROFILE               stack profile: full|lite (default: full). lite is the
#                         ADR-041 lean laptop profile — collapses Temporal to a
#                         single in-memory start-dev pod (manifests/temporal-dev.
#                         yaml) and drops event-bus + NATS + memory-service via
#                         values-lite.yaml. CI uses full.
#
# Minimum host resources: 4 CPU, 8 GB RAM (see scripts/e2e/README.md).

set -euo pipefail

# ── configuration ───────────────────────────────────────────────────────────────

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

CLUSTER_NAME="${CLUSTER_NAME:-zynax-e2e}"
NAMESPACE="${NAMESPACE:-zynax}"
RELEASE_NAME="${RELEASE_NAME:-zynax}"
CERT_MANAGER_VERSION="${CERT_MANAGER_VERSION:-v1.14.5}"
# kind node image — pin to a digest in CI for reproducibility. The default tag
# tracks the kind release that the harness is validated against.
KIND_NODE_IMAGE="${KIND_NODE_IMAGE:-kindest/node:v1.29.2}"
WAIT_TIMEOUT="${WAIT_TIMEOUT:-600s}"
# Engine matrix axis (#1071, EPIC #771): which engine the deployed stack runs.
# Selection flows through umbrella values only — never hardcoded (ADR-015).
E2E_ENGINE="${E2E_ENGINE:-temporal}"
ARGO_WORKFLOWS_CHART_VERSION="${ARGO_WORKFLOWS_CHART_VERSION:-0.47.5}"
# Edge (M8.F, ADR-044): when EDGE_ENABLED=true, install the Envoy Gateway edge as
# an ordered prerequisite BEFORE the umbrella, so its Gateway/HTTPRoute/
# SecurityPolicy CRs admit cleanly (controller Ready first). The gateway-helm
# chart bundles the Gateway API CRDs, so a fresh cluster needs only this install.
# apiKeyAuth (bearer auth at the edge) requires Envoy Gateway v1.5.0+.
EDGE_ENABLED="${EDGE_ENABLED:-false}"
ENVOY_GATEWAY_CHART_VERSION="${ENVOY_GATEWAY_CHART_VERSION:-v1.5.0}"
# Stack profile (ADR-041). "full" (default, == CI) deploys the production-
# mirroring topology: the 5-pod Temporal chart, event-bus, memory-service.
# "lite" is the lean laptop profile — it collapses Temporal to ONE in-memory
# `start-dev` pod (manifests/temporal-dev.yaml) and drops event-bus + NATS +
# memory-service via values-lite.yaml. Same charts, same images, lighter floor.
PROFILE="${PROFILE:-full}"
# When set to a non-empty value, side-load the locally-built service images into
# the kind cluster with `kind load docker-image` before the Helm install, so the
# cluster runs the images already on the host instead of pulling from GHCR. The
# laptop demo path (`make kind-up` / `make demo`) sets this to make cold-start
# fast and offline-friendly; CI leaves it unset and lets the cluster pull the
# pinned staging/`:main` lane from GHCR (existing behaviour, ADR-027).
KIND_LOAD_IMAGES="${KIND_LOAD_IMAGES:-}"
# Registry/tag the loaded images carry — must match what values-e2e.yaml asks the
# chart to run (`:main`, the lane the chart's image.tag override pins). The echo
# capability worker image (langgraph-adapter, manifests/echo-worker.yaml) is
# loaded too so the dispatch chain round-trips with IfNotPresent.
KIND_LOAD_REGISTRY="${KIND_LOAD_REGISTRY:-ghcr.io/zynax-io/zynax}"
KIND_LOAD_TAG="${KIND_LOAD_TAG:-main}"

KIND_CONFIG="${SCRIPT_DIR}/kind-config.yaml"
# The lean profile fits on ONE node — use the single-node config, which removes
# two nodes' kubelet/containerd/kindnet tax and avoids loading the service images
# into three nodes' containerd (the dominant laptop cost; see
# docs/benchmarks/kind-lean-resources.md).
[[ "${PROFILE}" == "lite" ]] && KIND_CONFIG="${SCRIPT_DIR}/kind-config-lite.yaml"
UMBRELLA_CHART="${REPO_ROOT}/helm/zynax-umbrella"

# The Zynax service Deployments that must reach a healthy rollout. All 7
# services ship a GHCR image since #1089 added event-bus + memory-service to
# the pre-merge build matrix, so the full set is deployed and asserted (#1090).
# Deployment names are pinned via fullnameOverride in values-e2e.yaml to
# `zynax-<svc>` (so the umbrella's inter-service addresses resolve), not the
# release-prefixed `zynax-zynax-<svc>` default.
SERVICE_DEPLOYMENTS=(
  "zynax-api-gateway"
  "zynax-workflow-compiler"
  "zynax-engine-adapter"
  "zynax-task-broker"
  "zynax-agent-registry"
)
# event-bus + memory-service ship in the full profile only; the lean profile
# (values-lite.yaml) disables them, so they have no Deployment to wait on.
if [[ "${PROFILE:-full}" != "lite" ]]; then
  SERVICE_DEPLOYMENTS+=("zynax-event-bus" "zynax-memory-service")
fi

# ── helpers ──────────────────────────────────────────────────────────────────────

log()  { printf '\033[1;34m[cluster-up]\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m[cluster-up]\033[0m %s\n' "$*" >&2; }
die()  { printf '\033[1;31m[cluster-up]\033[0m %s\n' "$*" >&2; exit 1; }

require() {
  command -v "$1" >/dev/null 2>&1 || die "required tool not found on PATH: $1"
}

# ── preflight ──────────────────────────────────────────────────────────────────

require kind
require kubectl
require helm
require openssl

case "${E2E_ENGINE}" in
  temporal|argo) ;;
  *) die "E2E_ENGINE must be 'temporal' or 'argo' (got: '${E2E_ENGINE}')" ;;
esac

case "${PROFILE}" in
  full|lite) ;;
  *) die "PROFILE must be 'full' or 'lite' (got: '${PROFILE}')" ;;
esac

[[ -f "${KIND_CONFIG}" ]]   || die "kind config not found: ${KIND_CONFIG}"
[[ -d "${UMBRELLA_CHART}" ]] || die "umbrella chart not found: ${UMBRELLA_CHART}"

# ── 1. create (or reuse) the kind cluster ────────────────────────────────────────

if kind get clusters 2>/dev/null | grep -qx "${CLUSTER_NAME}"; then
  log "kind cluster '${CLUSTER_NAME}' already exists — reusing (idempotent)."
else
  log "creating kind cluster '${CLUSTER_NAME}' (node image: ${KIND_NODE_IMAGE})…"
  kind create cluster \
    --name "${CLUSTER_NAME}" \
    --image "${KIND_NODE_IMAGE}" \
    --config "${KIND_CONFIG}" \
    --wait "${WAIT_TIMEOUT}"
fi

# Point kubectl/helm at the cluster regardless of how it was created.
kubectl config use-context "kind-${CLUSTER_NAME}" >/dev/null

# ── 1.5 [opt-in] side-load locally-built images into the cluster ─────────────────

# When KIND_LOAD_IMAGES is set (the laptop demo path), copy the host's local
# `<registry>/<svc>:<tag>` images straight into the kind nodes so the chart runs
# them with IfNotPresent instead of pulling from GHCR — fast, offline-friendly
# cold-start. Skipped silently in CI (unset), where the cluster pulls the pinned
# staging/`:main` lane. Each `kind load` no-ops gracefully if the host image is
# absent (warn, continue) — a missing image surfaces as an ImagePull later, with
# the same diagnostics as the GHCR path.
if [[ -n "${KIND_LOAD_IMAGES}" ]]; then
  log "side-loading local images into kind cluster '${CLUSTER_NAME}' (tag: ${KIND_LOAD_TAG})…"
  for svc in api-gateway workflow-compiler engine-adapter task-broker \
             agent-registry event-bus memory-service langgraph-adapter; do
    img="${KIND_LOAD_REGISTRY}/${svc}:${KIND_LOAD_TAG}"
    if docker image inspect "${img}" >/dev/null 2>&1; then
      log "  → loading ${img}"
      kind load docker-image "${img}" --name "${CLUSTER_NAME}"
    else
      warn "  host image not found, skipping (will pull from registry): ${img}"
    fi
  done
  log "image side-load complete."
fi

# ── 2. install cert-manager (CRDs + controllers) ─────────────────────────────────

# The umbrella chart's zynax-cert-manager subchart creates Certificate /
# ClusterIssuer resources but does NOT install cert-manager itself (ADR-020).
# Install upstream cert-manager so those CRDs exist; idempotent via upgrade.
log "installing cert-manager ${CERT_MANAGER_VERSION}…"
helm upgrade --install cert-manager cert-manager \
  --repo https://charts.jetstack.io \
  --namespace cert-manager \
  --create-namespace \
  --version "${CERT_MANAGER_VERSION}" \
  --set installCRDs=true \
  --wait \
  --timeout "${WAIT_TIMEOUT}"

# ── 2.5 provision Postgres + Temporal credentials ───────────────────────────────

# The Temporal schema Job and server connect to the bundled Postgres using the
# `temporal-db` Secret, and the Postgres subchart reads its passwords from
# the `zynax-postgres-creds` Secret (values-e2e.yaml sets auth.existingSecret).
# Pre-create both with a single shared password so they always match. The schema
# Job runs `temporal-sql-tool create-database`, which only the Postgres superuser
# can do — values-e2e.yaml points Temporal at user `postgres`, so temporal-db
# carries the superuser password.
#
# Idempotent: the password is generated once and reused on re-runs so it never
# diverges from an already-initialised Postgres data volume.
log "provisioning Postgres + Temporal credentials in namespace '${NAMESPACE}'…"
kubectl create namespace "${NAMESPACE}" --dry-run=client -o yaml | kubectl apply -f -

if kubectl -n "${NAMESPACE}" get secret zynax-postgres-creds >/dev/null 2>&1; then
  log "  reusing existing zynax-postgres-creds Secret (idempotent)."
  PG_PASSWORD="$(kubectl -n "${NAMESPACE}" get secret zynax-postgres-creds \
    -o jsonpath='{.data.postgres-password}' | base64 -d)"
else
  PG_PASSWORD="$(openssl rand -hex 16)"
  kubectl -n "${NAMESPACE}" create secret generic zynax-postgres-creds \
    --from-literal=postgres-password="${PG_PASSWORD}" \
    --from-literal=password="${PG_PASSWORD}"
fi

if ! kubectl -n "${NAMESPACE}" get secret temporal-db >/dev/null 2>&1; then
  kubectl -n "${NAMESPACE}" create secret generic temporal-db \
    --from-literal=password="${PG_PASSWORD}"
fi

# api-gateway refuses to start without ZYNAX_GW_API_KEY (it reads the `api-key`
# key from the Secret named by api-gateway's apiKeySecretName, default
# `zynax-gw-api-key`). Provision a throwaway key for the smoke cluster.
if ! kubectl -n "${NAMESPACE}" get secret zynax-gw-api-key >/dev/null 2>&1; then
  kubectl -n "${NAMESPACE}" create secret generic zynax-gw-api-key \
    --from-literal=api-key="$(openssl rand -hex 16)"
fi

# ── 2.6 [argo only] install Argo Workflows + the IR-interpreter template ────────

# Engine matrix (#1071, EPIC #771): when E2E_ENGINE=argo, engine-adapter runs
# with activeEngine=argo (values-e2e-argo.yaml) and needs the Argo Workflows
# control plane (CRDs + workflow-controller + argo-server) in-cluster. Install
# it via the version-pinned community argo-helm chart — the same bring-up
# pattern as cert-manager above; idempotent via upgrade --install.
#
#   server.authModes={server}            → tokenless API for the smoke cluster
#                                          (the chart's server.secure default is
#                                          false, so the API is plain HTTP and
#                                          probes match)
#   workflow.serviceAccount.create=true  → executor SA "argo-workflow" + RBAC
#   controller.workflowNamespaces        → SA/RBAC land in the release namespace,
#                                          where ArgoEngine creates Workflow CRs
if [[ "${E2E_ENGINE}" == "argo" ]]; then
  log "installing Argo Workflows (argo-helm chart ${ARGO_WORKFLOWS_CHART_VERSION})…"
  helm upgrade --install argo-workflows argo-workflows \
    --repo https://argoproj.github.io/argo-helm \
    --namespace argo \
    --create-namespace \
    --version "${ARGO_WORKFLOWS_CHART_VERSION}" \
    --set "server.authModes={server}" \
    --set workflow.serviceAccount.create=true \
    --set "controller.workflowNamespaces={${NAMESPACE}}" \
    --set controller.resources.requests.cpu=50m \
    --set controller.resources.requests.memory=64Mi \
    --set controller.resources.limits.cpu=400m \
    --set controller.resources.limits.memory=256Mi \
    --set server.resources.requests.cpu=25m \
    --set server.resources.requests.memory=64Mi \
    --set server.resources.limits.cpu=400m \
    --set server.resources.limits.memory=256Mi \
    --wait \
    --timeout "${WAIT_TIMEOUT}"

  # The WorkflowTemplate that ArgoEngine instantiates per submitted workflow
  # (ZYNAX_ENGINE_ADAPTER_ARGO_WORKFLOW_TEMPLATE_REF → zynax-ir-interpreter).
  log "applying the zynax-ir-interpreter WorkflowTemplate…"
  kubectl -n "${NAMESPACE}" apply -f "${SCRIPT_DIR}/manifests/argo-ir-interpreter.yaml"
fi

# ── 2.7 [edge] install the Envoy Gateway edge (ordered prereq, M8.F/ADR-044) ─────

# When EDGE_ENABLED=true, bearer auth + rate-limiting are delegated to a Gateway
# API edge (Envoy Gateway) instead of api-gateway in-process middleware. The edge
# controller and its CRDs (the Gateway API + Envoy Gateway extension CRDs are
# bundled by the gateway-helm chart) must be Ready BEFORE the umbrella so the
# Gateway/HTTPRoute/SecurityPolicy CRs the chart carries admit cleanly (ADR-044
# §5). apiKeyAuth — the bearer check — requires Envoy Gateway v1.5.0+. Same
# install-and-wait idiom as cert-manager/argo above; idempotent via upgrade
# --install. Off by default until the edge resources + auth cutover land.
if [[ "${EDGE_ENABLED}" == "true" ]]; then
  log "installing Envoy Gateway edge (gateway-helm ${ENVOY_GATEWAY_CHART_VERSION})…"
  helm upgrade --install eg oci://docker.io/envoyproxy/gateway-helm \
    --version "${ENVOY_GATEWAY_CHART_VERSION}" \
    --namespace envoy-gateway-system \
    --create-namespace \
    --wait \
    --timeout "${WAIT_TIMEOUT}"
fi

# ── 3. deploy the full Zynax stack via the umbrella chart ────────────────────────

# ALWAYS (re)build chart dependencies from the live subchart sources before the
# install. The umbrella vendors its subcharts as packaged `.tgz` files under
# charts/, and those committed artifacts can drift from the subchart source —
# e.g. the api-gateway tgz once predated the template that renders
# service.nodePort, so the deployed Service silently dropped the pinned NodePort
# 30080 and fell back to a kube-proxy-random nodePort the kind extraPortMapping
# (host 8080 → 30080) does not target, leaving http://localhost:8080 unreachable
# out of the box (#1488). A prior guard only rebuilt when charts/ was EMPTY, so a
# stale-but-present tgz was deployed verbatim. Rebuilding unconditionally makes
# the subchart source the single source of truth — it is idempotent and fast
# (file:// deps, no network), and CI's helm-lint already rebuilds the same way.
log "building umbrella chart dependencies from source (subchart source is the SoT)…"
helm dependency build "${UMBRELLA_CHART}"

log "deploying zynax-umbrella as release '${RELEASE_NAME}' in namespace '${NAMESPACE}' (engine: ${E2E_ENGINE})…"
# values-e2e.yaml carries the e2e-only overrides (shared with helm-upgrade.sh so
# the release shape is identical across revisions): service image tags pinned to
# `main`, event-bus/memory-service enabled (#1090), and the Postgres /
# Temporal credential wiring.
# E2E_IMAGE_TAG (set by e2e-smoke.yml for docker-touching PRs) repoints the 7
# deployed services at the pre-merge staging lane — helm-upgrade.sh applies the
# same overrides so the release shape stays identical across revisions.
# Engine-selecting values overlay (#1071): the argo leg layers
# values-e2e-argo.yaml on top of values-e2e.yaml so the release shape is
# identical except for the engine selection (ADR-015).
ENGINE_VALUES=()
if [[ "${E2E_ENGINE}" == "argo" ]]; then
  ENGINE_VALUES+=(-f "${SCRIPT_DIR}/values-e2e-argo.yaml")
fi
# Lean profile overlay (ADR-041): layered last so it wins — disables the
# Temporal chart, event-bus, NATS, memory-service and trims the Postgres PVC.
PROFILE_VALUES=()
if [[ "${PROFILE}" == "lite" ]]; then
  log "lean profile: layering values-lite.yaml (Temporal→dev pod; no event-bus/NATS/memory-service)…"
  PROFILE_VALUES+=(-f "${SCRIPT_DIR}/values-lite.yaml")
fi
IMAGE_OVERRIDES=()
if [[ -n "${E2E_IMAGE_TAG:-}" ]]; then
  E2E_IMAGE_PREFIX="${E2E_IMAGE_PREFIX:-ghcr.io/zynax-io/zynax/staging}"
  log "service image lane override: ${E2E_IMAGE_PREFIX}/<svc>:${E2E_IMAGE_TAG}"
  for svc in api-gateway workflow-compiler engine-adapter task-broker agent-registry event-bus memory-service; do
    IMAGE_OVERRIDES+=(
      --set "zynax-${svc}.image.repository=${E2E_IMAGE_PREFIX}/${svc}"
      --set "zynax-${svc}.image.tag=${E2E_IMAGE_TAG}"
    )
  done
fi
# NOTE: no --wait here. engine-adapter cannot become Ready until the Temporal
# `default` namespace is registered (step 3.5), and a Helm --wait would deadlock
# waiting on it. Readiness is asserted explicitly by the rollout loop in step 4.
helm upgrade --install "${RELEASE_NAME}" "${UMBRELLA_CHART}" \
  --namespace "${NAMESPACE}" \
  --create-namespace \
  -f "${SCRIPT_DIR}/values-e2e.yaml" \
  "${ENGINE_VALUES[@]}" \
  "${PROFILE_VALUES[@]}" \
  --set zynax-cert-manager.enabled=true \
  "${IMAGE_OVERRIDES[@]}"

# ── 3.5 bring Temporal up + ensure the 'default' namespace exists ────────────────

if [[ "${PROFILE}" == "lite" ]]; then
  # Lean profile: the chart is off (values-lite.yaml). Deploy the single-binary
  # in-memory dev Temporal, which auto-registers the 'default' namespace at boot
  # — so there is no admintools pod and no namespace-registration loop. Its
  # Service is named zynax-temporal-frontend (the exact address engine-adapter
  # dials), so the swap needs no engine-adapter override.
  log "lean profile: deploying single-binary dev Temporal (in-memory)…"
  kubectl -n "${NAMESPACE}" apply -f "${SCRIPT_DIR}/manifests/temporal-dev.yaml"
  kubectl -n "${NAMESPACE}" rollout status deployment/zynax-temporal-dev \
    --timeout "${WAIT_TIMEOUT}"
  log "dev Temporal is up ('default' namespace auto-registered by start-dev)."
else
  # The temporalio/temporal chart does NOT auto-register a namespace, but
  # engine-adapter connects to namespace 'default' on startup and crash-loops with
  # "Namespace default is not found" until it exists. Wait for the Temporal frontend
  # to roll out, then register it via the admintools pod. Idempotent: skip if it is
  # already present (e.g. a re-run against a reused cluster).
  log "waiting for Temporal frontend, then registering the 'default' namespace…"
  kubectl -n "${NAMESPACE}" rollout status \
    "deployment/${RELEASE_NAME}-temporal-frontend" --timeout "${WAIT_TIMEOUT}"

  ADMINTOOLS="deployment/${RELEASE_NAME}-temporal-admintools"
  namespace_ready=""
  for _ in $(seq 1 30); do
    if kubectl -n "${NAMESPACE}" exec "${ADMINTOOLS}" -- \
         temporal operator namespace describe default >/dev/null 2>&1; then
      namespace_ready="yes"
      break
    fi
    kubectl -n "${NAMESPACE}" exec "${ADMINTOOLS}" -- \
      temporal operator namespace create default >/dev/null 2>&1 || true
    sleep 5
  done
  [[ -n "${namespace_ready}" ]] || die "Temporal 'default' namespace did not register"
  log "Temporal 'default' namespace is registered."
fi

# ── 4. wait for the profile's service deployments to become healthy ──────────────

log "waiting for ${#SERVICE_DEPLOYMENTS[@]} service deployments to roll out (profile: ${PROFILE})…"
for dep in "${SERVICE_DEPLOYMENTS[@]}"; do
  if ! kubectl -n "${NAMESPACE}" get deployment "${dep}" >/dev/null 2>&1; then
    die "expected deployment not found: ${dep} (umbrella values out of sync?)"
  fi
  log "  → ${dep}"
  kubectl -n "${NAMESPACE}" rollout status "deployment/${dep}" \
    --timeout "${WAIT_TIMEOUT}"
done

log "all ${#SERVICE_DEPLOYMENTS[@]} service deployments are healthy."

# ── 5. deploy the echo capability worker (#1088) ────────────────────────────────

# The umbrella deploys only the platform services — no capability provider — so
# dispatched tasks are never claimed. Deploy a minimal langgraph-adapter that
# registers the "echo" capability with agent-registry and completes its tasks,
# letting spec/workflows/examples/e2e-demo.yaml reach a terminal succeeded state.
# agent-registry is healthy by now (step 4), so the worker registers on startup.
log "deploying echo capability worker…"
kubectl -n "${NAMESPACE}" apply -f "${SCRIPT_DIR}/manifests/echo-worker.yaml"
kubectl -n "${NAMESPACE}" rollout status deployment/echo-worker --timeout "${WAIT_TIMEOUT}"
log "echo-worker is healthy and registered."

kubectl -n "${NAMESPACE}" get pods -o wide

# ── 6. assert the host NodePort path (the actual first-run user contract) ─────────

# #1488: a brand-new user's very first command is `zynax --api-url
# http://localhost:8080 apply …`, which relies SOLELY on the kind extraPortMapping
# (host 8080 → nodePort 30080) hitting the api-gateway Service's pinned nodePort
# 30080. The e2e assertion scripts deliberately port-forward to 18080 (kube-proxy
# can reset a NodePort on multi-node clusters), so NO existing test ever exercised
# raw localhost:8080 — which is exactly why the stale-tgz nodePort regression
# shipped undetected. Verify the host path here so cluster-up.sh proves the claim
# it prints below, and any future Service/extraPortMapping/nodePort drift fails
# bring-up loudly instead of silently breaking the first-run experience.
HOST_GW_URL="${HOST_GW_URL:-http://localhost:8080}"
if command -v curl >/dev/null 2>&1; then
  log "asserting the host NodePort path is reachable: ${HOST_GW_URL}/healthz (the first-run user contract, #1488)…"
  host_ok=""
  for _ in $(seq 1 30); do
    code="$(curl -sS -o /dev/null -w '%{http_code}' --max-time 5 "${HOST_GW_URL}/healthz" 2>/dev/null || true)"
    if [[ "${code}" == "200" ]]; then
      host_ok="yes"
      break
    fi
    sleep 2
  done
  [[ -n "${host_ok}" ]] || die "api-gateway is NOT reachable on the host port (${HOST_GW_URL}/healthz never returned 200) — the kind extraPortMapping (host 8080 → nodePort 30080) and the Service nodePort (must be 30080) are out of sync (#1488). Check: kubectl -n ${NAMESPACE} get svc zynax-api-gateway -o jsonpath='{.spec.ports[0].nodePort}'"
  log "  ✓ host NodePort path verified — ${HOST_GW_URL} reaches the api-gateway (HTTP 200)."
else
  warn "curl not found — skipping the host NodePort reachability assertion (#1488)."
fi

log "cluster '${CLUSTER_NAME}' is up. api-gateway REST is reachable on host port 8080."
log "tear down with: scripts/e2e/cluster-down.sh"
