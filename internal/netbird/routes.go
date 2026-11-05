package netbird

import (
	"context"
	"fmt"
)

// Route is a NetBird network route, decoded from the API Route schema. Peer is
// empty when the route uses peer groups, and the optional arrays are nil when
// the API omits them.
type Route struct {
	ID                  string   `json:"id"`
	NetworkType         string   `json:"network_type"`
	Description         string   `json:"description"`
	NetworkID           string   `json:"network_id"`
	Enabled             bool     `json:"enabled"`
	Peer                string   `json:"peer"`
	PeerGroups          []string `json:"peer_groups"`
	Network             string   `json:"network"`
	Domains             []string `json:"domains"`
	Metric              int      `json:"metric"`
	Masquerade          bool     `json:"masquerade"`
	Groups              []string `json:"groups"`
	KeepRoute           bool     `json:"keep_route"`
	AccessControlGroups []string `json:"access_control_groups"`
	SkipAutoApply       bool     `json:"skip_auto_apply"`
}

// ListRoutes returns all network routes of the account.
func (c *Client) ListRoutes(ctx context.Context) ([]Route, error) {
	var routes []Route
	if err := c.get(ctx, &routes, nil, "routes"); err != nil {
		return nil, fmt.Errorf("list routes: %w", err)
	}
	return routes, nil
}
