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
}

type getPeerAccessInput struct {
	PeerID string `json:"peer_id" jsonschema:"the 20-character NetBird peer ID, as returned by list_peers"`
}

// getPeerAccess builds the get_peer_access tool definition and its handler.
// The handler validates the ID, fetches the peer, policies and routes
// concurrently and joins them. Any failing call fails the tool with that
// call's own error.
func getPeerAccess(c peerAccessAPI) (*mcp.Tool, mcp.ToolHandlerFor[getPeerAccessInput, peerAccessOutput]) {
	tool := &mcp.Tool{
		Name:        "get_peer_access",
		Description: "Describes one NetBird peer's access: its groups, the access policies whose rules apply to it (with direction and the other side) and the network routes it is part of.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true},
	}
	handler := func(ctx context.Context, _ *mcp.CallToolRequest, in getPeerAccessInput) (*mcp.CallToolResult, peerAccessOutput, error) {
		if !validID(in.PeerID) {
			return nil, peerAccessOutput{}, errInvalidPeerID
		}
		var (
			peer     netbird.Peer
			policies []netbird.Policy
			routes   []netbird.Route
		)
		// errgroup keeps the first error, and a sibling only sees
		// context.Canceled after that error is recorded, so it can't replace it.
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
		if err := g.Wait(); err != nil {
			return nil, peerAccessOutput{}, err
		}
		return nil, toPeerAccessOutput(peer, policies, routes), nil
	}
	return tool, handler
}
