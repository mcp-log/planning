.PHONY: help build test test-unit test-integration lint clean docker-up docker-down migrate-up migrate-down run

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Build the planning service
	go build -o bin/planning ./cmd/planning

test: ## Run all tests
	go test ./... -v -race -coverprofile=coverage.out

test-unit: ## Run unit tests only
	go test ./internal/domain/... -v -race

test-integration: ## Run integration tests
	go test ./internal/ports/... -v -race

lint: ## Run linters
	golangci-lint run

clean: ## Clean build artifacts
	rm -rf bin/
	rm -f coverage.out

docker-up: ## Start infrastructure (Postgres + Kafka)
	docker-compose up -d
	@echo "Waiting for services to be healthy..."
	@sleep 5

docker-down: ## Stop infrastructure
	docker-compose down

migrate-up: ## Apply database migrations
	migrate -path migrations -database "postgres://planning:planning@localhost:5433/oms_planning?sslmode=disable" up

migrate-down: ## Rollback database migrations
	migrate -path migrations -database "postgres://planning:planning@localhost:5433/oms_planning?sslmode=disable" down

run: ## Run the planning service
	go run ./cmd/planning

.DEFAULT_GOAL := help
