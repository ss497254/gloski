.PHONY: proto dev-server dev-web dev-agent build-server build-agent build-agents build-web build clean install install-agent test lint format format-check help

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

dev-agent: ## Run the device agent in development mode (requires master URL)
	cd server && GLOSKI_LOG_LEVEL=debug go run $(LDFLAGS) ./cmd/gloski-agent

dev: ## Run both backend and frontend (requires tmux)
	@tmux new-session -d -s gloski 'make dev-server' \; split-window -h 'make dev-web' \; attach

# Build
AGENT_BUILDS := server/internal/agentdist/builds
# No build time here: an agent updates itself whenever its binary differs from
# the master's copy, so identical source must give an identical binary.
AGENT_LDFLAGS := -trimpath -ldflags "-s -w -X main.version=$(VERSION)"

build-agents: ## Build the agents the master hands to devices (amd64, arm64, arm)
	cd server && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(AGENT_LDFLAGS) -o ../$(AGENT_BUILDS)/gloski-agent-linux-amd64 ./cmd/gloski-agent
	cd server && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(AGENT_LDFLAGS) -o ../$(AGENT_BUILDS)/gloski-agent-linux-arm64 ./cmd/gloski-agent
	cd server && CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=6 go build $(AGENT_LDFLAGS) -o ../$(AGENT_BUILDS)/gloski-agent-linux-arm ./cmd/gloski-agent
	@echo "$(GREEN)Built agents for amd64, arm64 and arm$(NC)"

build-server: build-agents ## Build master binary (agents are embedded in it)
	@mkdir -p bin
	cd server && CGO_ENABLED=0 go build $(LDFLAGS) -o ../bin/gloski ./cmd/gloski
	@echo "$(GREEN)Built bin/gloski$(NC)"

build-agent: ## Build a standalone agent binary for this machine
	@mkdir -p bin
	cd server && CGO_ENABLED=0 go build $(LDFLAGS) -o ../bin/gloski-agent ./cmd/gloski-agent
	@echo "$(GREEN)Built bin/gloski-agent$(NC)"

build-web: ## Build frontend for production
	cd web && bun run build
	@echo "$(GREEN)Built web/dist/$(NC)"

build: build-server build-web ## Build master (with embedded agents) and frontend

# Installation
install: build-server ## Install the master to ~/.local/bin
	cp bin/gloski ~/.local/bin/
	@echo "$(GREEN)Installed to ~/.local/bin/gloski$(NC)"

install-agent: build-agent ## Install only the device agent binary
	cp bin/gloski-agent ~/.local/bin/
	@echo "$(GREEN)Installed to ~/.local/bin/gloski-agent$(NC)"

install-hooks: ## Enable the checked-in pre-commit review hook for this clone
	@sh scripts/install-git-hooks.sh

install-service: ## Install systemd service (user)
	mkdir -p ~/.config/systemd/user
	mkdir -p ~/.config/gloski
	cp deploy/gloski.service ~/.config/systemd/user/
	echo "{}" > ~/.config/gloski/config.json
	chmod 600 ~/.config/gloski/config.json
	echo "$(GREEN)Created config file at ~/.config/gloski/config.json. Please edit it before starting the service.$(NC)"
	systemctl --user daemon-reload
	@echo "$(GREEN)Service installed. Enable with: systemctl --user enable --now gloski$(NC)"

proto: ## Regenerate gRPC code from agent.proto
	cd server && protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative proto/agentv1/agent.proto

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
	rm -rf bin/ web/dist/ server/coverage.out server/coverage.html $(AGENT_BUILDS)/gloski-agent-*
