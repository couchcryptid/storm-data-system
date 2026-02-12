.PHONY: up up-ci down logs test-e2e test-e2e-ci test-e2e-only build clean ps wait-healthy help

# --- Stack Management ---

up: ## Start the full stack (builds from local source)
	docker compose up -d --build --wait

up-ci: ## Start the full stack using published images
	docker compose -f compose.yml -f compose.ci.yml up -d --wait

down: ## Stop and remove all containers
	docker compose down

clean: ## Stop, remove containers, volumes, and orphans
	docker compose down -v --remove-orphans

build: ## Build all service images
	docker compose build

ps: ## Show running services
	docker compose ps

logs: ## Tail logs from all services
	docker compose logs -f

logs-collector: ## Tail collector logs
	docker compose logs -f collector

logs-etl: ## Tail ETL logs
	docker compose logs -f etl

logs-api: ## Tail API logs
	docker compose logs -f api

# --- E2E Tests ---

test-e2e: up ## Run E2E tests (starts stack, builds from source)
	@echo "Running E2E tests..."
	cd e2e && go test -v -count=1 -timeout 5m ./...

test-e2e-ci: up-ci ## Run E2E tests (starts stack, published images)
	@echo "Running E2E tests..."
	cd e2e && go test -v -count=1 -timeout 5m ./...

test-e2e-only: ## Run E2E tests against an already-running stack
	cd e2e && go test -v -count=1 -timeout 5m ./...

# --- Helpers ---

wait-healthy: ## Wait for all services to be healthy
	@echo "Waiting for services..."
	@docker compose ps --format json | grep -q '"Health":"healthy"' || \
		(echo "Some services are not healthy. Run 'make ps' to check." && exit 1)
	@echo "All services healthy."

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
