package netbird

import (
	"context"
	"fmt"
)

type Network struct {
	ID                string   `json:"id"`
	Name              string   `json:"name"`
	Description       string   `json:"description"`
	Routers           []string `json:"routers"`
	RoutingPeersCount int      `json:"routing_peers_count"`
	Resources         []string `json:"resources"`
	Policies          []string `json:"policies"`
}

type NetworkRouter struct {
	ID         string   `json:"id"`
	Peer       string   `json:"peer"`
	PeerGroups []string `json:"peer_groups"`
	Metric     int      `json:"metric"`
	Masquerade bool     `json:"masquerade"`
	Enabled    bool     `json:"enabled"`
}

type NetworkResource struct {
	ID          string  `json:"id"`
	Type        string  `json:"type"`
	Groups      []Group `json:"groups"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Address     string  `json:"address"`
	Enabled     bool    `json:"enabled"`
}

func (c *Client) ListNetworks(ctx context.Context) ([]Network, error) {
	var networks []Network
	if err := c.get(ctx, &networks, nil, "networks"); err != nil {
		return nil, fmt.Errorf("list networks: %w", err)
	}
	return networks, nil
}

func (c *Client) ListNetworkRouters(ctx context.Context, networkID string) ([]NetworkRouter, error) {
	var routers []NetworkRouter
	if err := c.get(ctx, &routers, nil, "networks", networkID, "routers"); err != nil {
		return nil, fmt.Errorf("list network routers: %w", err)
	}
	return routers, nil
}

func (c *Client) ListNetworkResources(ctx context.Context, networkID string) ([]NetworkResource, error) {
	var resources []NetworkResource
	if err := c.get(ctx, &resources, nil, "networks", networkID, "resources"); err != nil {
		return nil, fmt.Errorf("list network resources: %w", err)
	}
	return resources, nil
}
