.PHONY: help build run test clean docker-up docker-down migrate

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

build: ## Build the application
	go build -o bin/xsm ./cmd/server

run: ## Run the application locally
	go run ./cmd/server/main.go

test: ## Run tests
	go test ./... -v -cover

clean: ## Clean build artifacts
	rm -rf bin/

docker-up: ## Start docker containers
	docker-compose up -d

docker-down: ## Stop docker containers
	docker-compose down

docker-build: ## Build docker image
	docker-compose build

migrate: ## Run database migrations
	go run ./cmd/migrate/main.go

deps: ## Download dependencies
	go mod download
	go mod tidy

dev: ## Run with hot reload (requires air)
	air

lint: ## Run linter
	golangci-lint run

security: ## Run security checks
	gosec ./...