.PHONY: build run docker-build docker-push deploy clean test

# Variables
IMAGE_CONTROLLER ?= ghcr.io/adiii717/kube-ai-sre-agent-controller
IMAGE_ANALYZER ?= ghcr.io/adiii717/kube-ai-sre-agent-analyzer
VERSION ?= latest

# Build binaries
build:
	@echo "Building controller..."
	@go build -o bin/controller ./cmd/controller
	@echo "Building analyzer..."
	@go build -o bin/analyzer ./cmd/analyzer

# Run locally (for development)
run:
	@go run ./cmd/controller

# Run tests
test:
	@go test -v ./...

# Build Docker images
docker-build:
	@echo "Building controller image..."
	@docker build -t $(IMAGE_CONTROLLER):$(VERSION) -f Dockerfile.controller .
	@echo "Building analyzer image..."
	@docker build -t $(IMAGE_ANALYZER):$(VERSION) -f Dockerfile.analyzer .

# Push Docker images
docker-push:
	@docker push $(IMAGE_CONTROLLER):$(VERSION)
	@docker push $(IMAGE_ANALYZER):$(VERSION)

# Deploy to Kubernetes
deploy:
	@helm upgrade --install sre-agent ./helm/kube-ai-sre-agent

# Uninstall from Kubernetes
undeploy:
	@helm uninstall sre-agent

# Clean build artifacts
clean:
	@rm -rf bin/

# Lint
lint:
	@golangci-lint run

# Format code
fmt:
	@go fmt ./...
