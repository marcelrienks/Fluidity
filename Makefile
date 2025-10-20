# Makefile for Fluidity project

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Binary names
AGENT_BINARY=fluidity-agent
SERVER_BINARY=fluidity-server

# Build directories
BUILD_DIR=build
AGENT_DIR=cmd/agent
SERVER_DIR=cmd/server

# Docker parameters
DOCKER_REGISTRY=
AGENT_IMAGE=$(DOCKER_REGISTRY)fluidity-agent
SERVER_IMAGE=$(DOCKER_REGISTRY)fluidity-server
VERSION=latest

.PHONY: all build clean test deps docker-build docker-push help

# Default target
all: clean deps build

# Build both binaries
build: build-agent build-server

# Build agent binary
build-agent:
	@echo "Building agent..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux $(GOBUILD) -a -installsuffix cgo -o $(BUILD_DIR)/$(AGENT_BINARY) ./$(AGENT_DIR)

# Build server binary
build-server:
	@echo "Building server..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux $(GOBUILD) -a -installsuffix cgo -o $(BUILD_DIR)/$(SERVER_BINARY) ./$(SERVER_DIR)

# Build for Windows
build-windows: build-agent-windows build-server-windows

build-agent-windows:
	@echo "Building agent for Windows..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=windows $(GOBUILD) -a -installsuffix cgo -o $(BUILD_DIR)/$(AGENT_BINARY).exe ./$(AGENT_DIR)

build-server-windows:
	@echo "Building server for Windows..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=windows $(GOBUILD) -a -installsuffix cgo -o $(BUILD_DIR)/$(SERVER_BINARY).exe ./$(SERVER_DIR)

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@$(GOCLEAN)
	@rm -rf $(BUILD_DIR)

# Run tests
test:
	@echo "Running tests..."
	@$(GOTEST) -v ./...

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	@$(GOMOD) download
	@$(GOMOD) tidy

# Docker build
docker-build: docker-build-agent docker-build-server

docker-build-agent:
	@echo "Building agent Docker image..."
	@docker build -f deployments/agent/Dockerfile -t $(AGENT_IMAGE):$(VERSION) .

docker-build-server:
	@echo "Building server Docker image..."
	@docker build -f deployments/server/Dockerfile -t $(SERVER_IMAGE):$(VERSION) .

# Docker push
docker-push: docker-push-agent docker-push-server

docker-push-agent:
	@echo "Pushing agent Docker image..."
	@docker push $(AGENT_IMAGE):$(VERSION)

docker-push-server:
	@echo "Pushing server Docker image..."
	@docker push $(SERVER_IMAGE):$(VERSION)

# Run agent locally
run-agent:
	@echo "Running agent locally..."
	@$(GOBUILD) -o $(BUILD_DIR)/$(AGENT_BINARY) ./$(AGENT_DIR)
	@./$(BUILD_DIR)/$(AGENT_BINARY) --config ./configs/agent.yaml

# Run server locally
run-server:
	@echo "Running server locally..."
	@$(GOBUILD) -o $(BUILD_DIR)/$(SERVER_BINARY) ./$(SERVER_DIR)
	@./$(BUILD_DIR)/$(SERVER_BINARY) --config ./configs/server.yaml

# Format code
fmt:
	@echo "Formatting code..."
	@$(GOCMD) fmt ./...

# Lint code (requires golangci-lint)
lint:
	@echo "Linting code..."
	@golangci-lint run

# Generate certificates (development only)
gen-certs:
	@echo "Generating development certificates..."
	@./scripts/generate-certs.sh

# Local build and run targets
build-local: clean deps build

run-agent-local:
	@echo "Running agent locally (local config)..."
	@$(GOBUILD) -o $(BUILD_DIR)/$(AGENT_BINARY) ./$(AGENT_DIR)
	@./$(BUILD_DIR)/$(AGENT_BINARY) --config ./configs/agent.local.yaml

run-server-local:
	@echo "Running server locally (local config)..."
	@$(GOBUILD) -o $(BUILD_DIR)/$(SERVER_BINARY) ./$(SERVER_DIR)
	@./$(BUILD_DIR)/$(SERVER_BINARY) --config ./configs/server.local.yaml

# Help
help:
	@echo "Available targets:"
	@echo "  all           - Clean, download deps, and build both binaries"
	@echo "  build         - Build both agent and server binaries"
	@echo "  build-agent   - Build agent binary only"
	@echo "  build-server  - Build server binary only"
	@echo "  build-windows - Build Windows binaries"
	@echo "  clean         - Remove build artifacts"
	@echo "  test          - Run tests"
	@echo "  deps          - Download and tidy dependencies"
	@echo "  docker-build  - Build Docker images"
	@echo "  docker-push   - Push Docker images"
	@echo "  run-agent     - Build and run agent locally"
	@echo "  run-server    - Build and run server locally"
	@echo "  fmt           - Format Go code"
	@echo "  lint          - Lint Go code"
	@echo "  gen-certs     - Generate development certificates"
	@echo "  help          - Show this help message"
	@echo "  build-local   - Clean, download deps, and build for local debugging"
	@echo "  run-agent-local - Build and run agent with local config"
	@echo "  run-server-local - Build and run server with local config"