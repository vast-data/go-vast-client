# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOLINT=golangci-lint
MKDOCS ?= mkdocs
ADDR ?= localhost:8000

# Binary names
BINARY_NAME=go-vast-client
BINARY_UNIX=$(BINARY_NAME)_unix

# Build info
VERSION ?= $(shell git describe --tags --always --dirty)
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
COMMIT=$(shell git rev-parse --short HEAD)

# Directories
EXAMPLES_DIR=examples
COVERAGE_DIR=coverage

.PHONY: all build test clean deps lint fmt vet coverage examples help

all: clean deps fmt lint test build ## Run all main targets

help: ## Display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# Development
deps: ## Download and install dependencies
	$(GOMOD) download
	$(GOMOD) tidy
	$(GOMOD) verify

fmt: ## Format Go code
	$(GOCMD) fmt ./...

vet: ## Run go vet
	$(GOCMD) vet ./...

lint: ## Run golangci-lint
	$(GOLINT) run

# Testing
test: ## Run tests
	$(GOTEST) -v -race ./...

test-short: ## Run tests with short flag
	$(GOTEST) -v -short ./...

test-coverage: ## Run tests with coverage
	@mkdir -p $(COVERAGE_DIR)
	$(GOTEST) -v -race -coverprofile=$(COVERAGE_DIR)/coverage.out ./...
	$(GOCMD) tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html

coverage-report: test-coverage ## Generate and open coverage report
	$(GOCMD) tool cover -func=$(COVERAGE_DIR)/coverage.out
	@echo "Coverage report generated: $(COVERAGE_DIR)/coverage.html"

# Building
build: ## Build the library (test build)
	$(GOBUILD) -v ./...

# Cleaning
clean: ## Clean build artifacts
	$(GOCLEAN)
	rm -rf $(COVERAGE_DIR)
	rm -rf $(EXAMPLES_DIR)/bin

# CI/CD helpers
ci-deps: ## Install CI dependencies
	$(GOCMD) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

ci-test: ## Run CI tests
	$(GOTEST) -v -race -coverprofile=coverage.out ./...

# Security
security: ## Run security checks
	$(GOCMD) install golang.org/x/vuln/cmd/govulncheck@latest
	govulncheck ./...

# Documentation
docs: ## Generate documentation
	$(GOCMD) doc -all

# Git helpers
tag: ## Create a new git tag (usage: make tag VERSION=v1.0.0)
	@if [ -z "$(VERSION)" ]; then echo "VERSION is required. Usage: make tag VERSION=v1.0.0"; exit 1; fi
	git tag -a $(VERSION) -m "Release $(VERSION)"
	git push origin $(VERSION)

# Check if required tools are installed
check-tools: ## Check if required tools are installed
	@command -v golangci-lint >/dev/null 2>&1 || { echo >&2 "golangci-lint is required but not installed. Run: make ci-deps"; exit 1; }
	@command -v go >/dev/null 2>&1 || { echo >&2 "Go is required but not installed."; exit 1; }

docs-build:
	$(MKDOCS) build --clean --strict

docs-serve:
	$(MKDOCS) serve --dev-addr $(ADDR)

docs-deploy:
	$(MKDOCS) gh-deploy --force
