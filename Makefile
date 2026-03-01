.PHONY: start stop destroy up down status logs port-forward \
        test-e2e test-e2e-only reset-db help \
        install-strimzi apply-infra apply-apps build-local

# --- Cluster Lifecycle ---

start: ## Start minikube cluster
	minikube start --memory=4096 --cpus=2
	kubectl create namespace kafka --dry-run=client -o yaml | kubectl apply -f -
	kubectl create namespace hailtrace --dry-run=client -o yaml | kubectl apply -f -

stop: ## Stop minikube (preserves state)
	minikube stop

destroy: ## Delete minikube cluster entirely
	minikube delete

# --- Infrastructure ---

install-strimzi: ## Install Strimzi operator
	kubectl apply -f 'https://strimzi.io/install/latest?namespace=kafka' -n kafka
	kubectl wait --for=condition=ready pod -l name=strimzi-cluster-operator -n kafka --timeout=120s

apply-infra: ## Deploy Kafka + Postgres
	kubectl apply -f k8s/base/kafka/ -n kafka
	kubectl wait kafka/kafka --for=condition=Ready -n kafka --timeout=300s
	kubectl apply -f k8s/base/postgres/ -n hailtrace
	kubectl wait --for=condition=ready pod -l app=postgres -n hailtrace --timeout=120s

# --- Application ---

build-local: ## Build and load local-only images into minikube
	eval $$(minikube docker-env) && docker build -t storm-data-mock-server:latest ./mock-server

apply-apps: ## Deploy all application services (dev overlay)
	kubectl apply -k k8s/overlays/dev/

apply-apps-ci: ## Deploy all application services (CI overlay)
	kubectl apply -k k8s/overlays/ci/

# --- Full Stack ---

up: start install-strimzi build-local apply-infra apply-apps ## Full stack from nothing
	@echo "Stack deployed. Run 'make status' to check pods."

down: ## Delete all workloads but keep cluster
	kubectl delete -k k8s/overlays/dev/ --ignore-not-found
	kubectl delete -f k8s/base/kafka/ -n kafka --ignore-not-found

clean: down ## Delete workloads + PVCs
	kubectl delete pvc --all -n hailtrace --ignore-not-found
	kubectl delete pvc --all -n kafka --ignore-not-found

# --- E2E Tests ---

reset-db: ## Truncate storm_reports and restart collector
	kubectl exec -n hailtrace postgres-0 -- psql -U storm -d stormdata -c "TRUNCATE storm_reports;"
	kubectl rollout restart deployment/collector -n hailtrace
	@echo "Database reset. Collector restarting..."

test-e2e: reset-db ## Reset DB and run E2E tests
	@echo "Waiting for collector to be ready..."
	kubectl wait --for=condition=ready pod -l app=collector -n hailtrace --timeout=60s
	@echo "Running E2E tests..."
	cd e2e && go test -v -count=1 -timeout 5m ./...

test-e2e-only: ## Run E2E tests against running stack
	cd e2e && go test -v -count=1 -timeout 5m ./...

# --- Observability ---

status: ## Show all pods
	@kubectl get pods -n kafka
	@kubectl get pods -n hailtrace

logs: ## Tail all app service logs
	kubectl logs -f -l app -n hailtrace --max-log-requests=10

logs-collector: ## Tail collector logs
	kubectl logs -f deployment/collector -n hailtrace

logs-etl: ## Tail ETL logs
	kubectl logs -f deployment/etl -n hailtrace

logs-api: ## Tail API logs
	kubectl logs -f deployment/api -n hailtrace

# --- Port Forwarding ---

port-forward: ## Forward all services to localhost
	@echo "API:        http://localhost:8080"
	@echo "Collector:  http://localhost:3000"
	@echo "Dashboard:  http://localhost:8000"
	@echo "Prometheus: http://localhost:9090"
	@echo "Kafka UI:   http://localhost:8082"
	@echo ""
	@echo "Starting port-forwards (Ctrl+C to stop)..."
	kubectl port-forward -n hailtrace deployment/api 8080:8080 &
	kubectl port-forward -n hailtrace deployment/collector 3000:3000 &
	kubectl port-forward -n hailtrace deployment/dashboard 8000:80 &
	kubectl port-forward -n hailtrace deployment/prometheus 9090:9090 &
	kubectl port-forward -n hailtrace deployment/kafka-ui 8082:8080 &
	@wait

# --- Help ---

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
