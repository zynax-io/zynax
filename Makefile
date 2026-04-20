# SPDX-License-Identifier: Apache-2.0
# Keel — Makefile
# Docker-first. Only prerequisite: Docker Desktop.
# Go platform services + Python/SDK agent layer.

.DEFAULT_GOAL := help
SHELL         := /bin/bash
GO_SERVICES   := agent-registry task-broker memory-service event-bus api-gateway workflow-compiler engine-adapter
AGENTS        := summarizer researcher calculator
COMPOSE       := docker compose -f infra/docker/docker-compose.yml
COMPOSE_TOOLS := docker compose -f infra/docker/docker-compose.tools.yml
TOOLS_IMAGE   := keel-tools:local
REGISTRY      := ghcr.io/keel-io
TOOLS_RUN     := docker run --rm -v "$(PWD)":/workspace -w /workspace --env-file infra/docker/.env.tools $(TOOLS_IMAGE)

.PHONY: help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: bootstrap check-docker build-tools
bootstrap: check-docker build-tools ## ★ Run once after clone
	@echo "✅ Done. make dev-up → start stack, make lint → lint all"

check-docker:
	@docker info >/dev/null 2>&1 || (echo "❌ Docker not running" && exit 1)
	@echo "✅ Docker $(shell docker version --format '{{.Server.Version}}')"

build-tools: check-docker ## Build the dev-tools image (Go 1.22 + Python 3.12 + all tools)
	docker build -f infra/docker/Dockerfile.tools -t $(TOOLS_IMAGE) .
	@echo "✅ Tools image: $(TOOLS_IMAGE)"

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

# ── Lint ───────────────────────────────────────────────────────────────────
.PHONY: lint lint-go lint-agents lint-go-svc lint-agent lint-fix
lint: lint-protos lint-go lint-agents ## Lint everything (proto + Go + Python)

lint-go: build-tools ## Lint all Go platform services with golangci-lint
	@for svc in $(GO_SERVICES); do \
		echo "🔍 $$svc"; \
		$(TOOLS_RUN) sh -c "cd services/$$svc && golangci-lint run ./... --config ../../tools/golangci-lint.yml"; \
	done && echo "✅ Go lint passed"

lint-go-svc: build-tools ## Lint one Go service: make lint-go-svc SVC=agent-registry
	$(TOOLS_RUN) sh -c "cd services/$(SVC) && golangci-lint run ./... --config ../../tools/golangci-lint.yml"

lint-agents: build-tools ## Lint SDK + all Python agents (ruff + mypy)
	$(TOOLS_RUN) sh -c "cd agents/sdk && uv run ruff check src/ && uv run mypy src/ --strict"
	@for a in $(AGENTS); do [ -f "agents/examples/$$a/pyproject.toml" ] && \
		$(TOOLS_RUN) sh -c "cd agents/examples/$$a && uv run ruff check src/ tests/ && uv run mypy src/ --strict" || true; done
	@echo "✅ Python lint passed"

lint-agent: build-tools ## Lint one agent: make lint-agent AGENT=summarizer
	$(TOOLS_RUN) sh -c "cd agents/examples/$(AGENT) && uv run ruff check src/ tests/ && uv run mypy src/ --strict"

lint-fix: build-tools ## Auto-fix Python agent lint errors
	@for a in $(AGENTS); do [ -f "agents/examples/$$a/pyproject.toml" ] && \
		$(TOOLS_RUN) sh -c "cd agents/examples/$$a && uv run ruff check --fix src/ tests/ && uv run ruff format src/ tests/" || true; done

# ── Tests ──────────────────────────────────────────────────────────────────
.PHONY: test test-unit test-unit-go test-unit-svc test-unit-agents test-unit-agent test-integration
test: test-unit ## Run all unit tests
test-unit: test-unit-go test-unit-agents ## All unit tests (Go + Python)

test-unit-go: build-tools ## Go unit tests for all services
	@for svc in $(GO_SERVICES); do \
		echo "🧪 $$svc"; \
		$(TOOLS_RUN) sh -c "cd services/$$svc && go test ./... -v -race -timeout 60s"; \
	done && echo "✅ Go tests passed"

test-unit-svc: build-tools ## Go tests for one service: make test-unit-svc SVC=agent-registry
	$(TOOLS_RUN) sh -c "cd services/$(SVC) && go test ./... -v -race -timeout 60s"

test-unit-agents: build-tools ## pytest-bdd for SDK + all Python agents
	$(TOOLS_RUN) sh -c "cd agents/sdk && uv run pytest tests/ --cov=src --cov-fail-under=90 -v"
	@for a in $(AGENTS); do [ -f "agents/examples/$$a/pyproject.toml" ] && \
		$(TOOLS_RUN) sh -c "cd agents/examples/$$a && uv run pytest tests/ --cov=src --cov-fail-under=90 -v" || true; done
	@echo "✅ Python tests passed"

test-unit-agent: build-tools ## pytest for one agent: make test-unit-agent AGENT=summarizer
	$(TOOLS_RUN) sh -c "cd agents/examples/$(AGENT) && uv run pytest tests/ -v"

test-integration: check-docker ## Integration tests against real backing services
	@for svc in $(GO_SERVICES); do \
		$(COMPOSE_TOOLS) run --rm test-runner sh -c "cd services/$$svc && go test ./tests/integration/... -v -timeout 120s"; \
	done

# ── Proto generation + lint ────────────────────────────────────────────────
.PHONY: generate-protos lint-protos
generate-protos: build-tools ## Generate Go + Python stubs from .proto files
	$(TOOLS_RUN) sh -c "cd protos && buf generate --template buf.gen.yaml"
	@echo "✅ Stubs in protos/generated/go/ and protos/generated/python/ — commit them"

lint-protos: build-tools ## buf lint + format check on all proto files
	$(TOOLS_RUN) sh -c "cd protos && buf lint && buf format --diff --exit-code"
	@echo "✅ Proto lint passed"

# ── Security ───────────────────────────────────────────────────────────────
.PHONY: security security-go security-agents
security: security-go security-agents ## All security scans

security-go: build-tools ## govulncheck on all Go services
	@for svc in $(GO_SERVICES); do $(TOOLS_RUN) sh -c "cd services/$$svc && govulncheck ./..."; done

security-agents: build-tools ## bandit + pip-audit on all agents
	@for a in $(AGENTS); do [ -f "agents/examples/$$a/pyproject.toml" ] && \
		$(TOOLS_RUN) sh -c "cd agents/examples/$$a && uv run bandit -r src/ -ll && uv run pip-audit" || true; done

# ── Build images ───────────────────────────────────────────────────────────
.PHONY: build build-svc build-agent
build: check-docker ## Build all Docker images
	@for svc in $(GO_SERVICES); do docker build services/$$svc -t $(REGISTRY)/keel-$$svc:local; done
	@for a in $(AGENTS); do [ -f "agents/examples/$$a/Dockerfile" ] && docker build agents/examples/$$a -t $(REGISTRY)/keel-agent-$$a:local || true; done

build-svc: ## Build one service image: make build-svc SVC=agent-registry
	docker build services/$(SVC) -t $(REGISTRY)/keel-$(SVC):local

build-agent: ## Build one agent image: make build-agent AGENT=summarizer
	docker build agents/examples/$(AGENT) -t $(REGISTRY)/keel-agent-$(AGENT):local

# ── Cleanup ────────────────────────────────────────────────────────────────
.PHONY: clean clean-all clean-tools
clean:      ## Remove cache files
	@find . -type d -name "__pycache__" -exec rm -rf {} + 2>/dev/null; true
	@find . -name "*.pyc" -delete 2>/dev/null; true && echo "✅ Clean"
clean-tools: ## Remove tools image
	docker rmi $(TOOLS_IMAGE) 2>/dev/null || true
clean-all: clean dev-down clean-tools ## ⚠ Remove everything

# ── Spec validation ───────────────────────────────────────────────────────────
.PHONY: validate-spec validate-asyncapi dry-run

validate-spec: validate-asyncapi validate-capability-schemas ## Validate all specs (AsyncAPI + capability schemas)

validate-capability-schemas: ## Validate capability declarations in spec/ against capability.schema.json
	docker run --rm \
		-v "$(PWD)":/workspace \
		-w /workspace \
		python:3.12-slim sh -c " \
			pip install --quiet jsonschema pyyaml && \
			python tools/validate_capabilities.py spec/schemas/capability.schema.json spec/workflows/examples/ \
		"
	@echo "✅ Capability schemas valid"

validate-asyncapi: ## Validate spec/asyncapi/zynax-events.yaml via AsyncAPI CLI (Docker)
	docker run --rm \
		-v "$(PWD)/spec":/spec \
		asyncapi/cli:latest \
		validate /spec/asyncapi/zynax-events.yaml
	@echo "✅ AsyncAPI spec valid"

dry-run: build-tools ## Dry-run a workflow: make dry-run FILE=spec/workflows/examples/code-review.yaml
	@test -n "$(FILE)" || (echo "Usage: make dry-run FILE=<path>" && exit 1)
	$(TOOLS_RUN) sh -c "keel apply --dry-run $(FILE)"
