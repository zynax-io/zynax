# Dependency Strategy — Zynax

Policy for Go modules, Python packages, Docker images, and GitHub Actions.
Governs version pinning, security scanning, upgrade cadence, and footprint.

---

## Version Pinning Rules

**Go modules** — all `go.mod` files pin to exact versions (e.g. `v1.5.1`, never `latest`).
Indirect dependencies are pinned via `go.sum`. No floating ranges.

**Python packages** — `pyproject.toml` pins to exact minor versions in the `dev` dependency
group (e.g. `ruff>=0.4.0`). Production dependencies in `[project].dependencies` use minimum
lower bounds; the lock is enforced by `uv.lock` (not committed — generated fresh in CI).

**Docker images** — all `FROM` instructions in `Dockerfile.tools` use immutable image+tag pairs
(e.g. `golang:1.25-alpine`, `python:3.12-alpine`). `:latest` is never used anywhere.
Tool versions are documented in the pinning comment block at the top of each Dockerfile.
Renovate `# renovate:` annotations keep CI versions in sync with Dockerfile pins.

**GitHub Actions** — `uses:` references are pinned to a specific tag (e.g. `actions/checkout@v4`).
SHA pinning (`uses: actions/checkout@abc1234`) is enforced for third-party actions via the
Renovate `github-actions` group once Renovate is active (see issue #136).

---

## Security Scanning Requirements

All scans run inside the `zynax-tools` Docker image — no local tool installs required.

| Scope | Tool | Makefile target | Severity threshold |
|-------|------|-----------------|--------------------|
| Go modules | `govulncheck` | `make security-go` | Any finding fails |
| Python packages | `pip-audit` | `make security-agents` | Any finding fails |
| Python SAST | `bandit` | `make security-agents` | Medium+ fails |
| Combined audit | govulncheck + pip-audit | `make audit` | Any finding exits 1 |
| Full security scan | audit + bandit | `make security` | Any finding exits 1 |

`make audit` was added in PR #177 and runs in CI on every push to main and on every PR.
A medium-severity or higher finding blocks merge.

---

## Upgrade Cadence

**Go modules** — monthly review via `go list -m -u all`. Immediate upgrade when
`govulncheck` reports a CVE in a direct dependency. Security patches within 48 hours.

**Python packages** — monthly review via `pip-audit --dry-run`. Same 48-hour SLA
for CVEs. Dev-group upgrades (ruff, mypy, bandit) follow the monthly cadence.

**Docker base images** — aligned with upstream Alpine LTS minor releases. When
`golang:X.Y-alpine` or `python:3.12-alpine` drops a security advisory, upgrade
within one week. Pin comment block in `Dockerfile.tools` is updated together
with `ci.yml` env vars that must stay in sync.

**GitHub Actions** — track patch releases via Renovate (once active). For security
advisories, pin to a fixed SHA within 48 hours regardless of the Renovate cadence.

Renovate automation (issue #136) will generate grouped PRs for each scope on the
cadences above, reducing manual effort.

---

## Footprint Minimisation

All Docker images use Alpine variants — no Debian, no apt, no systemd:
- Go toolchain stage: `golang:1.25-alpine`
- Final image: `python:3.12-alpine`

Multi-stage builds copy only compiled binaries into the final image; the Go build
toolchain does not appear in the shipped layer (see `infra/docker/Dockerfile.tools`).

No `apt-get`, `yum`, or `apk add` of build toolchains in the final stage.
Only runtime system deps are allowed: `git curl ca-certificates libstdc++ libgcc`.

Python: no transitive dependency on heavy ML frameworks (PyTorch, TensorFlow) in
the core `zynax-sdk`. Agent examples may add framework deps in their own
`pyproject.toml` but must not propagate them into the SDK.

`uv` is used as the Python package manager — it resolves and installs deps in
isolated virtual environments without polluting the system Python.
