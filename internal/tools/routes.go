package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/koinsaari/netbird-mcp/internal/netbird"
)

type routeAPI interface {
	ListRoutes(ctx context.Context) ([]netbird.Route, error)
}

type listRoutesInput struct{}

type listRoutesOutput struct {
	Routes []routeOutput `json:"routes"`
}

// routeOutput passes peer and group IDs through unresolved.
type routeOutput struct {
	ID                  string   `json:"id"`
	NetworkID           string   `json:"network_id"`
	Description         string   `json:"description,omitempty"`
	Enabled             bool     `json:"enabled"`
	NetworkType         string   `json:"network_type"`
	Network             string   `json:"network,omitempty"`
	Domains             []string `json:"domains"`
	Peer                string   `json:"peer,omitempty"`
	PeerGroups          []string `json:"peer_groups"`
	Groups              []string `json:"groups"`
	AccessControlGroups []string `json:"access_control_groups"`
	Metric              int      `json:"metric"`
	Masquerade          bool     `json:"masquerade"`
	KeepRoute           bool     `json:"keep_route"`
	SkipAutoApply       bool     `json:"skip_auto_apply"`
}

// listRoutes builds the list_routes tool definition and its handler.
func listRoutes(c routeAPI) (*mcp.Tool, mcp.ToolHandlerFor[listRoutesInput, listRoutesOutput]) {
	tool := &mcp.Tool{
		Name:        "list_routes",
		Description: "Lists all network routes in the NetBird account: network or domains, routing peer or peer groups, access control groups, metric and masquerade.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true},
	}
	handler := func(ctx context.Context, _ *mcp.CallToolRequest, _ listRoutesInput) (*mcp.CallToolResult, listRoutesOutput, error) {
		routes, err := c.ListRoutes(ctx)
		if err != nil {
			return nil, listRoutesOutput{}, err
		}
		return nil, toListRoutesOutput(routes), nil
	}
	return tool, handler
}

func toListRoutesOutput(routes []netbird.Route) listRoutesOutput {
	out := listRoutesOutput{Routes: make([]routeOutput, 0, len(routes))}
	for _, r := range routes {
		out.Routes = append(out.Routes, routeOutput{
			ID: r.ID, NetworkID: r.NetworkID, Description: r.Description, Enabled: r.Enabled,
			NetworkType: r.NetworkType, Network: r.Network, Domains: nonNil(r.Domains), Peer: r.Peer,
			PeerGroups: nonNil(r.PeerGroups), Groups: nonNil(r.Groups),
			AccessControlGroups: nonNil(r.AccessControlGroups), Metric: r.Metric,
			Masquerade: r.Masquerade, KeepRoute: r.KeepRoute, SkipAutoApply: r.SkipAutoApply,
		})
	}
	return out
}

// nonNil returns s, or an empty slice when s is nil, so it encodes as [].
func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
