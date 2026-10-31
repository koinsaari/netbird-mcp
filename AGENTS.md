# netbird-mcp

Model Context Protocol server that exposes the NetBird management API as MCP tools.

## NetBird API

- Management REST API at `https://api.netbird.io/api`, or a self-hosted URL.
- Auth header: `Authorization: Token <PAT>`.
- Resources: peers, groups, policies, networks, routes, setup keys, users, DNS, posture checks, events.

## Layout

- `cmd/netbird-mcp/` wiring only (`main.go`).
- `internal/` all other code.

## Commands

- `make test`
- `make lint`
- `make build`

## Rules

- `gofmt`, `go vet` and `golangci-lint` must be clean.
- Wrap errors with context (`fmt.Errorf("...: %w", err)`). `ctx context.Context` is the first parameter of anything doing I/O.
- Never log or return tokens (the NetBird PAT) in logs, errors, or tool output.
- Tools do only what was requested; validate and escape arguments used in API paths.
- Stdlib first; no new dependency without a reason.
- Conventional commits (`feat:`, `fix:`, `chore:`, ...).
