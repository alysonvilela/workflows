-include .env
export

.DEFAULT_GOAL := help
.PHONY: help tools env deps deps-down dev dev-api dev-web migrate test test-unit test-race lint

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

env: ## Create .env from .env.example if missing
	@test -f .env || (cp .env.example .env && echo "created .env — add LLM_API_KEY before running make dev")

tools: ## Install dev tooling (air for Go hot reload, goose for migrations)
	go install github.com/air-verse/air@latest
	go install github.com/pressly/goose/v3/cmd/goose@latest
	cd frontend && pnpm install

deps: ## Start infra dependencies (postgres + rustfs)
	@mkdir -p compose-data/postgres compose-data/rustfs
	@chmod 777 compose-data/rustfs # rustfs runs as a non-root uid in its image
	docker compose up -d --wait

deps-down: ## Stop infra dependencies
	docker compose down

dev: env deps ## Start everything: infra, Go API with hot reload, frontend
	@trap 'kill 0' EXIT INT TERM; \
	$(MAKE) dev-api & \
	$(MAKE) dev-web & \
	wait

dev-api: ## Go API with hot reload (air)
	cd backend && air

dev-web: ## Frontend dev server
	cd frontend && pnpm dev --port $(WEB_PORT)

migrate: ## Apply database migrations
	cd backend && goose -dir migrations postgres "$(DATABASE_URL)" up

test: ## Full backend suite (needs deps running)
	cd backend && go test ./... -count=1

test-unit: ## Backend tests that need no infra
	cd backend && go test ./... -short -count=1

test-race: ## Concurrency invariants under the race detector (needs deps running)
	cd backend && go test ./... -race -count=1

lint:
	cd backend && go vet ./...
