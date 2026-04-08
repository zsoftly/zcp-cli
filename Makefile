BINARY_NAME   := zcp
BUILD_DIR     := bin
CMD_DIR       := cmd/zcp
MODULE        := github.com/zsoftly/zcp-cli
VERSION_PKG   := $(MODULE)/internal/version

VERSION       := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS       := -ldflags "-s -w -X $(VERSION_PKG).Version=$(VERSION)"

GO            := go
GOFMT         := gofmt
GOVET         := $(GO) vet
GOTEST        := $(GO) test
PRETTIER      := $(shell command -v prettier 2>/dev/null || echo "npx prettier")

.DEFAULT_GOAL := help

##@ Build

.PHONY: all
all: fmt vet build ## Format, vet, and build for the current platform

.PHONY: build
build: ## Build for the current platform
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./$(CMD_DIR)
	@echo "Built: $(BUILD_DIR)/$(BINARY_NAME)"

.PHONY: dev
dev: build ## Alias for build (build for current platform)

.PHONY: build-all
build-all: build-linux build-darwin build-windows ## Build for all supported platforms

.PHONY: build-linux
build-linux: ## Build for Linux (amd64 and arm64)
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./$(CMD_DIR)
	@echo "Built: $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64"
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./$(CMD_DIR)
	@echo "Built: $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64"

.PHONY: build-darwin
build-darwin: ## Build for macOS (amd64 and arm64)
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./$(CMD_DIR)
	@echo "Built: $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64"
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./$(CMD_DIR)
	@echo "Built: $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64"

.PHONY: build-windows
build-windows: ## Build for Windows (amd64 and arm64)
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./$(CMD_DIR)
	@echo "Built: $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe"
	GOOS=windows GOARCH=arm64 CGO_ENABLED=0 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-arm64.exe ./$(CMD_DIR)
	@echo "Built: $(BUILD_DIR)/$(BINARY_NAME)-windows-arm64.exe"

##@ Test

.PHONY: test
test: ## Run all tests with verbose output
	$(GOTEST) -v ./...

.PHONY: test-race
test-race: ## Run all tests with race detector
	$(GOTEST) -race ./...

##@ Quality

.PHONY: fmt
fmt: ## Format all Go source files and Markdown docs
	$(GOFMT) -w .
	$(PRETTIER) --write '**/*.md' --prose-wrap preserve 2>/dev/null || true

.PHONY: fmt-check
fmt-check: ## Check formatting without writing (useful in CI)
	@test -z "$$($(GOFMT) -l .)" || { echo "Go files need formatting:"; $(GOFMT) -l .; exit 1; }
	$(PRETTIER) --check '**/*.md' --prose-wrap preserve 2>/dev/null || { echo "Markdown files need formatting — run: make fmt"; exit 1; }

.PHONY: vet
vet: ## Run go vet
	$(GOVET) ./...

.PHONY: tidy
tidy: ## Tidy go.mod and go.sum
	$(GO) mod tidy

.PHONY: lint
lint: ## Run staticcheck linter (install: go install honnef.co/go/tools/cmd/staticcheck@latest)
	@command -v staticcheck >/dev/null 2>&1 || { echo "staticcheck not found — install: go install honnef.co/go/tools/cmd/staticcheck@latest"; exit 1; }
	staticcheck ./...

##@ Install

.PHONY: install
install: build ## Install zcp to /usr/local/bin
	@echo "Installing $(BINARY_NAME) to /usr/local/bin/$(BINARY_NAME)..."
	install -m 0755 $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
	@echo "Installed: /usr/local/bin/$(BINARY_NAME)"

##@ Release

.PHONY: release-checksums
release-checksums: ## Generate SHA256 checksums for all binaries in bin/
	@echo "Generating SHA256 checksums for bin/*..."
	@cd $(BUILD_DIR) && sha256sum $(BINARY_NAME)-* > checksums.txt
	@echo "Written: $(BUILD_DIR)/checksums.txt"
	@cat $(BUILD_DIR)/checksums.txt

##@ Cleanup

.PHONY: clean
clean: ## Remove build artifacts
	@rm -rf $(BUILD_DIR)
	@echo "Cleaned: $(BUILD_DIR)/"

##@ Help

.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
