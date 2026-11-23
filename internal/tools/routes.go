package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/koinsaari/netbird-mcp/internal/netbird"
)

type routeAPI interface {
	ListRoutes(ctx context.Context) ([]netbird.Route, error)
}

type listRoutesAPI interface {
	routeAPI
	groupAPI
	peerAPI
}

type listRoutesInput struct{}

type listRoutesOutput struct {
	Routes     []routeOutput `json:"routes"`
	NamesError string        `json:"names_error,omitempty"`
}

type routeOutput struct {
	ID                  string      `json:"id"`
	NetworkID           string      `json:"network_id"`
	Description         string      `json:"description,omitempty"`
	Enabled             bool        `json:"enabled"`
	NetworkType         string      `json:"network_type"`
	Network             string      `json:"network,omitempty"`
	Domains             []string    `json:"domains"`
	Peer                *refOutput  `json:"peer,omitempty"`
	PeerGroups          []refOutput `json:"peer_groups"`
	Groups              []refOutput `json:"groups"`
	AccessControlGroups []refOutput `json:"access_control_groups"`
	Metric              int         `json:"metric"`
	Masquerade          bool        `json:"masquerade"`
	KeepRoute           bool        `json:"keep_route"`
	SkipAutoApply       bool        `json:"skip_auto_apply"`
}

func listRoutes(c listRoutesAPI) (*mcp.Tool, mcp.ToolHandlerFor[listRoutesInput, listRoutesOutput]) {
	tool := &mcp.Tool{
		Name:        "list_routes",
		Description: "Lists all network routes in the NetBird account: network or domains, routing peer or peer groups, access control groups, metric and masquerade. Group and peer references include names; names_error appears when names cannot be fetched.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true},
	}
	handler := func(ctx context.Context, _ *mcp.CallToolRequest, _ listRoutesInput) (*mcp.CallToolResult, listRoutesOutput, error) {
		var routes []netbird.Route
		names, err := resolveNames(ctx, c, c, func(ctx context.Context) (err error) {
			routes, err = c.ListRoutes(ctx)
			return err
		})
		if err != nil {
			return nil, listRoutesOutput{}, err
		}
		return nil, toListRoutesOutput(routes, names), nil
	}
	return tool, handler
}

func toListRoutesOutput(routes []netbird.Route, names refNames) listRoutesOutput {
	out := listRoutesOutput{Routes: make([]routeOutput, 0, len(routes)), NamesError: names.err}
	for _, r := range routes {
		out.Routes = append(out.Routes, routeOutput{
			ID: r.ID, NetworkID: r.NetworkID, Description: r.Description, Enabled: r.Enabled,
			NetworkType: r.NetworkType, Network: r.Network, Domains: nonNil(r.Domains), Peer: toPeerRef(r.Peer, names.peers),
			PeerGroups: toGroupRefs(r.PeerGroups, names.groups), Groups: toGroupRefs(r.Groups, names.groups),
			AccessControlGroups: toGroupRefs(r.AccessControlGroups, names.groups), Metric: r.Metric,
			Masquerade: r.Masquerade, KeepRoute: r.KeepRoute, SkipAutoApply: r.SkipAutoApply,
		})
	}
	return out
}
