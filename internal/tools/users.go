package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/koinsaari/netbird-mcp/internal/netbird"
)

type userAPI interface {
	ListUsers(ctx context.Context, params netbird.ListUsersParams) ([]netbird.User, error)
}

type listUsersInput struct {
	ServiceUser *bool `json:"service_user,omitempty" jsonschema:"true lists only service users, false only regular users; omit for all users"`
}

type listUsersOutput struct {
	Users []userOutput `json:"users"`
}

// userOutput leaves out permissions, idp_id and issued.
type userOutput struct {
	ID              string   `json:"id"`
	Email           string   `json:"email"`
	Name            string   `json:"name"`
	Role            string   `json:"role"`
	Status          string   `json:"status"`
	LastLogin       string   `json:"last_login,omitempty"`
	AutoGroups      []string `json:"auto_groups"`
	IsServiceUser   bool     `json:"is_service_user"`
	IsBlocked       bool     `json:"is_blocked"`
	PendingApproval bool     `json:"pending_approval"`
	IsCurrent       bool     `json:"is_current,omitempty"`
}

// listUsers builds the list_users tool definition and its handler.
func listUsers(c userAPI) (*mcp.Tool, mcp.ToolHandlerFor[listUsersInput, listUsersOutput]) {
	tool := &mcp.Tool{
		Name:        "list_users",
		Description: "Lists users in the NetBird account with role, status and auto-assigned groups. Set service_user to true to list only service users, or false for only regular users.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true},
	}
	handler := func(ctx context.Context, _ *mcp.CallToolRequest, in listUsersInput) (*mcp.CallToolResult, listUsersOutput, error) {
		users, err := c.ListUsers(ctx, netbird.ListUsersParams{ServiceUser: in.ServiceUser})
		if err != nil {
			return nil, listUsersOutput{}, err
		}
		return nil, toListUsersOutput(users), nil
	}
	return tool, handler
}

func toListUsersOutput(users []netbird.User) listUsersOutput {
	out := listUsersOutput{Users: make([]userOutput, 0, len(users))}
	for _, u := range users {
		groups := u.AutoGroups
		if groups == nil {
			groups = []string{}
		}
		out.Users = append(out.Users, userOutput{
			ID: u.ID, Email: u.Email, Name: u.Name, Role: u.Role, Status: u.Status,
			LastLogin: u.LastLogin, AutoGroups: groups, IsServiceUser: u.IsServiceUser,
			IsBlocked: u.IsBlocked, PendingApproval: u.PendingApproval, IsCurrent: u.IsCurrent,
		})
	}
	return out
}
