# ABOUTME: This Makefile provides common development tasks for the gotestshow project
# ABOUTME: It includes targets for building, testing, linting, and formatting the code

.PHONY: help
help: ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

.PHONY: lint
lint: ## Run basic Go linters
	@echo "Running gofmt..."
	@gofmt -l . | grep -v vendor | grep -v .sixeight | grep -v example/broken | tee /tmp/gofmt.out
	@test ! -s /tmp/gofmt.out || (echo "Files need formatting. Run 'make fmt'" && exit 1)
	@echo "Running go vet..."
	@go vet `go list ./... | grep -v example/broken`
	@echo "Lint check completed!"

.PHONY: fmt
fmt: ## Format code using gofmt
	gofmt -s -w .

.PHONY: test
test: ## Run tests
	go test -v ./...

.PHONY: test-cover
test-cover: ## Run tests with coverage
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

.PHONY: build
build: ## Build the application
	go build -o gotestshow .

.PHONY: clean
clean: ## Clean build artifacts
	rm -f gotestshow coverage.out coverage.html /tmp/gofmt.out

.PHONY: mod-tidy
mod-tidy: ## Tidy go modules
	go mod tidy

.PHONY: all
all: fmt lint test build ## Run all checks and build