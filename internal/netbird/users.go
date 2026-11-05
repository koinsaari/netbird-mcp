package netbird

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// User is a NetBird user, decoded from the API User schema. The API password
// field is create-only and left out so it can't reach tool output, and permissions
// is not decoded either since it's a large nested map.
type User struct {
	ID              string   `json:"id"`
	Email           string   `json:"email"`
	Name            string   `json:"name"`
	Role            string   `json:"role"`
	Status          string   `json:"status"`
	LastLogin       string   `json:"last_login"`
	AutoGroups      []string `json:"auto_groups"`
	IsCurrent       bool     `json:"is_current"`
	IsServiceUser   bool     `json:"is_service_user"`
	IsBlocked       bool     `json:"is_blocked"`
	PendingApproval bool     `json:"pending_approval"`
	Issued          string   `json:"issued"`
	IdpID           string   `json:"idp_id"`
}

// ListUsersParams filters ListUsers. A nil ServiceUser is not sent, so the
// API decides.
type ListUsersParams struct {
	ServiceUser *bool
}

// ListUsers returns all users matching params.
func (c *Client) ListUsers(ctx context.Context, params ListUsersParams) ([]User, error) {
	query := url.Values{}
	if params.ServiceUser != nil {
		query.Set("service_user", strconv.FormatBool(*params.ServiceUser))
	}

	var users []User
	if err := c.get(ctx, &users, query, "users"); err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	return users, nil
}
