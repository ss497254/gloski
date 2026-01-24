.PHONY: dev-server dev-web build-server build-web build clean install test lint help

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
	cd web && npm run dev

dev: ## Run both backend and frontend (requires tmux)
	@tmux new-session -d -s gloski 'make dev-server' \; split-window -h 'make dev-web' \; attach

# Build
build-server: ## Build backend binary
	@mkdir -p bin
	cd server && CGO_ENABLED=0 go build $(LDFLAGS) -o ../bin/gloski ./cmd/gloski
	@echo "$(GREEN)Built bin/gloski$(NC)"

build-web: ## Build frontend for production
	cd web && npm run build
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
	cd web && npm run lint

# Utilities
gen-password: ## Generate password hash (usage: make gen-password PASSWORD=mypassword)
	@cd server && go run ./cmd/gloski -gen-password "$(PASSWORD)"

clean: ## Clean build artifacts
	rm -rf bin/ web/dist/ server/coverage.out server/coverage.html

# Docker
docker-build: ## Build Docker image
	docker build -t gloski:$(VERSION) .

docker-run: ## Run Docker container
	docker run -d -p 8080:8080 --name gloski gloski:$(VERSION)