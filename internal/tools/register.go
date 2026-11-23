package tools

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Client interface {
	peerAPI
	groupAPI
	policyAPI
	userAPI
	routeAPI
	networkAPI
	postureCheckAPI
	dnsAPI
}

func Register(s *mcp.Server, c Client, logger *slog.Logger) {
	tool, h := listPeers(c)
	mcp.AddTool(s, tool, logged(logger, tool.Name, h))
	getTool, getH := getPeer(c)
	mcp.AddTool(s, getTool, logged(logger, getTool.Name, getH))
	groupsTool, groupsH := listGroups(c)
	mcp.AddTool(s, groupsTool, logged(logger, groupsTool.Name, groupsH))
	getGroupTool, getGroupH := getGroup(c)
	mcp.AddTool(s, getGroupTool, logged(logger, getGroupTool.Name, getGroupH))
	policiesTool, policiesH := listPolicies(c)
	mcp.AddTool(s, policiesTool, logged(logger, policiesTool.Name, policiesH))
	usersTool, usersH := listUsers(c)
	mcp.AddTool(s, usersTool, logged(logger, usersTool.Name, usersH))
	routesTool, routesH := listRoutes(c)
	mcp.AddTool(s, routesTool, logged(logger, routesTool.Name, routesH))
	accessTool, accessH := getPeerAccess(c)
	mcp.AddTool(s, accessTool, logged(logger, accessTool.Name, accessH))
	networksTool, networksH := listNetworks(c)
	mcp.AddTool(s, networksTool, logged(logger, networksTool.Name, networksH))
	postureTool, postureH := listPostureChecks(c)
	mcp.AddTool(s, postureTool, logged(logger, postureTool.Name, postureH))
	dnsTool, dnsH := listDNS(c)
	mcp.AddTool(s, dnsTool, logged(logger, dnsTool.Name, dnsH))
}

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
