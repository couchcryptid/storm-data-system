.PHONY: start stop destroy up down clean status logs port-forward \
        test-e2e test-e2e-only reset-db help \
        install-strimzi apply-infra apply-apps apply-apps-ci build-local

HELM_RELEASE := storm-data
HELM_CHART   := helm/storm-data
NAMESPACE    := storm-data

# --- Cluster Lifecycle ---

start: ## Start minikube cluster
	minikube start --memory=4096 --cpus=2
	kubectl create namespace kafka --dry-run=client -o yaml | kubectl apply -f -
	kubectl create namespace $(NAMESPACE) --dry-run=client -o yaml | kubectl apply -f -

stop: ## Stop minikube (preserves state)
	minikube stop

destroy: ## Delete minikube cluster entirely
	minikube delete

# --- Infrastructure ---

install-strimzi: ## Install Strimzi operator
	kubectl apply -f 'https://strimzi.io/install/latest?namespace=kafka' -n kafka
	kubectl wait --for=condition=ready pod -l name=strimzi-cluster-operator -n kafka --timeout=120s

apply-infra: ## Deploy Kafka + Postgres
	kubectl apply -f k8s/kafka/ -n kafka
	kubectl wait kafka/kafka --for=condition=Ready -n kafka --timeout=300s

# --- Application ---

build-local: ## Build and load local-only images into minikube
	eval $$(minikube docker-env) && docker build -t storm-data-mock-server:latest ./mock-server

apply-apps: _create-dashboard-configmap ## Deploy all services (dev)
	helm upgrade --install $(HELM_RELEASE) $(HELM_CHART) \
		-f $(HELM_CHART)/values-dev.yaml \
		-n $(NAMESPACE)

apply-apps-ci: _create-dashboard-configmap ## Deploy all services (CI)
	helm upgrade --install $(HELM_RELEASE) $(HELM_CHART) \
		-f $(HELM_CHART)/values-ci.yaml \
		-n $(NAMESPACE)

_create-dashboard-configmap:
	kubectl create configmap dashboard-html \
		--from-file=index.html=dashboard/index.html \
		-n $(NAMESPACE) --dry-run=client -o yaml | kubectl apply -f -

# --- Full Stack ---

up: start install-strimzi build-local apply-infra apply-apps ## Full stack from nothing
	@echo "Stack deployed. Run 'make status' to check pods."

down: ## Delete all workloads but keep cluster
	-helm uninstall $(HELM_RELEASE) -n $(NAMESPACE)
	-kubectl delete configmap dashboard-html -n $(NAMESPACE) --ignore-not-found
	kubectl delete -f k8s/kafka/ -n kafka --ignore-not-found

clean: down ## Delete workloads + PVCs
	kubectl delete pvc --all -n $(NAMESPACE) --ignore-not-found
	kubectl delete pvc --all -n kafka --ignore-not-found

# --- E2E Tests ---

reset-db: ## Truncate storm_reports and restart collector
	kubectl exec -n $(NAMESPACE) postgres-0 -- psql -U storm -d stormdata -c "TRUNCATE storm_reports;"
	kubectl rollout restart deployment/collector -n $(NAMESPACE)
	@echo "Database reset. Waiting for collector to re-process..."

test-e2e: reset-db test-e2e-only ## Reset DB + run E2E tests

test-e2e-only: ## Run E2E tests against running stack
	cd e2e && go test -v -count=1 ./...

# --- Observability ---

status: ## Show all pods across namespaces
	@kubectl get pods -n kafka
	@echo "---"
	@kubectl get pods -n $(NAMESPACE)

logs: ## Tail all app service logs
	kubectl logs -f -l "app in (collector,etl,api)" -n $(NAMESPACE) --max-log-requests=10

logs-collector: ## Tail collector logs
	kubectl logs -f deployment/collector -n $(NAMESPACE)

logs-etl: ## Tail ETL logs
	kubectl logs -f deployment/etl -n $(NAMESPACE)

logs-api: ## Tail API logs
	kubectl logs -f deployment/api -n $(NAMESPACE)

port-forward: ## Forward all services to localhost
	@echo "Forwarding services to localhost..."
	@kubectl port-forward -n $(NAMESPACE) deployment/dashboard 8000:80 &
	@kubectl port-forward -n $(NAMESPACE) deployment/api 8080:8080 &
	@kubectl port-forward -n $(NAMESPACE) deployment/collector 3000:3000 &
	@kubectl port-forward -n $(NAMESPACE) deployment/etl 8081:8080 &
	@kubectl port-forward -n $(NAMESPACE) deployment/prometheus 9090:9090 &
	@kubectl port-forward -n $(NAMESPACE) deployment/kafka-ui 8082:8080 &
	@echo "Dashboard:  http://localhost:8000"
	@echo "GraphQL:    http://localhost:8080/query"
	@echo "Prometheus: http://localhost:9090"
	@echo "Kafka UI:   http://localhost:8082"
	@wait

# --- Help ---

help: ## Show all available targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
