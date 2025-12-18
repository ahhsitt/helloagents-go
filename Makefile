.PHONY: build test lint cover clean deps fmt vet

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOVET=$(GOCMD) vet
GOFMT=$(GOCMD) fmt
GOMOD=$(GOCMD) mod
GOGET=$(GOCMD) get

# Build parameters
BINARY_NAME=helloagents
BINARY_DIR=bin

# Test parameters
COVERAGE_FILE=coverage.out
COVERAGE_HTML=coverage.html

all: deps lint test build

# Download dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Build the project
build:
	mkdir -p $(BINARY_DIR)
	$(GOBUILD) -o $(BINARY_DIR)/$(BINARY_NAME) ./...

# Run tests
test:
	$(GOTEST) -v -race ./...

# Run tests with coverage
cover:
	$(GOTEST) -v -race -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./...
	$(GOCMD) tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@echo "Coverage report: $(COVERAGE_HTML)"

# Run linter
lint:
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run ./...

# Format code
fmt:
	$(GOFMT) ./...

# Run go vet
vet:
	$(GOVET) ./...

# Clean build artifacts
clean:
	rm -rf $(BINARY_DIR)
	rm -f $(COVERAGE_FILE) $(COVERAGE_HTML)
	$(GOCMD) clean -cache

# Run examples
run-simple:
	$(GOCMD) run examples/simple/main.go

run-react:
	$(GOCMD) run examples/react/main.go

# Development helpers
dev: deps fmt vet lint test

# CI pipeline
ci: deps lint test cover
