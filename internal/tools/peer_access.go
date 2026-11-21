package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/sync/errgroup"

	"github.com/koinsaari/netbird-mcp/internal/netbird"
)

type peerAccessAPI interface {
	peerAPI
	policyAPI
	routeAPI
	networkAPI
}

type getPeerAccessInput struct {
	PeerID string `json:"peer_id" jsonschema:"the 20-character NetBird peer ID, as returned by list_peers"`
}

func getPeerAccess(c peerAccessAPI) (*mcp.Tool, mcp.ToolHandlerFor[getPeerAccessInput, peerAccessOutput]) {
	tool := &mcp.Tool{
		Name:        "get_peer_access",
		Description: "Describes one NetBird peer's access: the access policies whose rules apply to it (with direction and the other side), the network routes it is part of, and the networks it routes for or can reach (the routers it is part of and the resources it can reach, with the policies granting that). Disabled policies, rules, routes and routers are included with their enabled flag. Only enabled accept rules of enabled policies count for network resource reach. Posture checks are not evaluated.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true},
	}
	handler := func(ctx context.Context, _ *mcp.CallToolRequest, in getPeerAccessInput) (*mcp.CallToolResult, peerAccessOutput, error) {
		if !validID(in.PeerID) {
			return nil, peerAccessOutput{}, errInvalidPeerID
		}
		var (
			peer      netbird.Peer
			policies  []netbird.Policy
			routes    []netbird.Route
			networks  []netbird.Network
			routers   [][]netbird.NetworkRouter
			resources [][]netbird.NetworkResource
		)
		g, gctx := errgroup.WithContext(ctx)
		g.Go(func() (err error) {
			peer, err = c.GetPeer(gctx, in.PeerID)
			return err
		})
		g.Go(func() (err error) {
			policies, err = c.ListPolicies(gctx)
			return err
		})
		g.Go(func() (err error) {
			routes, err = c.ListRoutes(gctx)
			return err
		})
		g.Go(func() (err error) {
			networks, routers, resources, err = fetchNetworks(gctx, c)
			return err
		})
		if err := g.Wait(); err != nil {
			return nil, peerAccessOutput{}, err
		}
		return nil, toPeerAccessOutput(peer, policies, routes, networks, routers, resources), nil
	}
	return tool, handler
}
