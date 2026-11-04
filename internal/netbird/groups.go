package netbird

import (
	"context"
	"fmt"
	"net/url"
)

// Group is a NetBird group, decoded from the API Group schema. Peers and
// Resources are nil when the group is embedded in another resource.
type Group struct {
	ID             string        `json:"id"`
	Name           string        `json:"name"`
	PeersCount     int           `json:"peers_count"`
	ResourcesCount int           `json:"resources_count"`
	Issued         string        `json:"issued"`
	Peers          []PeerMinimum `json:"peers"`
	Resources      []Resource    `json:"resources"`
}

// PeerMinimum is the peer reference listed in a group.
type PeerMinimum struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Resource is the resource reference listed in a group.
type Resource struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

// ListGroupsParams filters ListGroups. Empty fields are not sent.
type ListGroupsParams struct {
	Name string
}

// ListGroups returns all groups matching params.
func (c *Client) ListGroups(ctx context.Context, params ListGroupsParams) ([]Group, error) {
	query := url.Values{}
	if params.Name != "" {
		query.Set("name", params.Name)
	}

	var groups []Group
	if err := c.get(ctx, &groups, query, "groups"); err != nil {
		return nil, fmt.Errorf("list groups: %w", err)
	}
	return groups, nil
}

// GetGroup returns the group with the given ID. The caller validates id.
func (c *Client) GetGroup(ctx context.Context, id string) (Group, error) {
	var group Group
	if err := c.get(ctx, &group, nil, "groups", id); err != nil {
		return Group{}, fmt.Errorf("get group: %w", err)
	}
	return group, nil
}
