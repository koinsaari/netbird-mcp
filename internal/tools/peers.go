package tools

import (
	"context"
	"errors"
	"net/netip"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/koinsaari/netbird-mcp/internal/netbird"
)

var errInvalidIP = errors.New("invalid ip: must be an IP address without zone")

type peerLister interface {
	ListPeers(ctx context.Context, params netbird.ListPeersParams) ([]netbird.Peer, error)
}

type listPeersInput struct {
	Name string `json:"name,omitempty" jsonschema:"filter by peer name"`
	IP   string `json:"ip,omitempty" jsonschema:"filter by peer NetBird IP address"`
}

type listPeersOutput struct {
	Peers []peerOutput `json:"peers"`
}

type peerOutput struct {
	ID               string        `json:"id"`
	Name             string        `json:"name"`
	Hostname         string        `json:"hostname"`
	DNSLabel         string        `json:"dns_label"`
	IP               string        `json:"ip"`
	Connected        bool          `json:"connected"`
	LastSeen         string        `json:"last_seen"` // RFC 3339 string passed through
	OS               string        `json:"os"`
	Version          string        `json:"version"`
	Groups           []groupOutput `json:"groups"`
	UserID           string        `json:"user_id"`
	LoginExpired     bool          `json:"login_expired"`
	ApprovalRequired bool          `json:"approval_required"`
}

type groupOutput struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// listPeers builds the list_peers tool definition and its handler. The handler
// returns raw errors; Register maps them to model-facing messages.
func listPeers(c peerLister) (*mcp.Tool, mcp.ToolHandlerFor[listPeersInput, listPeersOutput]) {
	tool := &mcp.Tool{
		Name:        "list_peers",
		Description: "Lists peers (devices) in the NetBird account, optionally filtered by name or IP address.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true},
	}
	handler := func(ctx context.Context, _ *mcp.CallToolRequest, in listPeersInput) (*mcp.CallToolResult, listPeersOutput, error) {
		params := netbird.ListPeersParams{Name: in.Name}
		if in.IP != "" {
			addr, err := netip.ParseAddr(in.IP)
			if err != nil || addr.Zone() != "" {
				return nil, listPeersOutput{}, errInvalidIP
			}
			params.IP = addr.Unmap().String()
		}
		peers, err := c.ListPeers(ctx, params)
		if err != nil {
			return nil, listPeersOutput{}, err
		}
		return nil, toListPeersOutput(peers), nil
	}
	return tool, handler
}

func toListPeersOutput(peers []netbird.Peer) listPeersOutput {
	out := listPeersOutput{Peers: make([]peerOutput, 0, len(peers))}
	for _, p := range peers {
		out.Peers = append(out.Peers, toPeerOutput(p))
	}
	return out
}

func toPeerOutput(p netbird.Peer) peerOutput {
	groups := make([]groupOutput, 0, len(p.Groups))
	for _, g := range p.Groups {
		groups = append(groups, groupOutput{ID: g.ID, Name: g.Name})
	}
	return peerOutput{
		ID: p.ID, Name: p.Name, Hostname: p.Hostname, DNSLabel: p.DNSLabel, IP: p.IP,
		Connected: p.Connected, LastSeen: p.LastSeen, OS: p.OS, Version: p.Version,
		Groups: groups, UserID: p.UserID, LoginExpired: p.LoginExpired,
		ApprovalRequired: p.ApprovalRequired,
	}
}
