# SPDX-License-Identifier: Apache-2.0
# Zynax — Makefile
# Docker-first. Only prerequisite: Docker Desktop.
# Go platform services + Python/SDK agent layer.

.DEFAULT_GOAL := help
SHELL         := /bin/bash
GO_SERVICES   := agent-registry task-broker memory-service event-bus api-gateway workflow-compiler engine-adapter
AGENTS        := summarizer researcher calculator
COMPOSE       := docker compose -f infra/docker/docker-compose.yml
COMPOSE_TOOLS := docker compose -f infra/docker/docker-compose.tools.yml
# Override to skip the local build: make TOOLS_IMAGE=ghcr.io/zynax-io/zynax/tools:latest <target>
TOOLS_IMAGE   ?= zynax-tools:local
REGISTRY      := ghcr.io/zynax-io
GHCR_TOOLS    := ghcr.io/zynax-io/zynax/tools:latest
TOOLS_RUN     := docker run --rm -v "$(PWD)":/workspace -w /workspace --env-file infra/docker/.env.tools $(TOOLS_IMAGE)

.PHONY: help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: bootstrap check-docker build-tools
bootstrap: check-docker build-tools ## ★ Run once after clone — builds tools image and installs pre-commit hooks
	@if command -v pre-commit >/dev/null 2>&1; then \
	  pre-commit install && echo "✅ pre-commit hooks installed"; \
	else \
	  echo "⚠️  pre-commit not found — skipping hook install (pip install pre-commit)"; \
	fi
	@echo "✅ Done. make lint → lint all, make test → full test suite"

check-docker:
	@docker info >/dev/null 2>&1 || (echo "❌ Docker not running" && exit 1)
	@echo "✅ Docker $(shell docker version --format '{{.Server.Version}}')"

build-tools: check-docker ## Build zynax-tools:local (golang:1.22-alpine + python:3.12-alpine + all CI tools)
	docker build -f infra/docker/Dockerfile.tools -t $(TOOLS_IMAGE) .
	@echo "✅ Tools image: $(TOOLS_IMAGE)"

pull-tools: check-docker ## Pull tools image from GHCR (authenticates via `gh auth token`)
	@echo "🔐 Authenticating to GHCR via gh..."
	@gh auth token | docker login ghcr.io -u $$(gh api user --jq .login) --password-stdin 2>&1
	docker pull $(GHCR_TOOLS)
	@echo "✅ Pulled $(GHCR_TOOLS)"
	@echo "   Run targets with: make TOOLS_IMAGE=$(GHCR_TOOLS) <target>"

# Internal: build local image only when TOOLS_IMAGE is the local default.
# Targets that use this as a prereq skip the Docker build when the caller
# overrides TOOLS_IMAGE (e.g. TOOLS_IMAGE=ghcr.io/zynax-io/zynax/tools:latest).
.PHONY: ensure-tools
ensure-tools: check-docker
ifeq ($(TOOLS_IMAGE),zynax-tools:local)
	$(MAKE) build-tools
endif

# ── Local environment ──────────────────────────────────────────────────────
.PHONY: dev-up dev-down dev-logs dev-ps dev-reset dev-restart
dev-up: check-docker ## Start full local stack (platform + agents + observability)
	$(COMPOSE) up -d --build
	@echo "" && echo "  API Gateway → http://localhost:8080  |  Grafana → http://localhost:3000  |  Jaeger → http://localhost:16686"

dev-down:   ## Stop all services
	$(COMPOSE) down
dev-logs:   ## Tail all logs
	$(COMPOSE) logs -f
dev-ps:     ## Show service status
	$(COMPOSE) ps
dev-reset:  ## ⚠ Destroy data and restart
	@read -p "Delete all volumes? [y/N] " ans && [ "$$ans" = y ]
	$(COMPOSE) down -v --remove-orphans && $(MAKE) dev-up
dev-restart: ## Rebuild one service: make dev-restart SVC=agent-registry
	@test -n "$(SVC)" || (echo "Usage: make dev-restart SVC=<n>" && exit 1)
	$(COMPOSE) up -d --build $(SVC)

COMPOSE_LOCAL := docker compose -f infra/docker-compose/docker-compose.yml
.PHONY: run-local stop-local logs-local install-cli
run-local: check-docker ## ★ Build images + start local stack (api-gateway, engine-adapter, workflow-compiler, Temporal, NATS)
	$(COMPOSE_LOCAL) up -d --build
	@echo ""
	@echo "  api-gateway  → http://localhost:7080"
	@echo "  Temporal UI  → http://localhost:7088"
	@echo "  export ZYNAX_API_URL=http://localhost:7080"

stop-local: ## Stop and remove local stack containers
	$(COMPOSE_LOCAL) down

logs-local: ## Tail all local stack logs
	$(COMPOSE_LOCAL) logs -f

install-cli: ## Build and install zynax CLI to ~/bin/zynax (requires Go 1.25)
	cd cmd/zynax && GOWORK=off go build -trimpath -o ~/bin/zynax .
	@echo "✅ zynax installed → ~/bin/zynax  (ensure ~/bin is on your PATH)"

install-ci-tools: ## Build and install zynax-ci toolchain to ~/bin/zynax-ci (requires Go 1.25)
	cd cmd/zynax-ci && GOWORK=off go build -trimpath -o ~/bin/zynax-ci .
	@echo "✅ zynax-ci installed → ~/bin/zynax-ci  (ensure ~/bin is on your PATH)"

# ── Lint ───────────────────────────────────────────────────────────────────
.PHONY: lint lint-go lint-agents lint-go-svc lint-agent lint-fix
lint: lint-protos lint-go lint-agents ## Lint everything (proto + Go + Python)

lint-go: ensure-tools ## Lint all Go platform services with golangci-lint
	@for svc in $(GO_SERVICES); do \
		echo "🔍 $$svc"; \
		$(TOOLS_RUN) sh -c "cd services/$$svc && golangci-lint run ./... --config ../../tools/golangci-lint.yml"; \
	done && echo "✅ Go lint passed"

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
.PHONY: test test-unit test-unit-go test-unit-svc test-unit-agents test-unit-agent test-bdd test-coverage test-integration
test: validate-spec test-unit test-bdd test-coverage ## ★ Full local test suite — mirrors CI (spec + Go + Python + BDD + coverage gate)
test-unit: test-unit-go test-unit-agents ## All unit tests (Go + Python)

test-unit-go: ensure-tools ## Go unit tests for all services
	@for svc in $(GO_SERVICES); do \
		if [ -f "services/$$svc/go.mod" ]; then \
			echo "🧪 $$svc"; \
			$(TOOLS_RUN) sh -c "cd services/$$svc && GOWORK=off go test ./... -v -timeout 60s -count=1" || exit 1; \
		fi; \
	done && echo "✅ Go tests passed"

test-unit-svc: ensure-tools ## Go tests for one service: make test-unit-svc SVC=workflow-compiler
	$(TOOLS_RUN) sh -c "cd services/$(SVC) && GOWORK=off go test ./... -v -timeout 60s"

test-coverage: ensure-tools ## Domain coverage gate — ≥90% on internal/domain/ for every Go service
	@failed=false; \
	for svc in $(GO_SERVICES); do \
		if [ -f "services/$$svc/go.mod" ] && [ -d "services/$$svc/internal/domain" ]; then \
			$(TOOLS_RUN) sh -c "cd services/$$svc && GOWORK=off go test ./internal/domain/... -coverprofile=domain-coverage.out -covermode=atomic -count=1 2>/dev/null"; \
			total=$$($(TOOLS_RUN) sh -c "cd services/$$svc && GOWORK=off go tool cover -func=domain-coverage.out | grep '^total:' | awk '{print \$$3}' | tr -d '%'" 2>/dev/null); \
			if [ -z "$$total" ]; then echo "  ⚠  services/$$svc: no domain coverage data"; continue; fi; \
			printf "  %-35s %s%%\n" "services/$$svc" "$$total"; \
			if awk "BEGIN{exit !($$total < 90)}"; then \
				echo "  ❌ $$total%% < 90%% — coverage gate failed for services/$$svc"; \
				failed=true; \
			fi; \
		fi; \
	done; \
	$$failed && exit 1 || echo "✅ Domain coverage gate passed (all services ≥90%%)"

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

test-integration: check-docker ## Integration tests against NATS JetStream and Redis backing services
	@echo "Starting testing backing services..."
	$(COMPOSE) --profile testing up -d
	@echo "Waiting for services to be healthy (up to 60s)..."
	@timeout 60 sh -c \
		'until docker compose -f infra/docker/docker-compose.yml ps --format json 2>/dev/null \
		  | python3 -c "import sys,json; data=sys.stdin.read(); rows=json.loads(data) if data.strip().startswith(\"[\") else [json.loads(l) for l in data.strip().splitlines() if l]; exit(0 if all(r.get(\"Health\")==\"healthy\" for r in rows if r.get(\"Health\")) and len(rows)>0 else 1)" \
		  2>/dev/null; do sleep 2; done' || true
	@for svc in $(GO_SERVICES); do \
		if [ -f "services/$$svc/tests/integration" ] || find "services/$$svc" -name "*integration*" -name "*.go" 2>/dev/null | grep -q .; then \
			echo "Integration tests: $$svc"; \
			$(TOOLS_RUN) sh -c "cd services/$$svc && GOWORK=off go test ./tests/integration/... -v -timeout 120s" || exit 1; \
		fi; \
	done
	@echo "Stopping testing backing services..."
	$(COMPOSE) --profile testing down --remove-orphans
	@echo "Integration tests passed"

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
.PHONY: security security-go security-agents scan-image audit gitleaks
security: security-go security-agents ## Full security scan (govulncheck + bandit + pip-audit + trivy)

scan-image: ## Scan one service container image for CVEs: make scan-image SVC=agent-registry
	docker build -t zynax/$(SVC):scan services/$(SVC)/
	trivy image --exit-code 1 --severity HIGH,CRITICAL --ignorefile .trivyignore zynax/$(SVC):scan
	docker rmi zynax/$(SVC):scan

gitleaks: ## Scan local repo for secrets (requires gitleaks installed: brew install gitleaks)
	gitleaks detect --source . --config tools/gitleaks-ai-context.toml --verbose

security-go: ensure-tools ## govulncheck on all Go services
	@for svc in $(GO_SERVICES); do $(TOOLS_RUN) sh -c "cd services/$$svc && govulncheck ./..."; done

security-agents: ensure-tools ## bandit + pip-audit on SDK + all agents
	@$(TOOLS_RUN) sh -c "cd agents/sdk && uv run bandit -r src/ -ll && uv run pip-audit"
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
build: check-docker ## Build all Docker images
	@for svc in $(GO_SERVICES); do docker build services/$$svc -t $(REGISTRY)/zynax-$$svc:local; done
	@for a in $(AGENTS); do [ -f "agents/examples/$$a/Dockerfile" ] && docker build agents/examples/$$a -t $(REGISTRY)/zynax-agent-$$a:local || true; done

build-svc: ## Build one service image: make build-svc SVC=agent-registry
	docker build services/$(SVC) -t $(REGISTRY)/zynax-$(SVC):local

build-agent: ## Build one agent image: make build-agent AGENT=summarizer
	docker build agents/examples/$(AGENT) -t $(REGISTRY)/zynax-agent-$(AGENT):local

# ── Cleanup ────────────────────────────────────────────────────────────────
.PHONY: clean clean-all clean-tools
clean:      ## Remove cache files
	@find . -type d -name "__pycache__" -exec rm -rf {} + 2>/dev/null; true
	@find . -name "*.pyc" -delete 2>/dev/null; true && echo "✅ Clean"
clean-tools: ## Remove tools image
	docker rmi $(TOOLS_IMAGE) 2>/dev/null || true
clean-all: clean dev-down clean-tools ## ⚠ Remove everything

# ── Spec validation ───────────────────────────────────────────────────────────
.PHONY: validate-spec validate-asyncapi validate-workflow-schema validate-agent-def-schema validate-policy-schema validate-canvas dry-run

validate-spec: validate-asyncapi validate-capability-schemas validate-workflow-schema validate-agent-def-schema validate-policy-schema ## Validate all specs (AsyncAPI + capability schemas + workflow + agent-def + policy manifests)

validate-canvas: ensure-tools ## Validate REASONS Canvas files under docs/spdd/ (SPDD — ADR-019)
	$(TOOLS_RUN) zynax-ci validate canvas docs/spdd/
	@echo "✅ Canvas validation passed"

validate-capability-schemas: ensure-tools ## Validate capability declarations in spec/ against capability.schema.json
	$(TOOLS_RUN) zynax-ci validate capabilities spec/workflows/examples/ --schema-dir spec/schemas
	@echo "✅ Capability schemas valid"

validate-workflow-schema: ensure-tools ## Validate Workflow manifests in spec/workflows/examples/ against workflow.schema.json
	$(TOOLS_RUN) zynax-ci validate workflows spec/workflows/examples/ --schema-dir spec/schemas
	@echo "✅ Workflow schemas valid"

validate-agent-def-schema: ensure-tools ## Validate AgentDef manifests in spec/workflows/examples/ against agent-def.schema.json
	$(TOOLS_RUN) zynax-ci validate agent-defs spec/workflows/examples/ --schema-dir spec/schemas
	@echo "✅ AgentDef schemas valid"

validate-policy-schema: ensure-tools ## Validate Policy manifests in spec/workflows/examples/ against policy.schema.json
	$(TOOLS_RUN) zynax-ci validate policies spec/workflows/examples/ --schema-dir spec/schemas
	@echo "✅ Policy schemas valid"

validate-asyncapi: ## Validate spec/asyncapi/zynax-events.yaml via AsyncAPI CLI (Docker)
	# renovate: datasource=docker depName=asyncapi/cli
	docker run --rm \
		-v "$(PWD)/spec":/spec \
		asyncapi/cli:6.0.0 \
		validate /spec/asyncapi/zynax-events.yaml
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
