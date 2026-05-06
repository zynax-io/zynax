# SPDX-License-Identifier: Apache-2.0
# Zynax — Local Docker Compose Stack

Minimal local stack for end-to-end testing of the three implemented platform services.

## Quick start

```bash
make run-local    # build images + start all services
make logs-local   # tail all logs
make stop-local   # stop and remove containers
```

## Port map

| Host port | Service | Purpose |
|-----------|---------|---------|
| 7080 | api-gateway | HTTP REST — `export ZYNAX_API_URL=http://localhost:7080` |
| 7233 | Temporal | gRPC (worker/SDK connections) |
| 7088 | Temporal Web UI | Workflow inspection — http://localhost:7088 |
| 7422 | NATS | Client port (optional direct access) |

Internal-only (no host port):

| Container port | Service |
|---------------|---------|
| 50054 | workflow-compiler gRPC |
| 50055 | engine-adapter gRPC |

## Service startup order

```
postgres (healthy) → temporal (healthy) → engine-adapter (healthy)
                                        → workflow-compiler (healthy) → api-gateway
```

## Not included

The following services are unimplemented stubs awaiting M5 and are intentionally
omitted from this stack: `agent-registry`, `task-broker`, `memory-service`, `event-bus`.

## Verifying the stack

```bash
# All healthz probes
curl http://localhost:7080/healthz

# Apply an example workflow manifest
export ZYNAX_API_URL=http://localhost:7080
zynax apply spec/workflows/examples/code-review.yaml
```
