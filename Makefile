# SPDX-License-Identifier: Apache-2.0
# Zynax — Makefile
# Docker-first. Only prerequisite: Docker Desktop.
# Go platform services + Python/SDK agent layer.

.DEFAULT_GOAL := help
SHELL         := /bin/bash
GO_SERVICES   := $(shell grep -E '^\s+\./services/' go.work | sed 's|.*services/||' | tr -d '\t ' | sort)
# Auto-discovered from go.work — no manual update needed when adding a new adapter module under agents/adapters/.
GO_ADAPTERS   := $(shell grep -E '^\s+\./agents/adapters/' go.work | sed 's|.*agents/adapters/||' | tr -d '\t ' | sort)
# Services carrying Go benchmarks (EPIC R / #493). Add a service here when it gains a Benchmark* test.
BENCH_SERVICES := engine-adapter workflow-compiler
# Services carrying Go fuzz targets (EPIC R / #1210). Add a service here when it gains a Fuzz* test.
FUZZ_SERVICES := workflow-compiler
# Per-target fuzz campaign duration for `make fuzz` (override: make fuzz DURATION=60s).
DURATION ?= 60s
# Committed benchmark baseline — CI compares fresh runs against this with benchstat.
BENCH_BASELINE := tools/bench-baseline.txt
# Auto-discovered from agents/examples/*/pyproject.toml — no manual update needed when adding a new agent.
AGENTS        := $(shell find agents/examples -maxdepth 2 -name pyproject.toml -exec dirname {} \; 2>/dev/null | xargs -rI{} basename {} | sort)
COMPOSE_SERVICES := docker compose -f infra/docker-compose/docker-compose.services.yml
COMPOSE_TOOLS    := docker compose -f infra/docker-compose/docker-compose.tools.yml
# Override to use a local build: make TOOLS_IMAGE=zynax/tools:local build-tools
TOOLS_IMAGE   ?= ghcr.io/zynax-io/zynax/tools:latest
REGISTRY      := ghcr.io/zynax-io
GHCR_TOOLS    := ghcr.io/zynax-io/zynax/tools:latest
# Named-volume caches for the tools containers: Go module/build, uv, and
# golangci-lint caches survive the --rm container lifecycle. Without these,
# every tool-backed target recompiles from a cold cache (the caches lived and
# died inside each ephemeral container). Named volumes — not host bind mounts —
# keep the root-owned cache files off the host checkout. Paths match the ENV
# baked into Dockerfile.tools / .env.tools (GOPATH=/root/go,
# GOCACHE=/root/.cache/go-build, UV_CACHE_DIR=/root/.cache/uv); golangci-lint
# defaults to $HOME/.cache/golangci-lint. Reclaim disk: make clean-caches.
TOOLS_CACHES  := -v zynax-gomod:/root/go/pkg/mod -v zynax-gobuild:/root/.cache/go-build \
                   -v zynax-uv:/root/.cache/uv -v zynax-golangci:/root/.cache/golangci-lint
TOOLS_RUN     := docker run --rm -v ".:/workspace" -w /workspace --env-file infra/docker/.env.tools \
                   -e GIT_CONFIG_COUNT=1 -e GIT_CONFIG_KEY_0=safe.directory -e GIT_CONFIG_VALUE_0=/workspace \
                   $(TOOLS_CACHES) \
                   $(TOOLS_IMAGE)

.PHONY: help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: bootstrap check-docker build-tools bump-ci-runner
bootstrap: ensure-tools ## ★ Run once after clone — pulls tools image from GHCR and installs pre-commit hooks
	@if command -v pre-commit >/dev/null 2>&1; then \
	  pre-commit install && echo "✅ pre-commit hooks installed"; \
	else \
	  echo "⚠️  pre-commit not found — skipping hook install (pip install pre-commit)"; \
	fi
	@echo "✅ Done. make lint → lint all, make test → full test suite"

check-docker:
	@docker info >/dev/null 2>&1 || (echo "❌ Docker not running" && exit 1)
	@echo "✅ Docker $(shell docker version --format '{{.Server.Version}}')"

build-tools: check-docker ## Build zynax/tools:local from source — use when editing Dockerfile.tools
	docker build -f infra/docker/Dockerfile.tools -t $(TOOLS_IMAGE) .
	@echo "✅ Tools image: $(TOOLS_IMAGE)"

bump-ci-runner: ## Pin a new ci-runner digest in images.yaml and re-stamp consumers
	@[ -n "$(NEW_DIGEST)" ] || (echo "❌ Usage: make bump-ci-runner NEW_DIGEST=sha256:<64-hex>"; exit 1)
	cd cmd/zynax-ci && GOWORK=off go run . bump-runner "$(NEW_DIGEST)" --root "$(CURDIR)"

pull-tools: check-docker ## Pull tools image from GHCR (authenticates via `gh auth token`)
	@echo "🔐 Authenticating to GHCR via gh..."
	@gh auth token | docker login ghcr.io -u $$(gh api user --jq .login) --password-stdin 2>&1
	docker pull $(GHCR_TOOLS)
	@echo "✅ Pulled $(GHCR_TOOLS)"
	@echo "   Run targets with: make TOOLS_IMAGE=$(GHCR_TOOLS) <target>"

# Internal prereq used by every tool-backed target.
# - local image (zynax/tools:local): builds from Dockerfile.tools
# - remote image (default GHCR): pulls only when not already cached locally
.PHONY: ensure-tools
ensure-tools: check-docker
ifeq ($(TOOLS_IMAGE),zynax/tools:local)
	$(MAKE) build-tools
else
	@docker image inspect $(TOOLS_IMAGE) >/dev/null 2>&1 \
		|| (echo "⬇️  Pulling $(TOOLS_IMAGE)..." && docker pull $(TOOLS_IMAGE))
endif

# ── Local environment (ADR-041: kind is the one runtime — see `make demo`) ──
.PHONY: install-cli install-ci-tools sync-images check-images

# ── One-command kind demo (ADR-041 — kind is the unified runtime) ────────────
# `make demo` creates a single-node kind cluster, side-loads the local images,
# installs the zynax-umbrella Helm chart, waits for every Deployment's rollout,
# runs the hero workflow against the gateway, and prints "Platform ready" with
# the next command — ONE command, on the same charts that run in production.
# `make kind-up` / `make kind-down` front the cluster lifecycle directly.
.PHONY: demo kind-up kind-down
KIND_DEMO     := scripts/demo/kind-demo.sh
CLUSTER_UP    := scripts/e2e/cluster-up.sh
CLUSTER_DOWN  := scripts/e2e/cluster-down.sh
# Stack profile for the kind targets (ADR-041): `full` (default, prod-mirroring)
# or `lite` (lean laptop — 1 in-memory dev Temporal, no event-bus/NATS/memory-
# service). Usage: `make demo PROFILE=lite` / `make kind-up PROFILE=lite`.
PROFILE      ?= full
# Workflow engine the kind demo deploys (#1500, the engine-portability wedge —
# #1370 / ADR-041): `temporal` (default) or `argo`. The SAME workflow manifest
# runs unchanged on either — selection flows through umbrella values only
# (values-e2e-argo.yaml), never the workflow (ADR-015). `ENGINE` is the clean
# alias; the underlying `E2E_ENGINE` env var (read by cluster-up.sh) still works
# as an override, so `E2E_ENGINE=argo make demo` and `make demo ENGINE=argo` are
# equivalent. Usage: `make demo ENGINE=argo` / `make kind-up ENGINE=argo`.
ENGINE       ?= $(or $(E2E_ENGINE),temporal)
DEMO_WORKFLOW ?= spec/workflows/examples/code-review-ollama.yaml
ZYNAX        ?= zynax
# Optional: run a declarative scenario manifest set instead of the single hero
# workflow — e.g. `make demo SCENARIO=code-review`. When SCENARIO is set, the
# whole directory spec/scenarios/$(SCENARIO) is applied (AgentDef then Workflow,
# in the index's apply_order) over the existing /api/v1/apply REST path.
SCENARIO     ?=
# Optional: review a real GitHub PR's diff instead of the canned workflow —
# `make demo PR=1446` fetches the diff (read-only via `gh pr diff`) and reviews it.
PR           ?=
DEMO_PR_FILE := /tmp/zynax-demo-pr-review.yaml
DEMO_TARGET   = $(if $(PR),$(DEMO_PR_FILE),$(if $(SCENARIO),spec/scenarios/$(SCENARIO),$(DEMO_WORKFLOW)))
# Optional: STREAM=1 prints EVERY step's output live via `zynax logs --follow`
# instead of just the final result — useful for multi-step (data-flow) workflows.
# Note: streaming is per-step, not per-token — the engine polls the broker for
# each capability's final result, so a step's output appears once it completes.
STREAM       ?=
# Demo LLM model — read from the Ollama overlay config so the pre-flight check and
# the config never drift. `make demo` ensures this is pulled on the host (the
# ollama container reuses host models read-only), otherwise the codereview 404s.
DEMO_MODEL   := $(shell awk '/^[[:space:]]*model:/{print $$2; exit}' infra/ollama/llm-adapter.config.yaml)
# Services the demo needs; `up --wait` boots their depends_on closure (workflow-compiler,
# engine-adapter, task-broker, agent-registry, temporal, event-bus, nats, postgres*, ollama).
# Deliberately EXCLUDES the git/ci/langgraph adapters — they require GITHUB_TOKEN or are unused, and
# `up --wait` (no service list) would otherwise gate the whole demo on their health — and the
# standalone Temporal UI.
DEMO_SERVICES := api-gateway llm-adapter

demo: check-docker ## ★ One command (kind): create cluster → load images → install umbrella → wait rollout → "Platform ready" + run hero workflow (PROFILE=lite for the lean stack; ENGINE=argo for the Argo wedge)
	@echo "🧭 Zynax demo on kind (ADR-041, profile: $(PROFILE), engine: $(ENGINE)) — one command, prod-mirroring charts."
	PROFILE=$(PROFILE) E2E_ENGINE=$(ENGINE) $(KIND_DEMO)

kind-up: check-docker ## Create the kind cluster + install the zynax-umbrella chart (wraps scripts/e2e/cluster-up.sh, loads local images; PROFILE=lite for the lean stack; ENGINE=argo for Argo)
	@echo "☸️  Bringing up the kind cluster + Zynax umbrella (profile: $(PROFILE), engine: $(ENGINE), loads local images)..."
	KIND_LOAD_IMAGES=1 PROFILE=$(PROFILE) E2E_ENGINE=$(ENGINE) $(CLUSTER_UP)

kind-down: ## Tear down the kind cluster (wraps scripts/e2e/cluster-down.sh)
	@echo "🧹 Tearing down the kind cluster..."
	$(CLUSTER_DOWN)

install-cli: ## Build and install zynax CLI to ~/bin/zynax (requires Go 1.26.3)
	cd cmd/zynax && GOWORK=off go build -trimpath -o ~/bin/zynax .
	@echo "✅ zynax installed → ~/bin/zynax  (ensure ~/bin is on your PATH)"

install-ci-tools: ## Build and install zynax-ci toolchain to ~/bin/zynax-ci (requires Go 1.26.3)
	cd cmd/zynax-ci && GOWORK=off go build -trimpath -o ~/bin/zynax-ci .
	@echo "✅ zynax-ci installed → ~/bin/zynax-ci  (ensure ~/bin is on your PATH)"

sync-images: ## Stamp all consumer files with digests from images/images.yaml
	cd cmd/zynax-ci && GOWORK=off go run . images sync --root "$(CURDIR)"

check-images: ## Verify all consumer files match images/images.yaml digests
	cd cmd/zynax-ci && GOWORK=off go run . images check --root "$(CURDIR)"

# ── Local CI gate ──────────────────────────────────────────────────────────
.PHONY: ci
ci: lint test security gitleaks ## ★ Full local CI gate — lint → test (incl. validate-spec) → security → secret scan
	@echo "✅ Local CI gate passed — ready to push"

# ── Lint ───────────────────────────────────────────────────────────────────
.PHONY: lint lint-go lint-go-adapters lint-agents lint-go-svc lint-agent lint-fix
lint: lint-protos lint-go lint-go-adapters lint-agents ## Lint everything (proto + Go services + Go adapters + Python)

lint-go: ensure-tools ## Lint all Go platform services with golangci-lint
	@for svc in $(GO_SERVICES); do \
		echo "🔍 $$svc"; \
		$(TOOLS_RUN) sh -c "cd services/$$svc && golangci-lint run ./... --config ../../tools/golangci-lint.yml"; \
	done && echo "✅ Go lint passed"

lint-go-adapters: ensure-tools ## Lint all Go adapter modules with golangci-lint
	@for adp in $(GO_ADAPTERS); do \
		echo "🔍 adapters/$$adp"; \
		$(TOOLS_RUN) sh -c "cd agents/adapters/$$adp && golangci-lint run ./... --config ../../../tools/golangci-lint.yml"; \
	done && echo "✅ Go adapter lint passed"

lint-go-svc: ensure-tools ## Lint one Go service: make lint-go-svc SVC=agent-registry
	$(TOOLS_RUN) sh -c "cd services/$(SVC) && golangci-lint run ./... --config ../../tools/golangci-lint.yml"

lint-agents: ensure-tools ## Lint SDK + all Python agents (ruff + mypy)
	$(TOOLS_RUN) sh -c "cd agents/sdk && uv run ruff check src/ && uv run mypy src/ --strict"
	@for a in $(AGENTS); do [ -f "agents/examples/$$a/pyproject.toml" ] && \
		$(TOOLS_RUN) sh -c "cd agents/examples/$$a && uv run ruff check src/ tests/ && uv run mypy src/ --strict" || true; done
	@echo "✅ Python lint passed"

lint-agent: ensure-tools ## Lint one agent: make lint-agent AGENT=summarizer
	$(TOOLS_RUN) sh -c "cd agents/examples/$(AGENT) && uv run ruff check src/ tests/ && uv run mypy src/ --strict"

lint-fix: ensure-tools ## Auto-fix Python agent lint errors
	@for a in $(AGENTS); do [ -f "agents/examples/$$a/pyproject.toml" ] && \
		$(TOOLS_RUN) sh -c "cd agents/examples/$$a && uv run ruff check --fix src/ tests/ && uv run ruff format src/ tests/" || true; done

# ── Tests ──────────────────────────────────────────────────────────────────
.PHONY: test test-unit test-unit-go test-unit-adapters test-unit-svc test-unit-agents test-unit-agent test-bdd test-coverage test-coverage-adapters test-integration bench fuzz
test: validate-spec test-unit test-bdd test-coverage test-coverage-adapters ## ★ Full local test suite — mirrors CI (spec + Go + Python + BDD + coverage gate)
test-unit: test-unit-go test-unit-adapters test-unit-agents ## All unit tests (Go services + Go adapters + Python)

test-unit-go: ensure-tools ## Go unit tests for all services (excludes //go:build integration files)
	@for svc in $(GO_SERVICES); do \
		if [ -f "services/$$svc/go.mod" ]; then \
			echo "🧪 $$svc"; \
			$(TOOLS_RUN) sh -c "cd services/$$svc && GOWORK=off go test -tags=\"\" ./... -v -timeout 60s -count=1" || exit 1; \
		fi; \
	done && echo "✅ Go tests passed"

test-unit-adapters: ensure-tools ## Go unit tests for all adapter modules
	@for adp in $(GO_ADAPTERS); do \
		if [ -f "agents/adapters/$$adp/go.mod" ]; then \
			echo "🧪 adapters/$$adp"; \
			$(TOOLS_RUN) sh -c "cd agents/adapters/$$adp && GOWORK=off go test -tags=\"\" ./... -v -timeout 60s -count=1" || exit 1; \
		fi; \
	done && echo "✅ Go adapter tests passed"

test-unit-svc: ensure-tools ## Go tests for one service: make test-unit-svc SVC=workflow-compiler
	$(TOOLS_RUN) sh -c "cd services/$(SVC) && GOWORK=off go test -tags=\"\" ./... -v -timeout 60s"

test-coverage: ensure-tools ## Domain coverage gate — ≥90% on internal/domain/ for every Go service
	@failed=false; \
	for svc in $(GO_SERVICES); do \
		if [ -f "services/$$svc/go.mod" ] && [ -d "services/$$svc/internal/domain" ]; then \
			$(TOOLS_RUN) sh -c "cd services/$$svc && GOWORK=off go test -tags=\"\" ./internal/domain/... -coverprofile=domain-coverage.out -covermode=atomic -count=1 2>/dev/null"; \
			total=$$($(TOOLS_RUN) sh -c "cd services/$$svc && GOWORK=off go tool cover -func=domain-coverage.out | grep '^total:' | awk '{print \$$3}' | tr -d '%'" 2>/dev/null); \
			if [ -z "$$total" ]; then echo "  ⚠  services/$$svc: no domain coverage data"; continue; fi; \
			funcs=$$($(TOOLS_RUN) sh -c "cd services/$$svc && GOWORK=off go tool cover -func=domain-coverage.out | grep -vc '^total:'" 2>/dev/null); \
			if [ "$$funcs" = "0" ]; then printf "  %-35s %s\n" "services/$$svc" "no coverable statements — exempt"; continue; fi; \
			printf "  %-35s %s%%\n" "services/$$svc" "$$total"; \
			if awk "BEGIN{exit !($$total < 90)}"; then \
				echo "  ❌ $$total%% < 90%% — coverage gate failed for services/$$svc"; \
				failed=true; \
			fi; \
		fi; \
	done; \
	$$failed && exit 1 || echo "✅ Domain coverage gate passed (all services ≥90%%)"

test-coverage-adapters: ensure-tools ## Total coverage gate — ≥80% across all packages for every Go adapter
	@failed=false; \
	for adp in $(GO_ADAPTERS); do \
		if [ -f "agents/adapters/$$adp/go.mod" ]; then \
			$(TOOLS_RUN) sh -c "cd agents/adapters/$$adp && GOWORK=off go test -tags=\"\" ./... -coverprofile=coverage.out -covermode=atomic -count=1 2>/dev/null"; \
			total=$$($(TOOLS_RUN) sh -c "cd agents/adapters/$$adp && GOWORK=off go tool cover -func=coverage.out | grep '^total:' | awk '{print \$$3}' | tr -d '%'" 2>/dev/null); \
			if [ -z "$$total" ]; then echo "  ⚠  agents/adapters/$$adp: no coverage data"; continue; fi; \
			printf "  %-40s %s%%\n" "agents/adapters/$$adp" "$$total"; \
			if awk "BEGIN{exit !($$total < 80)}"; then \
				echo "  ❌ $$total%% < 80%% — coverage gate failed for agents/adapters/$$adp"; \
				failed=true; \
			fi; \
		fi; \
	done; \
	$$failed && exit 1 || echo "✅ Adapter coverage gate passed (all adapters ≥80%%)"

test-bdd: ensure-tools ## Godog BDD contract tests for all protos/tests/ packages
	$(TOOLS_RUN) sh -c "cd protos/tests && GOWORK=off go test ./... -v -timeout 120s"
	@echo "✅ BDD contract tests passed"

test-unit-agents: ensure-tools ## pytest-bdd for SDK + all Python agents
	$(TOOLS_RUN) sh -c "cd agents/sdk && uv run pytest tests/ --cov=src --cov-fail-under=90 -v"
	@for a in $(AGENTS); do [ -f "agents/examples/$$a/pyproject.toml" ] && \
		$(TOOLS_RUN) sh -c "cd agents/examples/$$a && uv run pytest tests/ --cov=src --cov-fail-under=90 -v" || true; done
	@echo "✅ Python tests passed"

test-unit-agent: ensure-tools ## pytest for one agent: make test-unit-agent AGENT=summarizer
	$(TOOLS_RUN) sh -c "cd agents/examples/$(AGENT) && uv run pytest tests/ -v"

test-integration: check-docker ## Integration tests (//go:build integration files) — requires Docker Compose stack
	@count=$$(grep -rl "//go:build integration" services/ 2>/dev/null | wc -l); \
	if [ "$$count" -eq 0 ]; then \
		echo "ℹ️  No integration test files found — skipping stack startup."; \
		echo "   Tag test files with //go:build integration to include them here."; \
		exit 0; \
	fi
	@echo "Starting testing backing services..."
	$(COMPOSE_SERVICES) --profile testing up -d
	@echo "Waiting for services to be healthy (up to 60s)..."
	@timeout 60 sh -c \
		'until docker compose -f infra/docker-compose/docker-compose.services.yml ps --format json 2>/dev/null \
		  | python3 -c "import sys,json; data=sys.stdin.read(); rows=json.loads(data) if data.strip().startswith(\"[\") else [json.loads(l) for l in data.strip().splitlines() if l]; exit(0 if all(r.get(\"Health\")==\"healthy\" for r in rows if r.get(\"Health\")) and len(rows)>0 else 1)" \
		  2>/dev/null; do sleep 2; done' || true
	@for svc in $(GO_SERVICES); do \
		if grep -rl "//go:build integration" "services/$$svc/" 2>/dev/null | grep -q .; then \
			echo "── integration: services/$$svc"; \
			$(TOOLS_RUN) sh -c "cd services/$$svc && GOWORK=off go test -tags=integration ./... -v -timeout 120s" || exit 1; \
		fi; \
	done
	@echo "Stopping testing backing services..."
	$(COMPOSE_SERVICES) --profile testing down --remove-orphans
	@echo "✅ Integration tests passed"

bench: ensure-tools ## Run domain benchmarks and regenerate $(BENCH_BASELINE) (-bench=. -benchmem -count=3)
	@echo "📊 Running benchmarks → $(BENCH_BASELINE)"
	@: > "$(BENCH_BASELINE)"
	@for svc in $(BENCH_SERVICES); do \
		echo "── bench: services/$$svc"; \
		$(TOOLS_RUN) sh -c "cd services/$$svc && GOWORK=off go test ./internal/domain/... -run='^\$$' -bench=. -benchmem -count=3" \
			| tee -a "$(BENCH_BASELINE)" || exit 1; \
	done
	@echo "✅ Benchmarks complete — baseline written to $(BENCH_BASELINE)"

fuzz: ensure-tools ## Run domain fuzz campaigns locally (override duration: make fuzz DURATION=60s)
	@echo "🎲 Running fuzz campaigns (DURATION=$(DURATION) per target)"
	@for svc in $(FUZZ_SERVICES); do \
		echo "── fuzz: services/$$svc"; \
		$(TOOLS_RUN) sh -c "cd services/$$svc && \
			for fn in \$$(GOWORK=off go test ./internal/domain/... -list='^Fuzz' 2>/dev/null | grep '^Fuzz'); do \
				echo \"   campaign: \$$fn\"; \
				GOWORK=off go test ./internal/domain/... -run='^\$$' -fuzz=\"^\$$fn\$$\" -fuzztime=$(DURATION) || exit 1; \
			done" || exit 1; \
	done
	@echo "✅ Fuzz campaigns complete — no crashers found"

# ── Proto generation + lint ────────────────────────────────────────────────
.PHONY: generate-protos go-generate lint-protos
generate-protos: ensure-tools ## Generate Go + Python stubs from .proto files
	$(TOOLS_RUN) sh -c "cd protos && buf generate --template buf.gen.yaml"
	@echo "✅ Stubs in protos/generated/go/ and protos/generated/python/ — commit them"

go-generate: ## Re-run //go:generate directives (requires buf locally; same as generate-protos but no Docker)
	@echo "Running go generate in protos/generated/go — requires buf in PATH"
	cd protos/generated/go && GOWORK=off go generate ./...
	@echo "✅ go generate complete — commit updated stubs"

lint-protos: ensure-tools ## buf lint + format check on all proto files
	$(TOOLS_RUN) sh -c "cd protos && buf lint && buf format --diff --exit-code"
	@echo "✅ Proto lint passed"

# ── Security ───────────────────────────────────────────────────────────────
.PHONY: security security-go security-go-adapters security-agents scan-image sbom audit gitleaks
security: security-go security-go-adapters security-agents ## Full security scan (govulncheck + bandit + pip-audit + trivy)

scan-image: ## Scan one service container image for CVEs: make scan-image SVC=agent-registry
	docker build -f infra/docker/Dockerfile.service --build-arg SVC=$(SVC) -t zynax/$(SVC):scan .
	trivy image --exit-code 1 --severity HIGH,CRITICAL --ignorefile .trivyignore zynax/$(SVC):scan
	docker rmi zynax/$(SVC):scan

sbom: ensure-tools ## Generate CycloneDX SBOM for one service image: make sbom SVC=api-gateway
	docker build -f infra/docker/Dockerfile.service --build-arg SVC=$(SVC) -t $(REGISTRY)/zynax-$(SVC):local .
	docker run --rm \
	  -v /var/run/docker.sock:/var/run/docker.sock \
	  -v "$(CURDIR):/workspace" -w /workspace \
	  $(TOOLS_IMAGE) \
	  syft $(REGISTRY)/zynax-$(SVC):local -o cyclonedx-json --file sbom-$(SVC).json
	@echo "✅ SBOM written to sbom-$(SVC).json"

gitleaks: ensure-tools ## Scan working tree for secrets/PII — mirrors the ci.yml Secret scan gate (no git history)
	$(TOOLS_RUN) gitleaks detect --no-git --source . \
	  --config tools/gitleaks-ai-context.toml \
	  --baseline-path tools/gitleaks-baseline.json \
	  --verbose

security-go: ensure-tools ## govulncheck on all Go services
	@for svc in $(GO_SERVICES); do $(TOOLS_RUN) sh -c "cd services/$$svc && govulncheck ./..."; done

security-go-adapters: ensure-tools ## govulncheck on all Go adapter modules
	@for adp in $(GO_ADAPTERS); do \
		$(TOOLS_RUN) sh -c "cd agents/adapters/$$adp && GOWORK=off govulncheck ./..."; \
	done && echo "✅ Adapter security scan passed"

security-agents: ensure-tools ## bandit + pip-audit on SDK + all agents
	@$(TOOLS_RUN) sh -c "cd agents/sdk && uv run bandit -r src/ -ll && uv run pip-audit --ignore-vuln PYSEC-2026-196"
	@for a in $(AGENTS); do [ -f "agents/examples/$$a/pyproject.toml" ] && \
		$(TOOLS_RUN) sh -c "cd agents/examples/$$a && uv run bandit -r src/ -ll && uv run pip-audit" || true; done

audit: ensure-tools ## Dependency vulnerability audit (govulncheck + pip-audit); exits 1 on any finding
	@failed=false; \
	for svc in $(GO_SERVICES); do \
		if [ -f "services/$$svc/go.mod" ]; then \
			echo "🔍 govulncheck: $$svc"; \
			$(TOOLS_RUN) sh -c "cd services/$$svc && GOWORK=off govulncheck ./..." || failed=true; \
		fi; \
	done; \
	for a in $(AGENTS); do \
		if [ -f "agents/examples/$$a/pyproject.toml" ]; then \
			echo "🔍 pip-audit: $$a"; \
			$(TOOLS_RUN) sh -c "cd agents/examples/$$a && uv run pip-audit" || failed=true; \
		fi; \
	done; \
	$$failed && exit 1 || echo "✅ Audit passed — no known vulnerabilities"

# ── Build images ───────────────────────────────────────────────────────────
.PHONY: build build-svc build-agent
# Service images are dual-tagged: <registry>/zynax-<svc>:local (scan-image /
# sbom compatibility) AND <registry>/zynax/<svc>:main — the exact name the kind
# side-load lane expects (cluster-up.sh KIND_LOAD_REGISTRY/KIND_LOAD_TAG
# defaults, values-e2e.yaml, echo-worker.yaml). Without the second tag,
# `make demo` / `make kind-up` never find local builds and pull :main from
# GHCR. A locally built zynax/<svc>:main shadows GHCR's inside the cluster —
# that is the side-load design (IfNotPresent); `docker pull` refreshes it.
build: check-docker ## Build all Docker images (side-load-ready for make demo / kind-up)
	@for svc in $(GO_SERVICES); do docker build -f infra/docker/Dockerfile.service --build-arg SVC=$$svc -t $(REGISTRY)/zynax-$$svc:local -t $(REGISTRY)/zynax/$$svc:main .; done
	@docker build -f agents/adapters/langgraph/Dockerfile -t $(REGISTRY)/zynax/langgraph-adapter:main .
	@for a in $(AGENTS); do [ -f "agents/examples/$$a/Dockerfile" ] && docker build agents/examples/$$a -t $(REGISTRY)/zynax-agent-$$a:local || true; done

build-svc: ## Build one service image: make build-svc SVC=agent-registry
	docker build -f infra/docker/Dockerfile.service --build-arg SVC=$(SVC) -t $(REGISTRY)/zynax-$(SVC):local -t $(REGISTRY)/zynax/$(SVC):main .

build-agent: ## Build one agent image: make build-agent AGENT=summarizer
	docker build agents/examples/$(AGENT) -t $(REGISTRY)/zynax-agent-$(AGENT):local

# ── Cleanup ────────────────────────────────────────────────────────────────
.PHONY: clean clean-all clean-tools clean-caches
clean:      ## Remove cache files
	@find . -type d -name "__pycache__" -exec rm -rf {} + 2>/dev/null; true
	@find . -name "*.pyc" -delete 2>/dev/null; true && echo "✅ Clean"
clean-tools: ## Remove tools image
	docker rmi $(TOOLS_IMAGE) 2>/dev/null || true
clean-caches: ## Remove the named Docker volumes holding the Go/uv tool caches
	@docker volume rm -f zynax-gomod zynax-gobuild zynax-uv zynax-golangci >/dev/null 2>&1 || true
	@echo "✅ Tool cache volumes removed"
clean-all: clean dev-down clean-tools clean-caches ## ⚠ Remove everything

# ── Spec validation ───────────────────────────────────────────────────────────
.PHONY: validate-spec validate-asyncapi validate-workflow-schema validate-agent-def-schema validate-policy-schema validate-scenario-schema check-expert-mapping validate-canvas validate-milestone-state dry-run

validate-milestone-state: ## Validate state/milestone.yaml against state/milestone.schema.json
	cd cmd/zynax-ci && GOWORK=off go run . validate milestone --root "$(CURDIR)"
	@echo "✅ Milestone state valid"

validate-spec: validate-asyncapi validate-capability-schemas validate-workflow-schema validate-agent-def-schema validate-policy-schema validate-scenario-schema check-expert-mapping ## Validate all specs (AsyncAPI + capability schemas + workflow + agent-def + policy + scenario manifests + expert mapping)

validate-canvas: ensure-tools ## Validate REASONS Canvas files under docs/spdd/ (SPDD — ADR-019)
	$(TOOLS_RUN) zynax-ci validate canvas docs/spdd/
	@echo "✅ Canvas validation passed"

validate-capability-schemas: ensure-tools ## Validate capability declarations in spec/ against capability.schema.json
	$(TOOLS_RUN) zynax-ci validate capabilities spec/workflows/examples/ --schema-dir spec/schemas
	@echo "✅ Capability schemas valid"

validate-workflow-schema: ensure-tools ## Validate Workflow manifests (spec examples + templates + scenarios + automation/workflows) against workflow.schema.json
	$(TOOLS_RUN) zynax-ci validate workflows spec/workflows/examples/ --schema-dir spec/schemas
	$(TOOLS_RUN) zynax-ci validate workflows spec/templates/workflow/ --schema-dir spec/schemas
	$(TOOLS_RUN) zynax-ci validate workflows spec/scenarios/code-review/ --schema-dir spec/schemas
	$(TOOLS_RUN) zynax-ci validate workflows automation/workflows/ --schema-dir spec/schemas
	@echo "✅ Workflow schemas valid"

validate-agent-def-schema: ensure-tools ## Validate AgentDef manifests (spec examples + templates + automation/workflows) against agent-def.schema.json
	$(TOOLS_RUN) zynax-ci validate agent-defs spec/workflows/examples/ --schema-dir spec/schemas
	$(TOOLS_RUN) zynax-ci validate agent-defs spec/templates/task/ --schema-dir spec/schemas
	$(TOOLS_RUN) zynax-ci validate agent-defs spec/templates/expert/ --schema-dir spec/schemas
	$(TOOLS_RUN) zynax-ci validate agent-defs spec/scenarios/code-review/ --schema-dir spec/schemas
	$(TOOLS_RUN) zynax-ci validate agent-defs automation/workflows/ --schema-dir spec/schemas
	$(TOOLS_RUN) zynax-ci validate agent-defs automation/workflows/experts/ --schema-dir spec/schemas
	@echo "✅ AgentDef schemas valid"

validate-policy-schema: ensure-tools ## Validate Policy manifests in spec/workflows/examples/ against policy.schema.json
	$(TOOLS_RUN) zynax-ci validate policies spec/workflows/examples/ --schema-dir spec/schemas
	@echo "✅ Policy schemas valid"

validate-scenario-schema: ensure-tools ## Validate Scenario index files in spec/scenarios/ against scenario.schema.json
	$(TOOLS_RUN) zynax-ci validate scenarios spec/scenarios/code-review/ --schema-dir spec/schemas
	@echo "✅ Scenario schemas valid"

check-expert-mapping: ## Drift guard: authoring <-> runtime expert mapping (ADR-033)
	cd cmd/zynax-ci && GOWORK=off go run . check expert-mapping --root "$(CURDIR)"

validate-asyncapi: ## Validate spec/asyncapi/zynax-events.yaml via AsyncAPI CLI (Docker)
	# renovate: datasource=docker depName=asyncapi/cli
	# asyncapi-latest-version: suppressed — spec stays at 2.6.0; upgrading to 3.x
	# requires a breaking structural rewrite. Revisit when 3.x tooling matures.
	docker run --rm \
		-v "$(PWD)/spec":/spec \
		asyncapi/cli:6.0.0 \
		validate /spec/asyncapi/zynax-events.yaml \
		--suppressWarnings asyncapi-latest-version
	@echo "✅ AsyncAPI spec valid"

dry-run: ensure-tools ## Dry-run a workflow: make dry-run FILE=spec/workflows/examples/code-review.yaml
	@test -n "$(FILE)" || (echo "Usage: make dry-run FILE=<path>" && exit 1)
	$(TOOLS_RUN) sh -c "keel apply --dry-run $(FILE)"

# ── AI knowledge base ──────────────────────────────────────────────────────────
.PHONY: preview-kb-changes
preview-kb-changes: ## Preview KB additions before pushing (local dry-run, mirrors CI kb-preview.yml)
	@git diff --diff-filter=AM HEAD -- \
		CLAUDE.md AGENTS.md '*/AGENTS.md' \
		'docs/ai-assistant-setup.md' 'docs/knowledge-base-policy.md' \
		'.ai/**' '.claude/**' \
		| grep '^+' | grep -v '^+++' | sed 's/^+//' \
	  && echo "── review the output above against docs/knowledge-base-policy.md before pushing"
