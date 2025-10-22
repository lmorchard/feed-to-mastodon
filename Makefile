.PHONY: build test clean run lint format fmt setup

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "v0.0.1")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

build:
	@echo "Building for $(shell go env GOOS)/$(shell go env GOARCH)"
	go build -o feed-to-mastodon ./cmd/feed-to-mastodon

test:
	go test ./internal/...

coverage:
	go test -coverprofile=coverage.out ./internal/...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

clean:
	rm -f feed-to-mastodon
	rm -f feeds.db
	rm -f coverage.out coverage.html

run: build
	./feed-to-mastodon

format fmt:
	@GOPATH=$$(go env GOPATH); \
	if [ ! -f "$$GOPATH/bin/gofumpt" ]; then \
		echo "gofumpt not found. Please install it: go install mvdan.cc/gofumpt@latest"; \
		exit 1; \
	fi
	go fmt ./...
	$$(go env GOPATH)/bin/gofumpt -w .

lint:
	@GOPATH=$$(go env GOPATH); \
	if [ ! -f "$$GOPATH/bin/golangci-lint" ]; then \
		echo "golangci-lint not found. Please install it: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		exit 1; \
	fi
	$$(go env GOPATH)/bin/golangci-lint run --timeout=5m

setup:
	@echo "Installing development tools..."
	go install mvdan.cc/gofumpt@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Tools installed successfully!"
