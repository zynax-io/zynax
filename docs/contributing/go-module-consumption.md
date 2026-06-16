<!-- SPDX-License-Identifier: Apache-2.0 -->
# Go module consumption path (pkg.go.dev)

This document records the verified import path for Zynax's published Go modules, how a
downstream consumer can `go get` them, the version pkg.go.dev / the Go module proxy serves
today, and the monorepo caveats that affect external consumption.

> Tracks epic [#1172](https://github.com/zynax-io/zynax/issues/1172) step Q.4 (issue
> [#1215](https://github.com/zynax-io/zynax/issues/1215), related [#582](https://github.com/zynax-io/zynax/issues/582)).

## Module paths

Zynax is a Go **multi-module monorepo**. Each service, adapter, lib, and the generated proto
package is its own module under the `github.com/zynax-io/zynax/` prefix. The module path is the
import path. Representative published modules:

| Module path | Subdir |
|-------------|--------|
| `github.com/zynax-io/zynax/services/api-gateway` | `services/api-gateway` |
| `github.com/zynax-io/zynax/protos/generated/go` | `protos/generated/go` |
| `github.com/zynax-io/zynax/libs/zynaxobs` | `libs/zynaxobs` |
| `github.com/zynax-io/zynax/libs/zynaxconfig` | `libs/zynaxconfig` |

The complete set of modules is the `use (...)` list in [`go.work`](../../go.work) at the repo
root. Each path maps 1:1 to a directory containing a `go.mod` whose `module` line is the path
above (for example `services/api-gateway/go.mod` declares
`module github.com/zynax-io/zynax/services/api-gateway`).

## Consume a module downstream

A consumer outside the monorepo imports a module by its full path and lets the Go module proxy
resolve it:

```bash
go get github.com/zynax-io/zynax/protos/generated/go@latest
```

```go
import enginev1 "github.com/zynax-io/zynax/protos/generated/go/zynax/engine/v1"
```

## Verified import availability (evidence)

Verified against the public Go module proxy (`proxy.golang.org`) on 2026-06-16, against
commit `6fee0b1` (HEAD of `main`):

- `GET /github.com/zynax-io/zynax/services/api-gateway/@latest` →
  `{"Version":"v0.0.0-20260616143554-6fee0b133bac", ... "Subdir":"services/api-gateway"}`
- `GET /github.com/zynax-io/zynax/protos/generated/go/@latest` →
  `{"Version":"v0.0.0-20260616143554-6fee0b133bac", ... "Subdir":"protos/generated/go"}`
- The pseudo-version `.info` endpoint serves the same metadata, confirming the proxy has the
  module zip cached and resolvable.

This proves both modules **are importable downstream**: the proxy resolves the path, identifies
the correct `Subdir`, and synthesizes a pseudo-version pinned to the commit. `go get <path>@latest`
or `@<commit>` succeeds from any consumer.

## Caveat 1 — no tagged semver; pkg.go.dev serves a pseudo-version

The repo's release tags are **repo-level** (`v0.4.0`, `v0.5.0`), not module-path-prefixed. Go
requires a submodule in a monorepo to be tagged as `<subdir>/vX.Y.Z`
(e.g. `services/api-gateway/v0.5.0`) for the proxy to serve that semver for that module.
Because no such tags exist, `@v/list` is empty for each module and the proxy serves only a
`v0.0.0-<timestamp>-<commit>` **pseudo-version**.

Consequence for consumers:
- `go get <path>@latest` resolves to a pseudo-version, not a clean semver.
- The pkg.go.dev page is populated on first proxy request; until a module is requested it may
  return 404. Requesting `@latest` (above) is what warms it.

To publish a real semver for a module, push a module-path-prefixed tag, e.g.:

```bash
git tag services/api-gateway/v0.6.0
git push origin services/api-gateway/v0.6.0
```

## Caveat 2 — `replace` directives and `go.work` (ADR-017)

Intra-repo modules use **local `replace` directives** (e.g. `services/api-gateway/go.mod` has
`replace github.com/zynax-io/zynax/protos/generated/go => ../../protos/generated/go`) and the
root [`go.work`](../../go.work) wires every module together for in-tree development. Per
[ADR-017](../adr/ADR-017-contract-test-isolation.md), `GOWORK=off` is required for per-module
`go` commands inside the repo.

These mechanisms are **monorepo-internal only** — they do not affect downstream consumers:
- The Go proxy resolves a module's dependencies from its published `go.mod`. A `replace` with a
  relative filesystem target (`../../...`) is ignored by consumers; Go resolves the real module
  by its path. The proxy serving the module above confirms this works.
- `go.work` is never consulted by an external consumer; it applies only within this repository.

A consumer therefore depends only on the module path and the version the proxy serves. No
`go.work` or in-repo `replace` knowledge is needed downstream.

## Re-running the verification

```bash
curl -s https://proxy.golang.org/github.com/zynax-io/zynax/services/api-gateway/@latest
curl -s https://proxy.golang.org/github.com/zynax-io/zynax/protos/generated/go/@latest
```

A non-empty JSON body with a `Version` field confirms the module is resolvable. A non-empty
`@v/list` body would indicate module-path-prefixed semver tags now exist.
