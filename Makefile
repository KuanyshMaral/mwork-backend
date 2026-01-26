.PHONY: dev build test run clean migrate-up migrate-down migrate-create docker-up docker-down lint help

# Variables
BINARY_NAME=mwork-api
MAIN_PATH=./cmd/api
MIGRATIONS_PATH=./migrations
DATABASE_URL?=postgresql://mwork:mwork_secret@localhost:5432/mwork_dev?sslmode=disable

# Colors for output
GREEN=\033[0;32m
NC=\033[0m

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# Development
dev: ## Run with hot reload (requires air)
	@echo "$(GREEN)Starting development server...$(NC)"
	air

run: ## Run the application
	@echo "$(GREEN)Running application...$(NC)"
	go run $(MAIN_PATH)/main.go

build: ## Build the binary
	@echo "$(GREEN)Building...$(NC)"
	go build -o bin/$(BINARY_NAME) $(MAIN_PATH)/main.go

clean: ## Clean build files
	@echo "$(GREEN)Cleaning...$(NC)"
	rm -rf bin/ tmp/

# Testing
test: ## Run tests
	@echo "$(GREEN)Running tests...$(NC)"
	go test -v -race ./...

test-coverage: ## Run tests with coverage
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Docker
docker-up: ## Start Docker containers
	@echo "$(GREEN)Starting Docker containers...$(NC)"
	docker-compose -f docker/docker-compose.yml up -d

docker-down: ## Stop Docker containers
	@echo "$(GREEN)Stopping Docker containers...$(NC)"
	docker-compose -f docker/docker-compose.yml down

docker-logs: ## View Docker logs
	docker-compose -f docker/docker-compose.yml logs -f

# Database Migrations
migrate-up: ## Apply all migrations
	@echo "$(GREEN)Applying migrations...$(NC)"
	migrate -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" up

migrate-down: ## Rollback last migration
	@echo "$(GREEN)Rolling back last migration...$(NC)"
	migrate -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" down 1

migrate-down-all: ## Rollback all migrations
	@echo "$(GREEN)Rolling back all migrations...$(NC)"
	migrate -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" down -all

migrate-create: ## Create new migration (usage: make migrate-create name=create_users)
	@echo "$(GREEN)Creating migration: $(name)$(NC)"
	migrate create -ext sql -dir $(MIGRATIONS_PATH) -seq $(name)

migrate-force: ## Force migration version (usage: make migrate-force version=1)
	migrate -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" force $(version)

# Linting
lint: ## Run linter
	@echo "$(GREEN)Running linter...$(NC)"
	golangci-lint run ./...

fmt: ## Format code
	@echo "$(GREEN)Formatting code...$(NC)"
	go fmt ./...
	goimports -w .

# Dependencies
deps: ## Download dependencies
	@echo "$(GREEN)Downloading dependencies...$(NC)"
	go mod download

deps-tidy: ## Tidy dependencies
	@echo "$(GREEN)Tidying dependencies...$(NC)"
	go mod tidy

# Generate
swagger: ## Generate Swagger docs
	@echo "$(GREEN)Generating Swagger docs...$(NC)"
	swag init -g $(MAIN_PATH)/main.go -o ./api/docs
