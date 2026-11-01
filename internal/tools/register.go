// Package tools defines the MCP tools exposed by the server.
package tools

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Client is the NetBird API surface the tools need.
type Client interface {
	peerAPI
}

// Register adds all tools to s. Every call is logged with tool name, duration
// and error status. Arguments are not logged, but error text may echo them.
func Register(s *mcp.Server, c Client, logger *slog.Logger) {
	tool, h := listPeers(c)
	mcp.AddTool(s, tool, logged(logger, tool.Name, h))
	getTool, getH := getPeer(c)
	mcp.AddTool(s, getTool, logged(logger, getTool.Name, getH))
}

// logged wraps h to log each call (full error at Warn) and to turn handler
// errors into the model-facing text from toolMessage.
func logged[In, Out any](logger *slog.Logger, name string, h mcp.ToolHandlerFor[In, Out]) mcp.ToolHandlerFor[In, Out] {
	return func(ctx context.Context, req *mcp.CallToolRequest, in In) (*mcp.CallToolResult, Out, error) {
		start := time.Now()
		res, out, err := h(ctx, req, in)
		attrs := []any{"tool", name, "duration", time.Since(start), "is_error", err != nil}
		if err == nil {
			logger.InfoContext(ctx, "tool call", attrs...)
			return res, out, nil
		}
		logger.WarnContext(ctx, "tool call failed", append(attrs, "error", err)...)
		var zero Out
		return nil, zero, errors.New(toolMessage(err))
	}
}
