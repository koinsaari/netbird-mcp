package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/koinsaari/netbird-mcp/internal/netbird"
)

type groupAPI interface {
	ListGroups(ctx context.Context, params netbird.ListGroupsParams) ([]netbird.Group, error)
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
