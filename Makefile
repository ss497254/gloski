.PHONY: dev-server dev-web build-server build-web build clean install test lint format format-check help

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME ?= $(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)"

# Colors for output
GREEN := \033[0;32m
NC := \033[0m

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "$(GREEN)%-15s$(NC) %s\n", $$1, $$2}'

# Development
dev-server: ## Run backend in development mode
	cd server && GLOSKI_LOG_LEVEL=debug go run $(LDFLAGS) ./cmd/gloski

dev-web: ## Run frontend in development mode
	cd web && bun run dev

dev: ## Run both backend and frontend (requires tmux)
	@tmux new-session -d -s gloski 'make dev-server' \; split-window -h 'make dev-web' \; attach

# Build
build-server: ## Build backend binary
	@mkdir -p bin
	cd server && CGO_ENABLED=0 go build $(LDFLAGS) -o ../bin/gloski ./cmd/gloski
	@echo "$(GREEN)Built bin/gloski$(NC)"

build-web: ## Build frontend for production
	cd web && bun run build
	@echo "$(GREEN)Built web/dist/$(NC)"

build: build-server build-web ## Build both backend and frontend

# Installation
install: build-server ## Install binary to /usr/local/bin
	sudo cp bin/gloski /usr/local/bin/
	@echo "$(GREEN)Installed to /usr/local/bin/gloski$(NC)"

install-service: ## Install systemd service (user)
	mkdir -p ~/.config/systemd/user
	cp deploy/gloski.service ~/.config/systemd/user/
	systemctl --user daemon-reload
	@echo "$(GREEN)Service installed. Enable with: systemctl --user enable --now gloski$(NC)"

# Testing
test: ## Run tests
	cd server && go test -v ./...

test-coverage: ## Run tests with coverage
	cd server && go test -coverprofile=coverage.out ./...
	cd server && go tool cover -html=coverage.out -o coverage.html

lint: ## Run linters
	cd server && golangci-lint run ./...
	cd web && bun run lint

format: ## Format all code (Go and TypeScript)
	cd server && go fmt ./...
	cd web && bun run format
	@echo "$(GREEN)Formatted all files$(NC)"

format-check: ## Check code formatting without making changes
	cd server && test -z "$$(gofmt -l .)" || (gofmt -l . && exit 1)
	cd web && bun run format:check

clean: ## Clean build artifacts
	rm -rf bin/ web/dist/ server/coverage.out server/coverage.html
