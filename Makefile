.PHONY: test lint build vuln run tidy

GO ?= go
GOVULNCHECK_VERSION ?= v1.9.0
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

test:
	$(GO) test -race -count=1 ./...

lint:
	golangci-lint run

build:
	CGO_ENABLED=0 $(GO) build -trimpath -ldflags="-s -w -X main.version=$(VERSION)" -o dist/netbird-mcp ./cmd/netbird-mcp

vuln:
	$(GO) run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION) ./...

run:
	$(GO) run ./cmd/netbird-mcp

tidy:
	$(GO) mod tidy
