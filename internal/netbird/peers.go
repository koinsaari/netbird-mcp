package netbird

import (
	"context"
	"fmt"
	"net/url"
)

// Peer is a NetBird peer, decoded with only the fields this server uses.
type Peer struct {
	ID               string  `json:"id"`
	Name             string  `json:"name"`
	Hostname         string  `json:"hostname"`
	DNSLabel         string  `json:"dns_label"`
	IP               string  `json:"ip"`
	Connected        bool    `json:"connected"`
	LastSeen         string  `json:"last_seen"`
	OS               string  `json:"os"`
	Version          string  `json:"version"`
	Groups           []Group `json:"groups"`
	UserID           string  `json:"user_id"`
	LoginExpired     bool    `json:"login_expired"`
	ApprovalRequired bool    `json:"approval_required"`
}

// Group is the minimal group reference embedded in other resources.
type Group struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ListPeersParams filters ListPeers. Empty fields are not sent.
type ListPeersParams struct {
	Name string
	IP   string
}

// ListPeers returns all peers matching params.
func (c *Client) ListPeers(ctx context.Context, params ListPeersParams) ([]Peer, error) {
	query := url.Values{}
	if params.Name != "" {
		query.Set("name", params.Name)
	}
	if params.IP != "" {
		query.Set("ip", params.IP)
	}

	var peers []Peer
	if err := c.get(ctx, &peers, query, "peers"); err != nil {
		return nil, fmt.Errorf("list peers: %w", err)
	}
	return peers, nil
}
