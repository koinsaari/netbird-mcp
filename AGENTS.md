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
- Layers: `tools` → `netbird` client. Only `tools` imports the MCP SDK; `netbird` knows nothing of MCP.
- Each tool is a closure `func xTool(c resourceAPI) (*mcp.Tool, handler)`, registered in `Register` through `logged`. One small interface per API resource (`peerAPI`), declared in `tools`.
- API structs (`netbird`) mirror the OpenAPI schema; tool output structs (`tools`) are separate and trimmed for the model. Lists are never `null`; optional API fields are omitted, not zeroed.
- Read-only tools set `ReadOnlyHint` and `IdempotentHint`.
- The `netbird` client never puts the token or the full URL in errors; API error messages are sanitized.
- Every new MCP tool has tests: output mapping, an in-memory MCP call, and a check that the token is absent from output and logs.
- Conventional commits (`feat:`, `fix:`, `chore:`, ...).
