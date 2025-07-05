# Variables
APP_NAME := ad-tracking
DOCKER_IMAGE := $(APP_NAME):latest
DOCKER_REGISTRY := your-registry.com
GO_VERSION := 1.21

# Default target
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  setup          - Setup local development environment"
	@echo "  build          - Build the Go application"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage"
	@echo "  docker-build   - Build Docker image"
	@echo "  docker-run     - Run Docker container"
	@echo "  compose-up     - Start services with docker-compose"
	@echo "  compose-down   - Stop services with docker-compose"
	@echo "  compose-logs   - View docker-compose logs"
	@echo "  clean          - Clean build artifacts"
	@echo "  fmt            - Format Go code"
	@echo "  lint           - Run linter"
	@echo "  dev            - Run in development mode"
	@echo "  db-migrate     - Run database migrations"
	@echo "  db-seed        - Seed database with test data"
	@echo "  load-test      - Run load tests"

# Setup local development
.PHONY: setup
setup:
	@chmod +x scripts/setup-local.sh
	@./scripts/setup-local.sh

# Build
.PHONY: build
build:
	go build -o bin/$(APP_NAME) .

# Test
.PHONY: test
test:
	go test -v ./...

.PHONY: test-coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Docker
.PHONY: docker-build
docker-build:
	docker build -t $(DOCKER_IMAGE) .

.PHONY: docker-run
docker-run:
	docker run -p 8080:8080 \
		-e DATABASE_URL="postgresql://ad_user:ad_password@localhost:5432/ad_tracking_db?sslmode=disable" \
		$(DOCKER_IMAGE)

# Docker Compose
.PHONY: compose-up
compose-up:
	docker-compose up -d --build

.PHONY: compose-down
compose-down:
	docker-compose down

.PHONY: compose-logs
compose-logs:
	docker-compose logs -f

.PHONY: compose-restart
compose-restart:
	docker-compose restart

# Development
.PHONY: dev
dev:
	@echo "Starting development server..."
	@if [ -f .env ]; then set -a && source .env && set +a; fi && \
	go run . --mode=debug

# Database
.PHONY: db-migrate
db-migrate:
	@echo "Running database migrations..."
	@go run scripts/migrate.go

.PHONY: db-seed
db-seed:
	@echo "Seeding database..."
	@go run scripts/seed.go

# Load testing
.PHONY: load-test
load-test:
	@echo "Running load tests..."
	@go run scripts/load_test.go

# Code quality
.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: lint
lint:
	golangci-lint run

.PHONY: vet
vet:
	go vet ./...

# Clean
.PHONY: clean
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html
	docker-compose down --volumes --remove-orphans
	docker system prune -f

# Install dependencies
.PHONY: deps
deps:
	go mod download
	go mod tidy

# Generate
.PHONY: generate
generate:
	go generate ./...

# Production deployment
.PHONY: deploy
deploy:
	@echo "Deploying to production..."
	@docker build -t $(DOCKER_REGISTRY)/$(DOCKER_IMAGE) .
	@docker push $(DOCKER_REGISTRY)/$(DOCKER_IMAGE)

# Quick start
.PHONY: start
start: setup compose-up
	@echo "🚀 Ad Tracking System is running!"
	@echo "API: http://localhost:8080"
	@echo "Metrics: http://localhost:9090"
	@echo "Dashboard: http://localhost:3000"

# Stop all services
.PHONY: stop
stop: compose-down
	@echo "🛑 All services stopped"