# GOV.UK Cost Dashboard Makefile
# =================================

# Variables
APP_NAME := govuk-cost-dashboard
BINARY_NAME := $(APP_NAME)
GO_VERSION := 1.21
DOCKER_TAG := latest
EXAMPLE_BINARY := govuk-example

# Default target
.DEFAULT_GOAL := help

# Colors for output
BLUE := \033[34m
GREEN := \033[32m
YELLOW := \033[33m
RED := \033[31m
RESET := \033[0m

## 🏗️  Build Commands
.PHONY: build
build: ## 🔨 Build the application binary
	@echo "$(BLUE)🔨 Building $(APP_NAME)...$(RESET)"
	@go build -o bin/$(BINARY_NAME) ./cmd/server
	@echo "$(GREEN)✅ Build complete: bin/$(BINARY_NAME)$(RESET)"

.PHONY: build-example
build-example: ## 📚 Build the example application
	@echo "$(BLUE)🔨 Building example application...$(RESET)"
	@go build -o bin/$(EXAMPLE_BINARY) ./examples/govuk_apps_example.go
	@echo "$(GREEN)✅ Example build complete: bin/$(EXAMPLE_BINARY)$(RESET)"

.PHONY: build-all
build-all: build build-example ## 🏗️  Build all binaries

## 🧪 Testing Commands
.PHONY: test
test: ## 🧪 Run all tests
	@echo "$(BLUE)🧪 Running tests...$(RESET)"
	@go test -v ./...

.PHONY: test-coverage
test-coverage: ## 📊 Run tests with coverage report
	@echo "$(BLUE)📊 Running tests with coverage...$(RESET)"
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)✅ Coverage report generated: coverage.html$(RESET)"

.PHONY: test-race
test-race: ## 🏃 Run tests with race detection
	@echo "$(BLUE)🏃 Running tests with race detection...$(RESET)"
	@go test -race -v ./...

.PHONY: test-short
test-short: ## ⚡ Run short tests only
	@echo "$(BLUE)⚡ Running short tests...$(RESET)"
	@go test -short -v ./...

.PHONY: test-govuk
test-govuk: ## 🏛️  Test GOV.UK client only
	@echo "$(BLUE)🏛️  Testing GOV.UK client...$(RESET)"
	@go test -v ./pkg/govuk

.PHONY: test-aws
test-aws: ## ☁️  Test AWS client only
	@echo "$(BLUE)☁️  Testing AWS client...$(RESET)"
	@go test -v ./pkg/aws

## 🚀 Run Commands
.PHONY: run
run: ## 🚀 Run the application
	@echo "$(BLUE)🚀 Starting $(APP_NAME)...$(RESET)"
	@go run ./cmd/server

.PHONY: run-example
run-example: ## 📚 Run the GOV.UK apps example
	@echo "$(BLUE)📚 Running GOV.UK apps example...$(RESET)"
	@go run ./examples/govuk_apps_example.go

.PHONY: run-with-profile
run-with-profile: ## 🔐 Run with AWS profile (set AWS_PROFILE env var)
	@echo "$(BLUE)🔐 Running with AWS profile: $(AWS_PROFILE)...$(RESET)"
	@AWS_PROFILE=$(AWS_PROFILE) go run ./cmd/server

## 🔧 Development Commands
.PHONY: deps
deps: ## 📦 Download and tidy dependencies
	@echo "$(BLUE)📦 Downloading dependencies...$(RESET)"
	@go mod download
	@go mod tidy
	@echo "$(GREEN)✅ Dependencies updated$(RESET)"

.PHONY: clean
clean: ## 🧹 Clean build artifacts
	@echo "$(BLUE)🧹 Cleaning build artifacts...$(RESET)"
	@rm -rf bin/
	@rm -f coverage.out coverage.html
	@go clean
	@echo "$(GREEN)✅ Clean complete$(RESET)"

.PHONY: fmt
fmt: ## 🎨 Format Go code
	@echo "$(BLUE)🎨 Formatting code...$(RESET)"
	@go fmt ./...
	@echo "$(GREEN)✅ Code formatted$(RESET)"

.PHONY: vet
vet: ## 🔍 Run go vet
	@echo "$(BLUE)🔍 Running go vet...$(RESET)"
	@go vet ./...
	@echo "$(GREEN)✅ Vet check passed$(RESET)"

.PHONY: lint
lint: ## 📏 Run linter (requires golangci-lint)
	@echo "$(BLUE)📏 Running linter...$(RESET)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
		echo "$(GREEN)✅ Linting complete$(RESET)"; \
	else \
		echo "$(YELLOW)⚠️  golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest$(RESET)"; \
	fi

.PHONY: security
security: ## 🔒 Run security checks (requires gosec)
	@echo "$(BLUE)🔒 Running security checks...$(RESET)"
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
		echo "$(GREEN)✅ Security check complete$(RESET)"; \
	else \
		echo "$(YELLOW)⚠️  gosec not installed. Install with: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest$(RESET)"; \
	fi

## 🐳 Docker Commands
.PHONY: docker-build
docker-build: ## 🐳 Build Docker image
	@echo "$(BLUE)🐳 Building Docker image...$(RESET)"
	@docker build -t $(APP_NAME):$(DOCKER_TAG) .
	@echo "$(GREEN)✅ Docker image built: $(APP_NAME):$(DOCKER_TAG)$(RESET)"

.PHONY: docker-run
docker-run: ## 🏃 Run Docker container
	@echo "$(BLUE)🏃 Running Docker container...$(RESET)"
	@docker run -p 8080:8080 --env-file .env $(APP_NAME):$(DOCKER_TAG)

.PHONY: docker-clean
docker-clean: ## 🧹 Clean Docker images
	@echo "$(BLUE)🧹 Cleaning Docker images...$(RESET)"
	@docker rmi $(APP_NAME):$(DOCKER_TAG) 2>/dev/null || true
	@echo "$(GREEN)✅ Docker cleanup complete$(RESET)"

## 📊 Quality Commands
.PHONY: check
check: fmt vet test ## ✅ Run all quality checks (fmt, vet, test)

.PHONY: check-all
check-all: fmt vet lint security test-coverage ## 🎯 Run comprehensive quality checks

.PHONY: pre-commit
pre-commit: check-all ## 🔄 Run pre-commit checks

## 🎯 Development Workflows
.PHONY: dev
dev: deps fmt vet test build ## 🛠️  Full development workflow

.PHONY: ci
ci: deps check-all build build-example ## 🤖 CI/CD pipeline simulation

## 📱 Local Environment
.PHONY: setup
setup: ## 🛠️  Set up development environment
	@echo "$(BLUE)🛠️  Setting up development environment...$(RESET)"
	@echo "$(YELLOW)📋 Checking Go version...$(RESET)"
	@go version
	@echo "$(YELLOW)📦 Installing dependencies...$(RESET)"
	@make deps
	@echo "$(YELLOW)🧪 Running initial tests...$(RESET)"
	@make test-short
	@echo "$(YELLOW)🔨 Building application...$(RESET)"
	@make build
	@echo "$(GREEN)✅ Development environment ready!$(RESET)"
	@echo ""
	@echo "$(BLUE)🚀 Quick start:$(RESET)"
	@echo "  • Set AWS profile: export AWS_PROFILE=your-profile"
	@echo "  • Run application: make run"
	@echo "  • Run tests: make test"
	@echo "  • See all commands: make help"

.PHONY: env-example
env-example: ## 📝 Create example environment file
	@echo "$(BLUE)📝 Creating example .env file...$(RESET)"
	@echo "# GOV.UK Cost Dashboard Environment Variables" > .env.example
	@echo "# ==========================================" >> .env.example
	@echo "" >> .env.example
	@echo "# Server Configuration" >> .env.example
	@echo "PORT=8080" >> .env.example
	@echo "ENVIRONMENT=development" >> .env.example
	@echo "READ_TIMEOUT=30" >> .env.example
	@echo "WRITE_TIMEOUT=30" >> .env.example
	@echo "" >> .env.example
	@echo "# AWS Configuration" >> .env.example
	@echo "AWS_REGION=eu-west-2" >> .env.example
	@echo "AWS_PROFILE=your-profile-name" >> .env.example
	@echo "# AWS_ACCESS_KEY_ID=your_access_key" >> .env.example
	@echo "# AWS_SECRET_ACCESS_KEY=your_secret_key" >> .env.example
	@echo "# AWS_SESSION_TOKEN=your_session_token" >> .env.example
	@echo "# AWS_MFA_TOKEN=123456" >> .env.example
	@echo "" >> .env.example
	@echo "# GOV.UK Configuration" >> .env.example
	@echo "GOVUK_API_BASE_URL=https://www.gov.uk/api" >> .env.example
	@echo "# GOVUK_API_KEY=your_api_key" >> .env.example
	@echo "GOVUK_APPS_API_TIMEOUT=30s" >> .env.example
	@echo "GOVUK_APPS_API_CACHE_TTL=15m" >> .env.example
	@echo "GOVUK_APPS_API_RETRIES=3" >> .env.example
	@echo "" >> .env.example
	@echo "# Logging Configuration" >> .env.example
	@echo "LOG_LEVEL=info" >> .env.example
	@echo "LOG_FORMAT=json" >> .env.example
	@echo "$(GREEN)✅ Created .env.example$(RESET)"
	@echo "$(YELLOW)💡 Copy to .env and customize: cp .env.example .env$(RESET)"

## 📚 Documentation
.PHONY: docs
docs: ## 📚 Generate documentation
	@echo "$(BLUE)📚 Generating documentation...$(RESET)"
	@go doc -all ./... > docs.txt
	@echo "$(GREEN)✅ Documentation generated: docs.txt$(RESET)"

.PHONY: godoc
godoc: ## 🌐 Start local documentation server
	@echo "$(BLUE)🌐 Starting documentation server...$(RESET)"
	@echo "$(YELLOW)📖 Visit: http://localhost:6060/pkg/govuk-cost-dashboard/$(RESET)"
	@godoc -http=:6060

## 🆘 Help
.PHONY: help
help: ## 📖 Show this help message
	@echo "$(BLUE)📖 GOV.UK Cost Dashboard - Available Commands$(RESET)"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-20s$(RESET) %s\n", $$1, $$2}' | \
		sort
	@echo ""
	@echo "$(BLUE)🚀 Quick Start:$(RESET)"
	@echo "  make setup      - Set up development environment"
	@echo "  make run        - Run the application"
	@echo "  make test       - Run all tests"
	@echo ""
	@echo "$(BLUE)💡 Examples:$(RESET)"
	@echo "  AWS_PROFILE=my-profile make run"
	@echo "  make test-coverage"
	@echo "  make docker-build && make docker-run"

## 🔄 Maintenance
.PHONY: update-deps
update-deps: ## 🔄 Update all dependencies
	@echo "$(BLUE)🔄 Updating dependencies...$(RESET)"
	@go get -u ./...
	@go mod tidy
	@echo "$(GREEN)✅ Dependencies updated$(RESET)"

.PHONY: mod-graph
mod-graph: ## 📊 Show module dependency graph
	@echo "$(BLUE)📊 Module dependency graph:$(RESET)"
	@go mod graph

.PHONY: mod-why
mod-why: ## ❓ Explain why modules are needed (usage: make mod-why MODULE=github.com/pkg/name)
	@echo "$(BLUE)❓ Why is $(MODULE) needed:$(RESET)"
	@go mod why $(MODULE)

## 🎪 Fun Commands
.PHONY: gopher
gopher: ## 🐹 Show Go gopher
	@echo "$(BLUE)"
	@echo "        ,_---~~~~~----._         "
	@echo "     _,,_,*^____      _____\`*g, "
	@echo "    / __/ /'     ^.  /      \ ^@q"
	@echo "   [  @f | @))    |  | @))   l  0 _/"
	@echo "    \`/   \  ~____ / __ \_____/    \  "
	@echo "     |           _l__l_           I  "
	@echo "     }          [______]           I "
	@echo "     ]            | | |            |  "
	@echo "     ]             ~ ~             |  "
	@echo "     |                            |   "
	@echo "      |                           |   "
	@echo "$(RESET)"
	@echo "$(YELLOW)🐹 Happy coding with Go!$(RESET)"

.PHONY: stats
stats: ## 📈 Show project statistics
	@echo "$(BLUE)📈 Project Statistics:$(RESET)"
	@echo ""
	@echo "$(YELLOW)📁 Files:$(RESET)"
	@find . -name "*.go" -not -path "./vendor/*" | wc -l | sed 's/^/  Go files: /'
	@find . -name "*.md" | wc -l | sed 's/^/  Markdown files: /'
	@find . -name "*.json" | wc -l | sed 's/^/  JSON files: /'
	@echo ""
	@echo "$(YELLOW)📏 Lines of code:$(RESET)"
	@find . -name "*.go" -not -path "./vendor/*" -exec wc -l {} + | tail -n 1 | sed 's/^/  Go LOC: /'
	@echo ""
	@echo "$(YELLOW)📦 Dependencies:$(RESET)"
	@go list -m all | wc -l | sed 's/^/  Modules: /'