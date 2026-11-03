package netbird

import (
	"context"
	"fmt"
)

// Policy is a NetBird access control policy, decoded from the API Policy
// schema.
type Policy struct {
	ID                  string       `json:"id"`
	Name                string       `json:"name"`
	Description         string       `json:"description"`
	Enabled             bool         `json:"enabled"`
	SourcePostureChecks []string     `json:"source_posture_checks"`
	Rules               []PolicyRule `json:"rules"`
}

// PolicyRule is a rule of a policy, decoded from the API PolicyRule schema.
// The resource fields are nil when the rule uses groups only, and
// AuthorizedGroups is nil when the API omits it.
type PolicyRule struct {
	ID                  string              `json:"id"`
	Name                string              `json:"name"`
	Description         string              `json:"description"`
	Enabled             bool                `json:"enabled"`
	Action              string              `json:"action"`
	Bidirectional       bool                `json:"bidirectional"`
	Protocol            string              `json:"protocol"`
	Ports               []string            `json:"ports"`
	PortRanges          []RulePortRange     `json:"port_ranges"`
	AuthorizedGroups    map[string][]string `json:"authorized_groups"`
	Sources             []Group             `json:"sources"`
	SourceResource      *Resource           `json:"sourceResource"`
	Destinations        []Group             `json:"destinations"`
	DestinationResource *Resource           `json:"destinationResource"`
}

// RulePortRange is a port range of a policy rule. Equal Start and End mean a
// single port.
type RulePortRange struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

// ListPolicies returns all policies of the account.
func (c *Client) ListPolicies(ctx context.Context) ([]Policy, error) {
	var policies []Policy
	if err := c.get(ctx, &policies, nil, "policies"); err != nil {
		return nil, fmt.Errorf("list policies: %w", err)
	}
	return policies, nil
}
