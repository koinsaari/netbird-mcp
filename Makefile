.PHONY: test lint build run tidy

GO ?= go
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

test:
	$(GO) test -race -count=1 ./...

lint:
	golangci-lint run

build:
	CGO_ENABLED=0 $(GO) build -trimpath -ldflags="-s -w -X main.version=$(VERSION)" -o dist/netbird-mcp ./cmd/netbird-mcp

run:
	$(GO) run ./cmd/netbird-mcp

tidy:
	$(GO) mod tidy
