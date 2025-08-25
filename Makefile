# Variables
EXECUTABLE=chatgbt
BINARY_DIR=bin
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
GO_VERSION=$(shell go version | cut -d' ' -f3)
LDFLAGS=-ldflags="-s -w -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GoVersion=$(GO_VERSION)"

# Go related variables
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt

.PHONY: all help build build-cli build-web build-prod clean run-cli run-web install-deps generate test fmt vet

all: test build ## Run tests and build

help: ## Show this help message
	@echo "ChatGBT Build System"
	@echo "==================="
	@echo ""
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Variables:"
	@echo "  VERSION=$(VERSION)"
	@echo "  EXECUTABLE=$(EXECUTABLE)"

# Build targets
build: generate ## Build the application (with template generation)
	$(GOBUILD) -o $(EXECUTABLE) .

build-cli: ## Build for CLI mode only (no template generation)
	$(GOBUILD) -o $(EXECUTABLE) .

build-web: generate ## Build for web mode (with template generation)
	$(GOBUILD) -o $(EXECUTABLE) .

build-prod: generate ## Build for production with optimizations
	mkdir -p $(BINARY_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/$(EXECUTABLE) .

# Development targets
run-cli: build-cli ## Run in CLI mode
	./$(EXECUTABLE) cli

run-web: build-web ## Run in web mode
	./$(EXECUTABLE) web

# Quality assurance targets
fmt: ## Format Go code and templates
	$(GOFMT) ./...
	go tool templ fmt .

vet: ## Run go vet
	$(GOCMD) vet ./...

test: fmt vet ## Run tests with formatting and vetting
	$(GOMOD) tidy
	$(GOTEST) ./... -v

install-deps: ## Install all dependencies and tools
	$(GOMOD) tidy
	$(GOMOD) download
	@echo "Installing development tools..."
	go get -tool github.com/a-h/templ/cmd/templ@latest

generate: ## Generate templates
	go tool templ generate

# Utility targets
clean: ## Clean build artifacts
	$(GOCLEAN)
	rm -f $(EXECUTABLE)
	rm -rf $(BINARY_DIR)

# Release and versioning targets
release: clean test build-prod ## Create a release build
	@echo "Building release $(VERSION)..."
	@echo "Binary created: $(BINARY_DIR)/$(EXECUTABLE)"
	@echo "Size: $$(du -h $(BINARY_DIR)/$(EXECUTABLE) | cut -f1)"

upstream: check-env ## Make sure you TAG correctly. E.g. export TAG=0.1.0
	git add .
	git commit -m "Bump to version ${TAG}"
	git tag -a -m "Bump to version ${TAG}" v${TAG}
	git push --follow-tags

check-env: ## Check if TAG variable is set
ifndef TAG
	$(error TAG is undefined)
endif
	@echo "TAG is ${TAG}"

tag: ## Create a git tag
	git tag <tagname>

# Info targets
info: ## Show build information
	@echo "Build Information:"
	@echo "=================="
	@echo "Version:     $(VERSION)"
	@echo "Build Time:  $(BUILD_TIME)"
	@echo "Go Version:  $(GO_VERSION)"
	@echo "Executable:  $(EXECUTABLE)"
	@echo "Binary Dir:  $(BINARY_DIR)"