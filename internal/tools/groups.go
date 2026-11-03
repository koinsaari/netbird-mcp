package tools

import (
	"context"
	"errors"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/koinsaari/netbird-mcp/internal/netbird"
)

var errInvalidGroupID = errors.New("invalid group_id: must be a 20-character NetBird group ID")

type groupAPI interface {
	ListGroups(ctx context.Context, params netbird.ListGroupsParams) ([]netbird.Group, error)
	GetGroup(ctx context.Context, id string) (netbird.Group, error)
}

type listGroupsInput struct {
	Name string `json:"name,omitempty" jsonschema:"filter by exact group name"`
}

type listGroupsOutput struct {
	Groups []groupListOutput `json:"groups"`
}

// groupListOutput leaves out the peer and resource lists because the All
// group lists every peer.
type groupListOutput struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	PeersCount     int    `json:"peers_count"`
	ResourcesCount int    `json:"resources_count"`
	Issued         string `json:"issued,omitempty"`
}

// listGroups builds the list_groups tool definition and its handler.
func listGroups(c groupAPI) (*mcp.Tool, mcp.ToolHandlerFor[listGroupsInput, listGroupsOutput]) {
	tool := &mcp.Tool{
		Name:        "list_groups",
		Description: "Lists groups in the NetBird account with peer and resource counts, optionally filtered by exact name.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true},
	}
	handler := func(ctx context.Context, _ *mcp.CallToolRequest, in listGroupsInput) (*mcp.CallToolResult, listGroupsOutput, error) {
		groups, err := c.ListGroups(ctx, netbird.ListGroupsParams{Name: in.Name})
		if err != nil {
			return nil, listGroupsOutput{}, err
		}
		return nil, toListGroupsOutput(groups), nil
	}
	return tool, handler
}

func toListGroupsOutput(groups []netbird.Group) listGroupsOutput {
	out := listGroupsOutput{Groups: make([]groupListOutput, 0, len(groups))}
	for _, g := range groups {
		out.Groups = append(out.Groups, groupListOutput{
			ID: g.ID, Name: g.Name, PeersCount: g.PeersCount, ResourcesCount: g.ResourcesCount, Issued: g.Issued,
		})
	}
	return out
}

type getGroupInput struct {
	GroupID string `json:"group_id" jsonschema:"the 20-character NetBird group ID, as returned by list_groups"`
}

type groupDetailsOutput struct {
	ID             string                `json:"id"`
	Name           string                `json:"name"`
	PeersCount     int                   `json:"peers_count"`
	ResourcesCount int                   `json:"resources_count"`
	Issued         string                `json:"issued,omitempty"`
	Peers          []groupMemberOutput   `json:"peers"`
	Resources      []groupResourceOutput `json:"resources"`
}

type groupMemberOutput struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type groupResourceOutput struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

// getGroup builds the get_group tool definition and its handler. The handler
// validates the ID before calling the client and returns raw errors.
func getGroup(c groupAPI) (*mcp.Tool, mcp.ToolHandlerFor[getGroupInput, groupDetailsOutput]) {
	tool := &mcp.Tool{
		Name:        "get_group",
		Description: "Gets one group in the NetBird account by its ID, including its member peers and resources. Use list_groups to find the ID.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true},
	}
	handler := func(ctx context.Context, _ *mcp.CallToolRequest, in getGroupInput) (*mcp.CallToolResult, groupDetailsOutput, error) {
		if !validID(in.GroupID) {
			return nil, groupDetailsOutput{}, errInvalidGroupID
		}
		group, err := c.GetGroup(ctx, in.GroupID)
		if err != nil {
			return nil, groupDetailsOutput{}, err
		}
		return nil, toGroupDetailsOutput(group), nil
	}
	return tool, handler
}

func toGroupDetailsOutput(g netbird.Group) groupDetailsOutput {
	peers := make([]groupMemberOutput, 0, len(g.Peers))
	for _, p := range g.Peers {
		peers = append(peers, groupMemberOutput{ID: p.ID, Name: p.Name})
	}
	resources := make([]groupResourceOutput, 0, len(g.Resources))
	for _, r := range g.Resources {
		resources = append(resources, groupResourceOutput{ID: r.ID, Type: r.Type})
	}
	return groupDetailsOutput{
		ID: g.ID, Name: g.Name, PeersCount: g.PeersCount, ResourcesCount: g.ResourcesCount, Issued: g.Issued,
		Peers: peers, Resources: resources,
	}
}
