# ============================================================================
# Project Alfred Makefile
# ============================================================================
# Multi-source calendar assistant with Go backend and React Native mobile app
# ============================================================================

# ----------------------------------------------------------------------------
# Variables
# ----------------------------------------------------------------------------
BINARY_NAME := alfred
GO := go
CGO_ENABLED := 1
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)

# Directories
ROOT_DIR := $(shell pwd)
MOBILE_DIR := $(ROOT_DIR)/mobile
INTERNAL_DIR := $(ROOT_DIR)/internal
E2E_DIR := $(INTERNAL_DIR)/e2e
CMD_DIR := $(ROOT_DIR)/cmd

# Docker
DOCKER_IMAGE := project-alfred
DOCKER_TAG := latest

# Ports
BACKEND_PORT := 8080
MOBILE_PORT := 8081

# Production URL
PROD_URL := https://alfred-production-d2c9.up.railway.app

# ----------------------------------------------------------------------------
# Phony Targets
# ----------------------------------------------------------------------------
.PHONY: help \
        dev dev-mobile dev-mobile-ios dev-mobile-android dev-all dev-stop \
        test test-unit test-e2e test-mobile test-mobile-watch test-mobile-coverage \
        test-mobile-e2e test-mobile-e2e-onboarding test-mobile-e2e-events \
        test-mobile-e2e-settings test-mobile-e2e-navigation test-all test-server \
        build build-linux build-docker build-docker-run \
        build-mobile-dev build-mobile-preview build-mobile-preview-ios build-mobile-preview-android \
        build-mobile-prod build-mobile-prod-ios build-mobile-prod-android \
        deploy deploy-status deploy-logs deploy-logs-follow deploy-env \
        clean clean-all install install-go install-mobile \
        lint lint-go lint-mobile fmt \
        db-reset health health-prod \
        ci-test ci-build

# Default target
.DEFAULT_GOAL := help

# ----------------------------------------------------------------------------
# Help
# ----------------------------------------------------------------------------
help: ## Show this help message
	@echo "Project Alfred Makefile"
	@echo ""
	@echo "Development:"
	@grep -E '^(dev|dev-mobile|dev-mobile-ios|dev-mobile-android|dev-all|dev-stop):.*##' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*## "}; {printf "  %-28s %s\n", $$1, $$2}'
	@echo ""
	@echo "Testing:"
	@grep -E '^(test|test-unit|test-e2e|test-mobile|test-mobile-watch|test-mobile-coverage|test-mobile-e2e|test-all|test-server):.*##' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*## "}; {printf "  %-28s %s\n", $$1, $$2}'
	@echo ""
	@echo "Building:"
	@grep -E '^(build|build-linux|build-docker|build-docker-run|build-mobile-dev|build-mobile-preview|build-mobile-prod):.*##' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*## "}; {printf "  %-28s %s\n", $$1, $$2}'
	@echo ""
	@echo "Deployment:"
	@grep -E '^(deploy|deploy-status|deploy-logs|deploy-logs-follow|deploy-env):.*##' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*## "}; {printf "  %-28s %s\n", $$1, $$2}'
	@echo ""
	@echo "Utilities:"
	@grep -E '^(clean|clean-all|install|install-go|install-mobile|lint|lint-go|lint-mobile|fmt|db-reset|health|health-prod):.*##' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*## "}; {printf "  %-28s %s\n", $$1, $$2}'
	@echo ""
	@echo "CI/CD:"
	@grep -E '^(ci-test|ci-build):.*##' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*## "}; {printf "  %-28s %s\n", $$1, $$2}'

# ----------------------------------------------------------------------------
# Development Targets
# ----------------------------------------------------------------------------
dev: ## Run Go backend locally (port 8080)
	@echo "Starting backend server on port $(BACKEND_PORT)..."
	CGO_ENABLED=$(CGO_ENABLED) $(GO) run main.go

dev-mobile: ## Run mobile app locally (web, port 8081)
	@echo "Starting mobile app on port $(MOBILE_PORT)..."
	cd $(MOBILE_DIR) && npm run web

dev-mobile-ios: ## Run mobile app on iOS simulator
	@echo "Starting mobile app on iOS..."
	cd $(MOBILE_DIR) && npm run ios

dev-mobile-android: ## Run mobile app on Android emulator
	@echo "Starting mobile app on Android..."
	cd $(MOBILE_DIR) && npm run android

dev-all: ## Run both backend and mobile (background)
	@echo "Starting backend and mobile in background..."
	@echo "Backend: http://localhost:$(BACKEND_PORT)"
	@echo "Mobile: http://localhost:$(MOBILE_PORT)"
	@CGO_ENABLED=$(CGO_ENABLED) $(GO) run main.go &
	@cd $(MOBILE_DIR) && npm run web &
	@echo "Services started. Use 'make dev-stop' to stop them."

dev-stop: ## Stop background development services
	@echo "Stopping development services..."
	@-pkill -f "go run main.go" 2>/dev/null || true
	@-pkill -f "expo start" 2>/dev/null || true
	@echo "Services stopped."

# ----------------------------------------------------------------------------
# Testing Targets
# ----------------------------------------------------------------------------
test: test-unit ## Run all backend tests (alias for test-unit)

test-unit: ## Run backend unit tests (excludes e2e)
	@echo "Running backend unit tests..."
	CGO_ENABLED=$(CGO_ENABLED) $(GO) test -v $$(go list ./internal/... | grep -v /e2e)

test-e2e: ## Run backend E2E tests
	@echo "Running backend E2E tests..."
	CGO_ENABLED=$(CGO_ENABLED) $(GO) test -v ./internal/e2e/...

test-mobile: ## Run mobile unit tests (Jest)
	@echo "Running mobile unit tests..."
	cd $(MOBILE_DIR) && npm test

test-mobile-watch: ## Run mobile tests in watch mode
	@echo "Running mobile tests in watch mode..."
	cd $(MOBILE_DIR) && npm run test:watch

test-mobile-coverage: ## Run mobile tests with coverage
	@echo "Running mobile tests with coverage..."
	cd $(MOBILE_DIR) && npm run test:coverage

test-mobile-e2e: ## Run mobile E2E tests (Maestro)
	@echo "Running mobile E2E tests with Maestro..."
	cd $(MOBILE_DIR) && npm run e2e

test-mobile-e2e-onboarding: ## Run mobile E2E onboarding tests
	@echo "Running mobile E2E onboarding tests..."
	cd $(MOBILE_DIR) && npm run e2e:onboarding

test-mobile-e2e-events: ## Run mobile E2E events tests
	@echo "Running mobile E2E events tests..."
	cd $(MOBILE_DIR) && npm run e2e:events

test-mobile-e2e-settings: ## Run mobile E2E settings tests
	@echo "Running mobile E2E settings tests..."
	cd $(MOBILE_DIR) && npm run e2e:settings

test-mobile-e2e-navigation: ## Run mobile E2E navigation tests
	@echo "Running mobile E2E navigation tests..."
	cd $(MOBILE_DIR) && npm run e2e:navigation

test-all: ## Run all tests (backend + mobile)
	@echo "Running all tests..."
	@$(MAKE) test-unit
	@$(MAKE) test-e2e
	@$(MAKE) test-mobile
	@echo "All tests completed."

test-server: ## Run E2E test server (in-memory DB, Claude API)
	@echo "Starting E2E test server..."
	@echo "Requires: ANTHROPIC_API_KEY environment variable"
	CGO_ENABLED=$(CGO_ENABLED) $(GO) run $(CMD_DIR)/testserver/main.go

# ----------------------------------------------------------------------------
# Build Targets
# ----------------------------------------------------------------------------
build: ## Build Go binary for current OS/arch
	@echo "Building $(BINARY_NAME) for $(GOOS)/$(GOARCH)..."
	CGO_ENABLED=$(CGO_ENABLED) $(GO) build -o $(BINARY_NAME) .
	@echo "Built: ./$(BINARY_NAME)"

build-linux: ## Build Go binary for Linux (for deployment)
	@echo "Building $(BINARY_NAME) for linux/amd64..."
	CGO_ENABLED=$(CGO_ENABLED) GOOS=linux GOARCH=amd64 $(GO) build -o $(BINARY_NAME)-linux .
	@echo "Built: ./$(BINARY_NAME)-linux"

build-docker: ## Build Docker image
	@echo "Building Docker image $(DOCKER_IMAGE):$(DOCKER_TAG)..."
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	@echo "Built: $(DOCKER_IMAGE):$(DOCKER_TAG)"

build-docker-run: build-docker ## Build and run Docker image locally
	@echo "Running Docker container..."
	docker run -p $(BACKEND_PORT):$(BACKEND_PORT) \
		-e ANTHROPIC_API_KEY \
		-v $(ROOT_DIR)/data:/data \
		$(DOCKER_IMAGE):$(DOCKER_TAG)

build-mobile-dev: ## Build mobile development client (EAS)
	@echo "Building mobile development client..."
	cd $(MOBILE_DIR) && npx eas build --profile development --platform all

build-mobile-preview: ## Build mobile preview (EAS, internal distribution)
	@echo "Building mobile preview..."
	cd $(MOBILE_DIR) && npx eas build --profile preview --platform all

build-mobile-preview-ios: ## Build mobile preview for iOS only
	@echo "Building mobile preview for iOS..."
	cd $(MOBILE_DIR) && npx eas build --profile preview --platform ios

build-mobile-preview-android: ## Build mobile preview for Android only
	@echo "Building mobile preview for Android..."
	cd $(MOBILE_DIR) && npx eas build --profile preview --platform android

build-mobile-prod: ## Build mobile production (EAS, store distribution)
	@echo "Building mobile production..."
	cd $(MOBILE_DIR) && npx eas build --profile production --platform all

build-mobile-prod-ios: ## Build mobile production for iOS only
	@echo "Building mobile production for iOS..."
	cd $(MOBILE_DIR) && npx eas build --profile production --platform ios

build-mobile-prod-android: ## Build mobile production for Android only
	@echo "Building mobile production for Android..."
	cd $(MOBILE_DIR) && npx eas build --profile production --platform android

# ----------------------------------------------------------------------------
# Deployment Targets
# ----------------------------------------------------------------------------
deploy: ## Deploy backend to Railway
	@echo "Deploying to Railway..."
	railway up
	@echo "Deployed to $(PROD_URL)"

deploy-status: ## Check Railway deployment status
	@echo "Checking Railway deployment status..."
	railway status

deploy-logs: ## View Railway deployment logs
	@echo "Viewing Railway logs..."
	railway logs

deploy-logs-follow: ## Follow Railway deployment logs
	@echo "Following Railway logs (Ctrl+C to stop)..."
	railway logs --follow

deploy-env: ## Show Railway environment variables
	@echo "Railway environment variables..."
	railway variables

# ----------------------------------------------------------------------------
# Utility Targets
# ----------------------------------------------------------------------------
clean: ## Clean build artifacts and caches
	@echo "Cleaning build artifacts..."
	rm -f $(BINARY_NAME) $(BINARY_NAME)-linux
	rm -rf $(MOBILE_DIR)/.expo
	rm -rf $(MOBILE_DIR)/dist
	$(GO) clean -cache -testcache
	@echo "Cleaned."

clean-all: clean ## Clean everything including node_modules
	@echo "Cleaning node_modules..."
	rm -rf $(MOBILE_DIR)/node_modules
	@echo "All cleaned."

install: install-go install-mobile ## Install all dependencies

install-go: ## Install Go dependencies
	@echo "Installing Go dependencies..."
	$(GO) mod download
	$(GO) mod tidy
	@echo "Go dependencies installed."

install-mobile: ## Install mobile dependencies (npm)
	@echo "Installing mobile dependencies..."
	cd $(MOBILE_DIR) && npm install
	@echo "Mobile dependencies installed."

lint: lint-go lint-mobile ## Run all linters

lint-go: ## Run Go linter (go vet)
	@echo "Linting Go code..."
	$(GO) vet ./...
	@echo "Go linting complete."

lint-mobile: ## Run mobile linter (ESLint)
	@echo "Linting mobile code..."
	cd $(MOBILE_DIR) && npm run lint

fmt: ## Format Go code
	@echo "Formatting Go code..."
	$(GO) fmt ./...
	@echo "Code formatted."

db-reset: ## Reset local database (WARNING: destroys data)
	@echo "WARNING: This will delete your local database!"
	@read -p "Are you sure? [y/N] " confirm && [ "$$confirm" = "y" ] || exit 1
	rm -f alfred.db whatsapp.db telegram.db
	@echo "Database reset. Run 'make dev' to recreate."

health: ## Check backend health endpoint (local)
	@echo "Checking backend health..."
	@curl -s http://localhost:$(BACKEND_PORT)/health | python3 -m json.tool 2>/dev/null || \
		curl -s http://localhost:$(BACKEND_PORT)/health || \
		echo "Backend not running or health check failed"

health-prod: ## Check production backend health
	@echo "Checking production health..."
	@curl -s $(PROD_URL)/health | python3 -m json.tool 2>/dev/null || \
		curl -s $(PROD_URL)/health || \
		echo "Production health check failed"

# ----------------------------------------------------------------------------
# CI/CD Targets
# ----------------------------------------------------------------------------
ci-test: ## Run tests suitable for CI (with coverage)
	@echo "Running CI tests..."
	CGO_ENABLED=$(CGO_ENABLED) $(GO) test -race -coverprofile=coverage.out ./internal/...
	cd $(MOBILE_DIR) && npm run test:coverage
	@echo "CI tests completed."

ci-build: build build-docker ## Build all artifacts for CI
	@echo "CI build completed."
